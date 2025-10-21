package main

import (
    "log"

    "front_start/internal/pkg"
)

// @title Astronomy Asteroid Request API
// @version 1.0
// @description API для управления астрономическими наблюдениями и заявками на астероиды
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите JWT токен в формате: Bearer {token}

func main() {
    log.Println("Application start!")
    pkg.App()
    log.Println("Application terminated!")
}
