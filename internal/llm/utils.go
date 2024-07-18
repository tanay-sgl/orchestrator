package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"orchestrator/internal/database"
	"regexp"
	"strings"
)


func ParseDataSources(response string) ([]string, error) {
	// Trim any whitespace from the response
	response = strings.TrimSpace(response)

	// Check if the response is empty
	if response == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Check if the response is "NA"
	if response == "NA" {
		return []string{"NA"}, nil
	}

	// Split the response by comma
	sources := strings.Split(response, ",")

	// Validate and clean each source
	validSources := make([]string, 0, len(sources))
	for _, source := range sources {
		// Trim whitespace
		source = strings.TrimSpace(source)

		// Validate the source
		switch source {
		case "documents", "sql", "default":
			validSources = append(validSources, source)
		default:
			return nil, fmt.Errorf("invalid data source: %s", source)
		}
	}

	// Check if we have at least one valid source
	if len(validSources) == 0 {
		return nil, fmt.Errorf("no valid data sources found")
	}

	return validSources, nil
}

func AnalyzeYesNoResponse(response string) bool {
	// Search for "YES" or "NO" in the entire string
	hasYes := strings.Contains(response, "YES")
	hasNo := strings.Contains(response, "NO")

	// Return true if either "YES" or "NO" is found
	return hasYes || hasNo
}

func FormatSubQuestionAnswers(subQuestionAnswers []string) string {
	var result strings.Builder

	for i, entry := range subQuestionAnswers {
		// Add a numbered header for each sub-question and answer pair
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry))

		// Add a separator between entries, except for the last one
		if i < len(subQuestionAnswers)-1 {
			result.WriteString("\n---\n\n")
		}
	}

	return result.String()
}

func ParseSubQuestions(response string) ([]string, error) {
	// Initialize a slice to hold the sub-questions
	var subQuestions []string

	// Create a scanner to read the response line by line
	scanner := bufio.NewScanner(strings.NewReader(response))

	// Flag to indicate when we've reached the sub-questions
	subQuestionsStarted := false

	// Iterate through each line of the response
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check if we've reached the start of the sub-questions
		if strings.HasPrefix(line, "SUB QUESTIONS:") {
			subQuestionsStarted = true
			continue
		}

		// If we've started processing sub-questions and the line is not empty
		if subQuestionsStarted && line != "" {
			// Remove the number and period at the start of the line
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				subQuestions = append(subQuestions, parts[1])
			}
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning response: %w", err)
	}

	// Check if we found any sub-questions
	if len(subQuestions) == 0 {
		return []string{}, nil
	}

	return subQuestions, nil
}

func SanitizeAndParseSQLQuery(response string) (string, error) {
	//fmt.Printf(response)
	// First, let's extract the SQL query from the response
	sqlRegex := regexp.MustCompile(`(?i)SQL:\s*(.+)`)
	matches := sqlRegex.FindStringSubmatch(response)
	if len(matches) < 2 {
		return "", fmt.Errorf("no SQL query found in the response")
	}
	query := matches[1]

	// Trim any whitespace and remove any trailing semicolon
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	// List of disallowed keywords (case-insensitive)
	disallowedKeywords := []string{
		"DROP", "DELETE", "TRUNCATE", "ALTER", "CREATE", "INSERT", "UPDATE",
		"GRANT", "REVOKE", "UNION", "--", "/*", "*/", "EXEC", "EXECUTE",
	}

	// Check for disallowed keywords
	lowerQuery := strings.ToLower(query)
	for _, keyword := range disallowedKeywords {
		if strings.Contains(lowerQuery, strings.ToLower(keyword)) {
			return "", fmt.Errorf("disallowed keyword found: %s", keyword)
		}
	}

	// Validate that the query starts with SELECT
	if !strings.HasPrefix(lowerQuery, "select") {
		return "", fmt.Errorf("query must start with SELECT")
	}

	// Basic structure validation
	// This is a simple check and might need to be expanded based on your specific needs
	if !strings.Contains(lowerQuery, "from") {
		return "", fmt.Errorf("invalid query structure: missing FROM clause")
	}

	return query, nil
} 
func SourceData(model string, data_sources []string, question string, search_limit int) (string, error) {
    relevant_data := database.RelevantData{}
    db, err := database.CreateDatabaseConnectionFromEnv()
    if err != nil {
        return "", fmt.Errorf("error connecting to database: %w", err)
    }
    defer db.Close()

    for _, data_source := range data_sources {
        switch data_source {
        case "documents":
            embedding, err := CreateEmbedding(model, question)
            if err != nil {
                continue
            }
            similarDocuments, err := database.GetSimilaritySearchDocuments(db, embedding, search_limit)
            if err != nil {
                continue
            }
            relevant_data.SimilarDocuments = similarDocuments

        case "sql":
            requestEmbedding, err := CreateEmbedding(model, question)
            if err != nil {
                continue
            }

            collection_string, err := database.GetSimilarRowsFromTable("collection", requestEmbedding, search_limit)
            if err != nil {
                continue
            }

            resultRow, err := json.Marshal(collection_string)
            if err != nil {
                return "", fmt.Errorf("failed to marshal result: %v", err)
            }

            sql_request, err := QueryOllama(model, []ChatMessage{
                {Role: "user", Content: string(SQLInstruction)},
                {Role: "user", Content: "METADATA:\n" + string(resultRow) + "\nQUERY:\n"},
                {Role: "user", Content: question},
            })
            if err != nil {
                continue
            }

            sanitized_sql_request, err := SanitizeAndParseSQLQuery(sql_request)
            if err != nil {
                continue
            }

            result, err := database.ExecuteSQLQuery(db, sanitized_sql_request)
            if err != nil {
                return "", fmt.Errorf("error executing SQL query: %w", err)
            }

            relevant_data.SimilarRows = map[string][]map[string]interface{}{
                "sql_result": result,
            }

        case "default":
            embedding, err := CreateEmbedding(model, question)
            if err != nil {
                return "", fmt.Errorf("error creating embedding: %w", err)
            }

            allSimilarRows, err := database.GetAllSimilarRowsFromDB(db, embedding, search_limit)
            if err != nil {
                return "", fmt.Errorf("error getting default data: %w", err)
            }

            similarDocuments, err := database.GetSimilaritySearchDocuments(db, embedding, search_limit)
            if err != nil {
                return "", fmt.Errorf("error getting similar documents: %w", err)
            }

            relevant_data.SimilarRows = allSimilarRows
            relevant_data.SimilarDocuments = similarDocuments

        case "NA":
            return QueryOllama(model, []ChatMessage{{Role: "user", Content: question}})
        }
    }

    return ParseRelevantData(relevant_data)
}
