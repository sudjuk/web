package repository

import (
    "errors"
    "fmt"
    "strings"

    "gorm.io/gorm"
)

// GetDays returns active services (days)
func (r *Repository) GetDays() ([]Day, error) {
    var rows []astroDay
    if err := r.db.Where("is_deleted = ?", false).Order("id").Find(&rows).Error; err != nil {
        return nil, err
    }
    if len(rows) == 0 {
        return nil, fmt.Errorf("records not found")
    }
    result := make([]Day, 0, len(rows))
    for _, d := range rows {
        result = append(result, Day{
            ID:          int(d.ID),
            Date:        d.Name,
            FullInfo:    d.Description,
            Image:       d.ImageURL,
            BodiesText:  d.BodiesText,
            EarthRA:     d.EarthRA,
            EarthDEC:    d.EarthDEC,
        })
    }
    return result, nil
}

func (r *Repository) GetDay(id int) (Day, error) {
    var d astroDay
    if err := r.db.Where("id = ? AND is_deleted = ?", id, false).First(&d).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return Day{}, ErrDayNotFound
        }
        return Day{}, err
    }
    return Day{
        ID:          int(d.ID),
        Date:        d.Name,
        FullInfo:    d.Description,
        Image:       d.ImageURL,
        BodiesText:  d.BodiesText,
        EarthRA:     d.EarthRA,
        EarthDEC:    d.EarthDEC,
    }, nil
}

func (r *Repository) GetDaysByDate(date string) ([]Day, error) {
    var rows []astroDay
    q := strings.ToLower(date)
    if err := r.db.Where("is_deleted = ? AND LOWER(name) LIKE ?", false, "%"+q+"%").Order("id").Find(&rows).Error; err != nil {
        return nil, err
    }
    result := make([]Day, 0, len(rows))
    for _, d := range rows {
        result = append(result, Day{
            ID:          int(d.ID),
            Date:        d.Name,
            FullInfo:    d.Description,
            Image:       d.ImageURL,
            BodiesText:  d.BodiesText,
            EarthRA:     d.EarthRA,
            EarthDEC:    d.EarthDEC,
        })
    }
    return result, nil
}

// GetServiceImageURL returns current image_url for the service
func (r *Repository) GetServiceImageURL(id int) (string, error) {
    var d astroDay
    if err := r.db.Select("image_url").Where("id = ?", id).First(&d).Error; err != nil {
        return "", err
    }
    return d.ImageURL, nil
}

// UpdateServiceImageURL updates image_url for the service
func (r *Repository) UpdateServiceImageURL(id int, url string) error {
    res := r.db.Model(&astroDay{}).Where("id = ? AND is_deleted = ?", id, false).Update("image_url", url)
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}

// CreateService creates a new service (astro_day) record
func (r *Repository) CreateService(name, description, bodiesText string, earthRA, earthDEC float64) (int, error) {
    row := astroDay{
        Name:        strings.TrimSpace(name),
        Description: strings.TrimSpace(description),
        BodiesText:  strings.TrimSpace(bodiesText),
        EarthRA:     earthRA,
        EarthDEC:    earthDEC,
        IsDeleted:   false,
        ImageURL:    "",
    }
    if err := r.db.Create(&row).Error; err != nil {
        return 0, err
    }
    return int(row.ID), nil
}

// UpdateService updates allowed fields only
func (r *Repository) UpdateService(id int, name, description, bodiesText *string, earthRA, earthDEC *float64) error {
    updates := map[string]interface{}{}
    if name != nil {
        updates["name"] = strings.TrimSpace(*name)
    }
    if description != nil {
        updates["description"] = strings.TrimSpace(*description)
    }
    if bodiesText != nil {
        updates["bodies_text"] = strings.TrimSpace(*bodiesText)
    }
    if earthRA != nil {
        updates["earth_ra"] = *earthRA
    }
    if earthDEC != nil {
        updates["earth_dec"] = *earthDEC
    }
    if len(updates) == 0 {
        return nil
    }
    res := r.db.Model(&astroDay{}).Where("id = ? AND is_deleted = ?", id, false).Updates(updates)
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}

// SoftDeleteService marks service deleted and clears image_url (actual MinIO deletion handled in handler)
func (r *Repository) SoftDeleteService(id int) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        var d astroDay
        if err := tx.Where("id = ? AND is_deleted = ?", id, false).First(&d).Error; err != nil {
            return err
        }
        if err := tx.Model(&astroDay{}).Where("id = ?", id).Updates(map[string]interface{}{
            "is_deleted": true,
            "image_url":  "",
        }).Error; err != nil {
            return err
        }
        return nil
    })
}


