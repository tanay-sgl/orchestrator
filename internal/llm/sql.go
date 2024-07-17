package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/tools/sqldatabase"
	_ "github.com/tmc/langchaingo/tools/sqldatabase/postgresql"
)


func QueryUserRequestAsSQL(request LLMQueryRequest) (string, error) {
	model, err := ollama.New(ollama.WithModel(request.Model))
	if err != nil {
		return "", err
	}

	db, err := sqldatabase.NewSQLDatabaseWithDSN("postgres", createPostgresDSN(), nil)
    if err != nil {
        return "", fmt.Errorf("error connecting to database: %w", err)
    }
    defer db.Close()

	sqlDatabaseChain := chains.NewSQLDatabaseChain(
		model, 100, db)

	ctx := context.Background()
	result, err := chains.Run(ctx, sqlDatabaseChain, request.Input)
	if err != nil {
		return "", err
	}
	return result, nil
}