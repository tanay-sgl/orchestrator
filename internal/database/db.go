package database

import (
	"context"
	"fmt"
	"os"

	"github.com/go-pg/pg/v10"
)

func CreateDatabaseConnectionFromEnv() (*pg.DB, error) {
	db := pg.Connect(&pg.Options{
		Addr:     os.Getenv("TIMESCALE_ADDRESS"),
		User:     os.Getenv("TIMESCALE_USER"),
		Password: os.Getenv("TIMESCALE_PASSWORD"),
		Database: os.Getenv("TIMESCALE_DATABASE"),
	})

	err := db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func CreatePostgresDSN() string {
    return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
        os.Getenv("TIMESCALE_USER"),
        os.Getenv("TIMESCALE_PASSWORD"),
        os.Getenv("TIMESCALE_ADDRESS"),
        os.Getenv("TIMESCALE_DATABASE"))
}

func PrintEnvVars() {
    fmt.Printf("TIMESCALE_USER: %s\n", os.Getenv("TIMESCALE_USER"))
    fmt.Printf("TIMESCALE_PASSWORD: %s\n", os.Getenv("TIMESCALE_PASSWORD"))
    fmt.Printf("TIMESCALE_ADDRESS: %s\n", os.Getenv("TIMESCALE_ADDRESS"))
    fmt.Printf("TIMESCALE_DATABASE: %s\n", os.Getenv("TIMESCALE_DATABASE"))
}