package api

import (
    "context"
    "front_start/internal/app/handler"
    appdb "front_start/internal/app/db"
    "front_start/internal/app/repository"
    appcfg "front_start/internal/app/config"
    appdsn "front_start/internal/app/dsn"
    "front_start/internal/app/redis"
    "log"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    ginSwagger "github.com/swaggo/gin-swagger"
    swaggerFiles "github.com/swaggo/files"
    _ "front_start/docs" // импорт сгенерированной документации
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

    // init Redis
    ctx := context.Background()
    redisClient, err := redis.New(ctx, cfg)
    if err != nil {
        logrus.Fatalf("Redis connect error: %v", err)
        return
    }
    defer redisClient.Close()

	handler := handler.NewHandler(repo, redisClient, cfg.JWT.Secret)

	r := gin.Default()
	// добавляем наш html/шаблон
    r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")
	// слева название папки, в которую выгрузится наша статика
	// справа путь к папке, в которой лежит статика

    // Swagger UI
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    r.GET("/astronomy", handler.GetDays)
    r.GET("/day_details/:id", handler.GetDay)
    r.GET("/asteroid-observation/:id", handler.GetAsteroidObservation)

    // Legacy POST routes per lab2 (UI forms)
    r.POST("/order/items", handler.PostAddToOrder)
    r.POST("/order/delete", handler.PostDeleteOrder)

    // REST API v3 under /api
    api := r.Group("/api")
    {
        // Public endpoints (no auth required)
        api.POST("/auth/register", handler.ApiRegister)
        api.POST("/auth/login", handler.ApiLogin)
        api.GET("/days", handler.ApiListDays)
        api.GET("/days/:id", handler.ApiGetDayAPI)
        api.GET("/asteroid-observations/:id", handler.ApiGetAsteroidObservation)

        // User endpoints (auth required)
        userAPI := api.Group("")
        userAPI.Use(handler.WithAuthCheck())
        {
            userAPI.POST("/auth/logout", handler.ApiLogout)
            userAPI.GET("/me", handler.ApiGetMe)
            userAPI.PUT("/me", handler.ApiUpdateMe)
            userAPI.GET("/asteroid-observations", handler.ApiListAsteroidObservations)
            userAPI.GET("/asteroid-observations/cart", handler.ApiGetCartIcon)
            userAPI.POST("/days/:id/add-to-draft", handler.ApiDayAddToDraft)
            userAPI.PUT("/asteroid-observations/:id", handler.ApiUpdateAsteroidObservation)
            userAPI.PUT("/asteroid-observations/:id/submit", handler.ApiSubmitAsteroidObservation)
            userAPI.DELETE("/asteroid-observations/:id", handler.ApiDeleteAsteroidObservation)
            userAPI.DELETE("/asteroid-observation-items", handler.ApiDeleteAsteroidObservationItem)
            userAPI.PUT("/asteroid-observation-items", handler.ApiUpdateAsteroidObservationItem)
        }

        // Moderator endpoints (moderator role required)
        moderatorAPI := api.Group("")
        moderatorAPI.Use(handler.WithModeratorCheck())
        {
            moderatorAPI.POST("/days", handler.ApiCreateDay)
            moderatorAPI.PUT("/days/:id", handler.ApiUpdateDay)
            moderatorAPI.DELETE("/days/:id", handler.ApiDeleteDay)
            moderatorAPI.POST("/days/:id/image", handler.ApiDayUploadImage)
            moderatorAPI.PUT("/asteroid-observations/:id/moderate", handler.ApiModerateAsteroidObservation)
        }
    }

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	log.Println("Server down")
}
