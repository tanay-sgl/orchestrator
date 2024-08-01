package main

import (
	"log"
	"orchestrator/internal/api"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
	r := api.SetupRouter()
	r.Run(":8080")
}
