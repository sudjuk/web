package repository

import (
    "errors"
    "fmt"
    "math"
    "sort"
    "time"
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


// ListObservations returns observations excluding deleted and draft, optionally filtered by status and formed date range
func (r *Repository) ListObservations(status *string, formedFrom, formedTo *time.Time) ([]struct{
    ID int64
    Status string
    CreatorLogin string
    ModeratorLogin *string
    Comment *string
    CalculatedKM *float64
    FormedAt *time.Time
}, error) {
    q := r.db.Table("observations o").
        Select("o.id, o.status, uc.login as creator_login, um.login as moderator_login, o.comment, o.calculated_km, o.formed_at").
        Joins("JOIN users uc ON uc.id = o.creator_id").
        Joins("LEFT JOIN users um ON um.id = o.moderator_id").
        Where("LOWER(o.status) NOT IN (?)", []string{"deleted", "draft"})
    if status != nil && *status != "" {
        q = q.Where("LOWER(o.status) = ?", strings.ToLower(*status))
    }
    if formedFrom != nil { q = q.Where("o.formed_at >= ?", *formedFrom) }
    if formedTo != nil { q = q.Where("o.formed_at <= ?", *formedTo) }
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

// UpdateObservation updates allowed user fields (comment)
func (r *Repository) UpdateObservation(userID int, id int, comment *string) error {
    // restrict by creator to simplify authorization
    updates := map[string]interface{}{}
    if comment != nil {
        updates["comment"] = *comment
    }
    if len(updates) == 0 {
        return nil
    }
    res := r.db.Model(&observation{}).
        Where("id = ? AND creator_id = ? AND status NOT IN ('deleted')", id, userID).
        Updates(updates)
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}

// SubmitObservation transitions draft -> formed
func (r *Repository) SubmitObservation(userID int, id int) error {
    now := time.Now()
    res := r.db.Model(&observation{}).
        Where("id = ? AND creator_id = ? AND status = 'draft'", id, userID).
        Updates(map[string]interface{}{"status": "formed", "formed_at": now})
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return fmt.Errorf("invalid state or not found")
    }
    return nil
}

// CompleteOrReject sets status and calculates CalculatedKM on complete
func (r *Repository) CompleteOrReject(moderatorID int, id int, complete bool) error {
    if complete {
        // Gather asteroid RA/DEC and day name (date string)
        type rowRaw struct {
            AstRA   float64
            AstDEC  float64
            DayName string
        }
        var raw []rowRaw
        if err := r.db.Table("observation_items oi").
            Select("oi.asteroid_ra as ast_ra, oi.asteroid_dec as ast_dec, ad.name as day_name").
            Joins("JOIN astro_days ad ON ad.id = oi.day_id").
            Where("oi.observation_id = ?", id).
            Scan(&raw).Error; err != nil {
            return err
        }

        // Parse dates and sort by time
        type point struct {
            ra  float64
            dec float64
            t   time.Time
        }
        points := make([]point, 0, len(raw))
        for _, rr := range raw {
            name := strings.TrimSpace(rr.DayName)
            if name == "" { continue }
            // try DD.MM.YYYY then ISO
            tt, err := time.Parse("02.01.2006", name)
            if err != nil {
                if t2, err2 := time.Parse("2006-01-02", name); err2 == nil {
                    tt = t2
                } else {
                    continue
                }
            }
            points = append(points, point{ra: rr.AstRA, dec: rr.AstDEC, t: tt})
        }
        if len(points) < 2 {
            // Not enough data; finish without distance
            now := time.Now()
            return r.db.Model(&observation{}).Where("id = ? AND status = 'formed'", id).
                Updates(map[string]interface{}{"status": "finished", "calculated_km": 0, "moderator_id": moderatorID, "completed_at": now}).Error
        }
        sort.Slice(points, func(i, j int) bool { return points[i].t.Before(points[j].t) })

        // Build distances (km) for all pairs using distance = V * dt / alpha
        const linearSpeed = 25000.0 // m/s
        distancesKm := make([]float64, 0, len(points)*(len(points)-1)/2)
        for i := 0; i < len(points)-1; i++ {
            for j := i + 1; j < len(points); j++ {
                dt := points[j].t.Sub(points[i].t).Seconds()
                if dt <= 0 { continue }
                alpha := angularDistance(points[i].ra, points[i].dec, points[j].ra, points[j].dec)
                if alpha <= 0 { continue }
                distMeters := linearSpeed * dt / alpha
                if distMeters <= 0 || math.IsInf(distMeters, 0) || math.IsNaN(distMeters) { continue }
                distancesKm = append(distancesKm, distMeters/1000.0)
            }
        }
        calcKm := 0.0
        if len(distancesKm) > 0 {
            calcKm = median(distancesKm)
        }
        now := time.Now()
        if err := r.db.Model(&observation{}).Where("id = ? AND status = 'formed'", id).
            Updates(map[string]interface{}{"status": "finished", "calculated_km": calcKm, "moderator_id": moderatorID, "completed_at": now}).Error; err != nil {
            return err
        }
        return nil
    }
    // reject
    now := time.Now()
    res := r.db.Model(&observation{}).Where("id = ? AND status = 'formed'", id).Updates(map[string]interface{}{"status": "rejected", "moderator_id": moderatorID, "completed_at": now})
    return res.Error
}

// angularDistance computes great-circle distance in radians between (ra1,dec1) and (ra2,dec2) in degrees
func angularDistance(ra1, dec1, ra2, dec2 float64) float64 {
    const degToRad = math.Pi / 180.0
    ra1r := ra1 * degToRad
    dec1r := dec1 * degToRad
    ra2r := ra2 * degToRad
    dec2r := dec2 * degToRad
    dRA := ra2r - ra1r
    dDEC := dec2r - dec1r
    a := math.Sin(dDEC/2)*math.Sin(dDEC/2) + math.Cos(dec1r)*math.Cos(dec2r)*math.Sin(dRA/2)*math.Sin(dRA/2)
    return 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// median returns median of slice (copy-sorts)
func median(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)
    n := len(sorted)
    if n%2 == 1 {
        return sorted[n/2]
    }
    return (sorted[n/2-1] + sorted[n/2]) / 2
}

// DeleteObservationItem deletes an item by observation/day for creator while in editable states
func (r *Repository) DeleteObservationItem(userID int, observationID int, dayID int) error {
    // ensure observation belongs to user and is editable
    var o observation
    if err := r.db.Where("id = ? AND creator_id = ? AND status IN ('draft','formed')", observationID, userID).First(&o).Error; err != nil {
        return err
    }
    res := r.db.Where("observation_id = ? AND day_id = ?", observationID, dayID).Delete(&observationItem{})
    return res.Error
}

// UpdateObservationItem updates fields without using PK m-m
func (r *Repository) UpdateObservationItem(userID int, observationID int, dayID int, quantity, sortOrder *int, isPrimary *bool, note *string, asteroidRA, asteroidDEC *float64) error {
    var o observation
    if err := r.db.Where("id = ? AND creator_id = ? AND status IN ('draft','formed')", observationID, userID).First(&o).Error; err != nil {
        return err
    }
    updates := map[string]interface{}{}
    if quantity != nil { updates["quantity"] = *quantity }
    if sortOrder != nil { updates["sort_order"] = *sortOrder }
    if isPrimary != nil { updates["is_primary"] = *isPrimary }
    if note != nil { updates["note"] = *note }
    if asteroidRA != nil { updates["asteroid_ra"] = *asteroidRA }
    if asteroidDEC != nil { updates["asteroid_dec"] = *asteroidDEC }
    if len(updates) == 0 { return nil }
    res := r.db.Model(&observationItem{}).Where("observation_id = ? AND day_id = ?", observationID, dayID).Updates(updates)
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}


