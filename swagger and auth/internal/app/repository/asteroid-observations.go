package repository

import (
    "errors"
    "strings"
    "time"

    "gorm.io/gorm"
)

// IsAsteroidObservationDeleted returns true if asteroid observation has status 'deleted'
func (r *Repository) IsAsteroidObservationDeleted(id int) (bool, error) {
    var ar asteroidObservation
    if err := r.db.Select("status").Where("id = ?", id).First(&ar).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return true, nil
        }
        return false, err
    }
    return strings.ToLower(ar.Status) == "deleted", nil
}

// CountDraftItems returns asteroidObservationID (0 if none) and items count for user's draft
func (r *Repository) CountDraftItems(userID int) (int, int, error) {
    var obs asteroidObservation
    if err := r.db.Where("creator_id = ? AND status = ?", userID, "draft").First(&obs).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return 0, 0, nil
        }
        return 0, 0, err
    }
    var cnt int64
    if err := r.db.Table("asteroid_observation_items").Where("observation_id = ?", obs.ID).Count(&cnt).Error; err != nil {
        return 0, 0, err
    }
    return int(obs.ID), int(cnt), nil
}

// AddServiceToDraft adds day to user's draft observation
func (r *Repository) AddServiceToDraft(userID int, dayID int) (int, error) {
    tx := r.db.Begin()
    if tx.Error != nil {
        return 0, tx.Error
    }
    var asteroidObs asteroidObservation
    err := tx.Where("creator_id = ? AND status = ?", userID, "draft").First(&asteroidObs).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        asteroidObs = asteroidObservation{Status: "draft", CreatorID: int64(userID)}
        if err := tx.Create(&asteroidObs).Error; err != nil {
            tx.Rollback()
            return 0, err
        }
    } else if err != nil {
        tx.Rollback()
        return 0, err
    }
    res := tx.Model(&asteroidObservationItem{}).
        Where("observation_id = ? AND day_id = ?", asteroidObs.ID, dayID).
        Updates(map[string]interface{}{"asteroid_ra": 0, "asteroid_dec": 0})
    if res.Error != nil {
        tx.Rollback()
        return 0, res.Error
    }
    if res.RowsAffected == 0 {
        item := asteroidObservationItem{AsteroidObservationID: asteroidObs.ID, DayID: int64(dayID), AsteroidRA: 0, AsteroidDEC: 0}
        if err := tx.Create(&item).Error; err != nil {
            tx.Rollback()
            return 0, err
        }
    }
    if err := tx.Commit().Error; err != nil {
        return 0, err
    }
    return int(asteroidObs.ID), nil
}

// SoftDeleteAsteroidObservationRaw deletes (sets status) via raw SQL
func (r *Repository) SoftDeleteAsteroidObservationRaw(userID int, asteroidObservationID int) error {
    res := r.db.Exec("UPDATE asteroid_observations SET status = 'deleted' WHERE id = ? AND creator_id = ? AND status <> 'deleted'", asteroidObservationID, userID)
    return res.Error
}


// ListAsteroidObservations returns asteroid observations excluding deleted and draft, optionally filtered by status, formed date range, and creator
func (r *Repository) ListAsteroidObservations(status *string, formedFrom, formedTo *time.Time, creatorID *int64) ([]struct{
    ID int64
    Status string
    CreatorLogin string
    ModeratorLogin *string
    Comment *string
    CalculatedKM *float64
    FormedAt *time.Time
}, error) {
    q := r.db.Table("asteroid_observations o").
        Select("o.id, o.status, uc.login as creator_login, um.login as moderator_login, o.comment, o.calculated_km, o.formed_at").
        Joins("JOIN users uc ON uc.id = o.creator_id").
        Joins("LEFT JOIN users um ON um.id = o.moderator_id").
        Where("LOWER(o.status) NOT IN (?)", []string{"deleted", "draft"})
    if status != nil && *status != "" {
        q = q.Where("LOWER(o.status) = ?", strings.ToLower(*status))
    }
    if formedFrom != nil { q = q.Where("o.formed_at >= ?", *formedFrom) }
    if formedTo != nil { q = q.Where("o.formed_at <= ?", *formedTo) }
    if creatorID != nil { q = q.Where("o.creator_id = ?", *creatorID) }
    var rows []struct{
        ID int64
        Status string
        CreatorLogin string
        ModeratorLogin *string
        Comment *string
        CalculatedKM *float64
        FormedAt *time.Time
    }
    if err := q.Order("o.id DESC").Scan(&rows).Error; err != nil {
        return nil, err
    }
    return rows, nil
}

