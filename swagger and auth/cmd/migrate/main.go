package main

import (
    "fmt"
    appcfg "front_start/internal/app/config"
    appdb "front_start/internal/app/db"
    "github.com/sirupsen/logrus"
    "os"
)

func main() {
    cfg, err := appcfg.Load("config/config.toml")
    if err == nil {
        os.Setenv("DB_HOST", cfg.DB.Host)
        os.Setenv("DB_PORT", cfg.DB.Port)
        os.Setenv("DB_USER", cfg.DB.User)
        os.Setenv("DB_PASS", cfg.DB.Pass)
        os.Setenv("DB_NAME", cfg.DB.Name)
        os.Setenv("DB_SSLMODE", cfg.DB.SSLMode)
    }
    db, err := appdb.Connect()
    if err != nil {
        logrus.Fatalf("db connect error: %v", err)
        return
    }
    // Apply minimal migrations required for lab-3
    stmts := []string{
        // observations system columns
        "ALTER TABLE observations ADD COLUMN IF NOT EXISTS moderator_id bigint",
        "ALTER TABLE observations ADD COLUMN IF NOT EXISTS created_at timestamp",
        "ALTER TABLE observations ADD COLUMN IF NOT EXISTS formed_at timestamp",
        "ALTER TABLE observations ADD COLUMN IF NOT EXISTS completed_at timestamp",
    }
    for _, s := range stmts {
        if err := db.Exec(s).Error; err != nil {
            logrus.Fatalf("migration failed on '%s': %v", s, err)
            return
        }
    }
    sqlDB, err := db.DB()
    if err != nil {
        logrus.Fatalf("db unwrap error: %v", err)
        return
    }
    defer sqlDB.Close()
    fmt.Println("DB connection OK")
}


