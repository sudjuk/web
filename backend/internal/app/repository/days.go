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
            Description: d.Description,
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
        Description: d.Description,
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
            Description: d.Description,
            FullInfo:    d.Description,
            Image:       d.ImageURL,
            BodiesText:  d.BodiesText,
            EarthRA:     d.EarthRA,
            EarthDEC:    d.EarthDEC,
        })
    }
    return result, nil
}


