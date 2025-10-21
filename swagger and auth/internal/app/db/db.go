package db

import (
    "fmt"
    "os"

    "github.com/sirupsen/logrus"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
    dsn := os.Getenv("DB_DSN")
    if dsn == "" {
        // default for local docker-compose
        host := getenv("DB_HOST", "localhost")
        port := getenv("DB_PORT", "5432")
        user := getenv("DB_USER", "lab2user")
        pass := getenv("DB_PASS", "lab2pass")
        name := getenv("DB_NAME", "lab2db")
        ssl := getenv("DB_SSLMODE", "disable")
        dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, name, ssl)
    }

    gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    logrus.Infof("connected to DB: %s", dsn)
    return gormDB, nil
}

func getenv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}


