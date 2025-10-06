package main

import (
    "log"

    "front_start/internal/pkg"
)

func main() {
    log.Println("Application start!")
    pkg.App()
    log.Println("Application terminated!")
}
