package repository

import (
    "errors"
    "gorm.io/gorm"
)

type Repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) (*Repository, error) {
    if db == nil {
        return nil, errors.New("nil gorm DB passed to repository")
    }
    return &Repository{db: db}, nil
}

var ErrDayNotFound = errors.New("day not found")

// Public DTOs used by handlers (kept for compatibility)
type Observation struct {
    ID_observation int
    Description    string
    Result         float64
}

type Day struct {
    ID          int
    Date        string
    Description string
    FullInfo    string
    Image       string
    EarthRA     float64
    EarthDEC    float64
    BodiesText  string
    AsteroidRA  float64
    AsteroidDEC float64
}

// GORM models mapping DB tables
type user struct {
    ID          int64  `gorm:"column:id;primaryKey"`
    Login       string `gorm:"column:login"`
    Password    string `gorm:"column:password_hash"`
    IsModerator bool   `gorm:"column:is_moderator"`
}

func (user) TableName() string { return "users" }

type astroDay struct {
    ID          int64  `gorm:"column:id;primaryKey"`
    Name        string `gorm:"column:name"`
    Description string `gorm:"column:description"`
    IsDeleted   bool   `gorm:"column:is_deleted"`
    ImageURL    string `gorm:"column:image_url"`
    BodiesText  string `gorm:"column:bodies_text"`
    EarthRA     float64 `gorm:"column:earth_ra"`
    EarthDEC    float64 `gorm:"column:earth_dec"`
}

func (astroDay) TableName() string { return "astro_days" }

type observation struct {
    ID           int64   `gorm:"column:id;primaryKey"`
    Status       string  `gorm:"column:status"`
    CreatorID    int64   `gorm:"column:creator_id"`
    Comment      *string `gorm:"column:comment"`
    CalculatedKM *float64 `gorm:"column:calculated_km"`
}

func (observation) TableName() string { return "observations" }

type observationItem struct {
    ObservationID int64   `gorm:"column:observation_id;primaryKey"`
    DayID         int64   `gorm:"column:day_id;primaryKey"`
    Quantity      int     `gorm:"column:quantity"`
    IsPrimary     bool    `gorm:"column:is_primary"`
    SortOrder     int     `gorm:"column:sort_order"`
    Note          *string `gorm:"column:note"`
    AsteroidRA    float64 `gorm:"column:asteroid_ra"`
    AsteroidDEC   float64 `gorm:"column:asteroid_dec"`
}

func (observationItem) TableName() string { return "observation_items" }

// GetObservationDays возвращает список дней заявки с координатами из м-м
func (r *Repository) GetObservationDays(observationID int) ([]Day, error) {
    type row struct {
        DayID       int64
        Name        string
        Description string
        ImageURL    string
        BodiesText  string
        AstRA       float64
        AstDEC      float64
        EarthRA     float64
        EarthDEC    float64
    }
    var rows []row
    if err := r.db.Table("observation_items oi").
        Select("ad.id as day_id, ad.name, ad.description, ad.image_url, ad.bodies_text, oi.asteroid_ra as ast_ra, oi.asteroid_dec as ast_dec, ad.earth_ra, ad.earth_dec").
        Joins("JOIN astro_days ad ON ad.id = oi.day_id").
        Where("oi.observation_id = ?", observationID).
        Order("oi.sort_order, ad.id").
        Scan(&rows).Error; err != nil {
        return nil, err
    }
    result := make([]Day, 0, len(rows))
    for _, rrow := range rows {
        result = append(result, Day{
            ID:          int(rrow.DayID),
            Date:        rrow.Name,
            Description: rrow.Description,
            FullInfo:    rrow.Description,
            Image:       rrow.ImageURL,
            BodiesText:  rrow.BodiesText,
            EarthRA:     rrow.EarthRA,
            EarthDEC:    rrow.EarthDEC,
            AsteroidRA:  rrow.AstRA,
            AsteroidDEC: rrow.AstDEC,
        })
    }
    return result, nil
}

func (r *Repository) GetObservation(id int) (Observation, error) {
    var o observation
    if err := r.db.Where("id = ?", id).First(&o).Error; err != nil {
        return Observation{}, err
    }
    desc := ""
    if o.Comment != nil {
        desc = *o.Comment
    }
    var res float64
    if o.CalculatedKM != nil {
        res = *o.CalculatedKM
    }
    return Observation{
        ID_observation: int(o.ID),
        Description:    desc,
        Result:         res,
    }, nil
}

// moved methods into feature files: days.go, observations.go

