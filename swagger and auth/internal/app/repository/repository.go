package repository

import (
    "errors"
    "time"
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

type asteroidObservation struct {
    ID           int64   `gorm:"column:id;primaryKey"`
    Status       string  `gorm:"column:status"`
    CreatorID    int64   `gorm:"column:creator_id"`
    ModeratorID  *int64  `gorm:"column:moderator_id"`
    Comment      *string `gorm:"column:comment"`
    CalculatedKM *float64 `gorm:"column:calculated_km"`
    CreatedAt    *time.Time `gorm:"column:created_at"`
    FormedAt     *time.Time `gorm:"column:formed_at"`
    CompletedAt  *time.Time `gorm:"column:completed_at"`
}

func (asteroidObservation) TableName() string { return "asteroid_observations" }

type asteroidObservationItem struct {
    AsteroidObservationID int64   `gorm:"column:observation_id;primaryKey;foreignKey:observation_id"`
    DayID                 int64   `gorm:"column:day_id;primaryKey"`
    AsteroidRA        float64 `gorm:"column:asteroid_ra"`
    AsteroidDEC           float64 `gorm:"column:asteroid_dec"`
}

func (asteroidObservationItem) TableName() string { return "asteroid_observation_items" }

// GetAsteroidObservationDays возвращает список дней наблюдения с координатами из м-м
func (r *Repository) GetAsteroidObservationDays(asteroidObservationID int) ([]Day, error) {
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
    if err := r.db.Table("asteroid_observation_items oi").
        Select("ad.id as day_id, ad.name, ad.image_url, ad.bodies_text, oi.asteroid_ra as ast_ra, oi.asteroid_dec as ast_dec, ad.earth_ra, ad.earth_dec").
        Joins("JOIN astro_days ad ON ad.id = oi.day_id").
        Where("oi.observation_id = ?", asteroidObservationID).
        Order("ad.id").
        Scan(&rows).Error; err != nil {
        return nil, err
    }
    result := make([]Day, 0, len(rows))
    for _, rrow := range rows {
        result = append(result, Day{
            ID:          int(rrow.DayID),
            Date:        rrow.Name,
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
    var o asteroidObservation
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

