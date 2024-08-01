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
