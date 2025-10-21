package config

import (
    "github.com/pelletier/go-toml/v2"
    "os"
)

type Config struct {
    DB struct {
        Host    string `toml:"host"`
        Port    string `toml:"port"`
        User    string `toml:"user"`
        Pass    string `toml:"pass"`
        Name    string `toml:"name"`
        SSLMode string `toml:"sslmode"`
    } `toml:"db"`
    JWT struct {
        Secret        string `toml:"secret"`
        ExpiresIn     int64  `toml:"expires_in"` // в секундах
        SigningMethod string `toml:"signing_method"`
    } `toml:"jwt"`
    Redis struct {
        Host        string `toml:"host"`
        Port        int    `toml:"port"`
        Password    string `toml:"password"`
        DialTimeout int    `toml:"dial_timeout"` // в секундах
        ReadTimeout int    `toml:"read_timeout"` // в секундах
    } `toml:"redis"`
}

func Load(path string) (Config, error) {
    var cfg Config
    data, err := os.ReadFile(path)
    if err != nil {
        return cfg, err
    }
    if err := toml.Unmarshal(data, &cfg); err != nil {
        return cfg, err
    }
    return cfg, nil
}


