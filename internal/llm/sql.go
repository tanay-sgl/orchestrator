package llm

import (
	"fmt"
	"orchestrator/internal/database"
	"regexp"
	"strings"

	_ "github.com/tmc/langchaingo/tools/sqldatabase/postgresql"
)

func SanitizeAndParseSQLQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	disallowedPatterns := []string{
		`\bDROP\b`, `\bDELETE\b`, `\bTRUNCATE\b`, `\bALTER\b`, `\bCREATE\b`,
		`\bINSERT\b`, `\bUPDATE\b`, `\bGRANT\b`, `\bREVOKE\b`,
		`--`, `/\*`, `\*/`, `\bEXEC\b`, `\bEXECUTE\b`,
	}

	pattern := strings.Join(disallowedPatterns, "|")
	regex := regexp.MustCompile(`(?i)(` + pattern + `)`)

	if match := regex.FindString(query); match != "" {
		return "", fmt.Errorf("disallowed keyword or pattern found: %s", match)
	}

	if !regexp.MustCompile(`(?i)^\s*SELECT\b`).MatchString(query) {
		return "", fmt.Errorf("query must start with SELECT")
	}

	if !regexp.MustCompile(`(?i)\bFROM\b`).MatchString(query) {
		return "", fmt.Errorf("invalid query structure: missing FROM clause")
	}

	return query, nil
}

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

	query, err := QueryOllama(modelName, []OllamaChatMessage{
		{Role: "user", Content: string(SQLInstruction)},
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
