package repository

import (
    "errors"
    "strings"

    "gorm.io/gorm"
)

// IsObservationDeleted returns true if observation has status 'deleted'
func (r *Repository) IsObservationDeleted(id int) (bool, error) {
    var o observation
    if err := r.db.Select("status").Where("id = ?", id).First(&o).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return true, nil
        }
        return false, err
    }
    return strings.ToLower(o.Status) == "deleted", nil
}

// CountDraftItems returns observationID (0 if none) and items count for user's draft
func (r *Repository) CountDraftItems(userID int) (int, int, error) {
    var obs observation
    if err := r.db.Where("creator_id = ? AND status = ?", userID, "draft").First(&obs).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return 0, 0, nil
        }
        return 0, 0, err
    }
    var cnt int64
    if err := r.db.Table("observation_items").Where("observation_id = ?", obs.ID).Count(&cnt).Error; err != nil {
        return 0, 0, err
    }
    return int(obs.ID), int(cnt), nil
}

// AddServiceToDraft creates draft if missing and upserts item
func (r *Repository) AddServiceToDraft(userID int, dayID int) (int, error) {
    tx := r.db.Begin()
    if tx.Error != nil {
        return 0, tx.Error
    }
    var obs observation
    err := tx.Where("creator_id = ? AND status = ?", userID, "draft").First(&obs).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        obs = observation{Status: "draft", CreatorID: int64(userID)}
        if err := tx.Create(&obs).Error; err != nil {
            tx.Rollback()
            return 0, err
        }
    } else if err != nil {
        tx.Rollback()
        return 0, err
    }
    res := tx.Model(&observationItem{}).
        Where("observation_id = ? AND day_id = ?", obs.ID, dayID).
        Updates(map[string]interface{}{"quantity": gorm.Expr("quantity + 1")})
    if res.Error != nil {
        tx.Rollback()
        return 0, res.Error
    }
    if res.RowsAffected == 0 {
        item := observationItem{ObservationID: obs.ID, DayID: int64(dayID), Quantity: 1, IsPrimary: false, SortOrder: 0, AsteroidRA: 0, AsteroidDEC: 0}
        if err := tx.Create(&item).Error; err != nil {
            tx.Rollback()
            return 0, err
        }
    }
    if err := tx.Commit().Error; err != nil {
        return 0, err
    }
    return int(obs.ID), nil
}

// SoftDeleteObservationRaw deletes (sets status) via raw SQL
func (r *Repository) SoftDeleteObservationRaw(userID int, observationID int) error {
    res := r.db.Exec("UPDATE observations SET status = 'deleted' WHERE id = ? AND creator_id = ? AND status <> 'deleted'", observationID, userID)
    return res.Error
}


