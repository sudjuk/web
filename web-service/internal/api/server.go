package api

import (
    "front_start/internal/app/handler"
    appdb "front_start/internal/app/db"
    "front_start/internal/app/repository"
    appcfg "front_start/internal/app/config"
    appdsn "front_start/internal/app/dsn"
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

func StartServer() {
	log.Println("Starting server")

    // load config if present, otherwise fallback to env/defaults
    cfg, err := appcfg.Load("config/config.toml")
    if err != nil {
        logrus.Warnf("config load failed, using env/defaults: %v", err)
    } else {
        // set envs so db.Connect picks them up (keeps single place for DSN open)
        os.Setenv("DB_HOST", cfg.DB.Host)
        os.Setenv("DB_PORT", cfg.DB.Port)
        os.Setenv("DB_USER", cfg.DB.User)
        os.Setenv("DB_PASS", cfg.DB.Pass)
        os.Setenv("DB_NAME", cfg.DB.Name)
        os.Setenv("DB_SSLMODE", cfg.DB.SSLMode)
        // also provide a complete DB_DSN for convenience
        d := appdsn.Postgres{Host: cfg.DB.Host, Port: cfg.DB.Port, User: cfg.DB.User, Password: cfg.DB.Pass, DBName: cfg.DB.Name, SSLMode: cfg.DB.SSLMode}
        os.Setenv("DB_DSN", d.String())
    }

    // init DB
    gormDB, err := appdb.Connect()
    if err != nil {
        logrus.Fatalf("db connect error: %v", err)
        return
    }

    repo, err := repository.NewRepository(gormDB)
    if err != nil {
        logrus.Fatalf("ошибка инициализации репозитория: %v", err)
        return
    }

	handler := handler.NewHandler(repo)

	r := gin.Default()
	// добавляем наш html/шаблон
    r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")
	// слева название папки, в которую выгрузится наша статика
	// справа путь к папке, в которой лежит статика

    r.GET("/astronomy", handler.GetDays)
    r.GET("/day_details/:id", handler.GetDay)
    r.GET("/observation/:id", handler.GetObservation)

    // Legacy POST routes per lab2 (UI forms)
    r.POST("/order/items", handler.PostAddToOrder)
    r.POST("/order/delete", handler.PostDeleteOrder)

    // REST API v3 under /api
    api := r.Group("/api")
    {
        // Days (services domain renamed to topic-specific days)
        api.GET("/days", handler.ApiListDays)
        api.GET("/days/:id", handler.ApiGetDayAPI)
        api.POST("/days", handler.ApiCreateDay)
        api.PUT("/days/:id", handler.ApiUpdateDay)
        api.DELETE("/days/:id", handler.ApiDeleteDay)
        api.POST("/days/:id/add-to-draft", handler.ApiDayAddToDraft)
        api.POST("/days/:id/image", handler.ApiDayUploadImage)

        // Observations (orders)
        api.GET("/observations/cart", handler.ApiGetCartIcon)
        api.GET("/observations", handler.ApiListObservations)
        api.GET("/observations/:id", handler.ApiGetObservation)
        api.PUT("/observations/:id", handler.ApiUpdateObservation)
        api.PUT("/observations/:id/submit", handler.ApiSubmitObservation)
        api.PUT("/observations/:id/moderate", handler.ApiModerateObservation)
        api.DELETE("/observations/:id", handler.ApiDeleteObservation)

        // Many-to-many items
        api.DELETE("/observation-items", handler.ApiDeleteObservationItem)
        api.PUT("/observation-items", handler.ApiUpdateObservationItem)

        // Users
        api.POST("/auth/register", handler.ApiRegister)
        api.POST("/auth/login", handler.ApiLogin)
        api.POST("/auth/logout", handler.ApiLogout)
        api.GET("/me", handler.ApiGetMe)
        api.PUT("/me", handler.ApiUpdateMe)
    }

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	log.Println("Server down")
}
