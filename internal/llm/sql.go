package llm

import (
	"fmt"
	"orchestrator/internal/database"

	_ "github.com/tmc/langchaingo/tools/sqldatabase/postgresql"
)

func QueryUserRequestAsSQL(modelName string, input any) (string, error) {
	db, err := database.CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", fmt.Errorf("error creating database connection: %w", err)
	}
	defer db.Close()
	tableSchema, err := database.GetTableSchemaAsString()
	if err != nil {
		return "", fmt.Errorf("error getting table schema: %w", err)
	}

	query, err := QueryOllama(modelName, []OllamaChatMessage{{Role: "user", Content: string(SnythesizeInstruction)},
		{Role:"user", Content: string(SQLInstruction)},
		{Role: "user", Content: tableSchema},
		{Role: "user", Content: "QUERY:\n" + fmt.Sprintf("%v", input)}})

	if err != nil {
		return "", fmt.Errorf("error querying Ollama: %w", err)
	}

	query, err = SanitizeAndParseSQLQuery(query)
	if err != nil {
		return "", fmt.Errorf("error sanitizing and parsing SQL query: %w", err)
	}

	result, err := database.ExecuteSQLQuery(db, query)
	if err != nil {
		return "", fmt.Errorf("error executing SQL query: %w", err)
	}
	return fmt.Sprintf("%v", result), nil
}