// UpdateAsteroidObservation updates observation comment (only by creator)
func (r *Repository) UpdateAsteroidObservation(userID int, id int, comment *string) error {
    updates := map[string]interface{}{}
    if comment != nil { updates["comment"] = *comment }
    if len(updates) == 0 { return nil }
    res := r.db.Model(&asteroidObservation{}).
        Where("id = ? AND creator_id = ? AND status IN ('draft','formed')", id, userID).
        Updates(updates)
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}

// SubmitAsteroidObservation changes status from draft to formed
func (r *Repository) SubmitAsteroidObservation(userID int, id int) error {
    now := time.Now()
    res := r.db.Model(&asteroidObservation{}).
        Where("id = ? AND creator_id = ? AND status = 'draft'", id, userID).
        Updates(map[string]interface{}{"status": "formed", "formed_at": now})
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}

// CompleteOrReject changes status to completed/rejected (only by moderator)
func (r *Repository) CompleteOrReject(moderatorID int, id int, approved bool, comment *string) error {
    now := time.Now()
    var status string
    if approved {
        status = "completed"
    } else {
        status = "rejected"
    }
    
    updates := map[string]interface{}{
        "status": status,
        "moderator_id": moderatorID,
        "completed_at": now,
    }
    if comment != nil { updates["comment"] = *comment }
    
    // First check if observation exists and is in correct status
    var count int64
    if err := r.db.Model(&asteroidObservation{}).Where("id = ? AND status = 'formed'", id).Count(&count).Error; err != nil {
        return err
    }
    if count == 0 {
        return gorm.ErrRecordNotFound
    }
    
    // Update the observation
    if err := r.db.Model(&asteroidObservation{}).Where("id = ? AND status = 'formed'", id).Updates(updates).Error; err != nil {
        return err
    }
    
    // If approved, calculate and update calculated_km
    if approved {
        // This would need actual calculation logic
        // For now, just set a placeholder value
        if err := r.db.Model(&asteroidObservation{}).Where("id = ?", id).Update("calculated_km", 0.0).Error; err != nil {
            return err
        }
    }
    
    return nil
}

// DeleteAsteroidObservationItem removes item from observation (only by creator)
func (r *Repository) DeleteAsteroidObservationItem(userID int, asteroidObservationID int, dayID int) error {
    // ensure asteroidObservation belongs to user and is editable
    var o asteroidObservation
    if err := r.db.Where("id = ? AND creator_id = ? AND status IN ('draft','formed')", asteroidObservationID, userID).First(&o).Error; err != nil {
        return err
    }
    res := r.db.Where("observation_id = ? AND day_id = ?", asteroidObservationID, dayID).Delete(&asteroidObservationItem{})
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}

// UpdateAsteroidObservationItem updates asteroid coordinates in observation item (only by creator)
func (r *Repository) UpdateAsteroidObservationItem(userID int, asteroidObservationID int, dayID int, asteroidRA, asteroidDEC *float64) error {
    var o asteroidObservation
    if err := r.db.Where("id = ? AND creator_id = ? AND status IN ('draft','formed')", asteroidObservationID, userID).First(&o).Error; err != nil {
        return err
    }
    updates := map[string]interface{}{}
    if asteroidRA != nil { updates["asteroid_ra"] = *asteroidRA }
    if asteroidDEC != nil { updates["asteroid_dec"] = *asteroidDEC }
    if len(updates) == 0 { return nil }
    res := r.db.Model(&asteroidObservationItem{}).Where("observation_id = ? AND day_id = ?", asteroidObservationID, dayID).Updates(updates)
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}
