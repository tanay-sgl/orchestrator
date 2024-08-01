package llm

import (
	"context"
	"fmt"
	"orchestrator/internal/database"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/tools/sqldatabase"
	_ "github.com/tmc/langchaingo/tools/sqldatabase/postgresql"
)

func QueryUserRequestAsSQL(modelName string, input any) (string, error) {
	model, err := ollama.New(ollama.WithModel(modelName))
	if err != nil {
		return "", err
	}

	db, err := sqldatabase.NewSQLDatabaseWithDSN("pgx", database.CreatePostgresDSN(), nil)
	if err != nil {
		return "", fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	sqlDatabaseChain := chains.NewSQLDatabaseChain(
		model, 100, db)

	tables, err := database.GetTableSchemaAsString()
	if err != nil {
		return "", err
	}
	input = fmt.Sprintf("%s\n%s", input, tables)

	ctx := context.Background()
	result, err := chains.Run(ctx, sqlDatabaseChain, input)
	if err != nil {
		return "", fmt.Errorf("error running chain: %w", err)
	}
	return result, nil
}
