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

func SanitizeAndParseSQLQuery(query string) (string, error) {
    // Trim any whitespace and remove any trailing semicolon
    query = strings.TrimSpace(query)
    query = strings.TrimSuffix(query, ";")

    // List of disallowed keywords and patterns (case-insensitive)
	disallowedPatterns := []string{
		`\bDROP\b`, `\bDELETE\b`, `\bTRUNCATE\b`, `\bALTER\b`, `\bCREATE\b`, 
		`\bINSERT\b`, `\bUPDATE\b`, `\bGRANT\b`, `\bREVOKE\b`, 
		`--`, `/\*`, `\*/`, `\bEXEC\b`, `\bEXECUTE\b`,
	}

    // Combine all disallowed patterns into a single regex pattern
    pattern := strings.Join(disallowedPatterns, "|")
    regex := regexp.MustCompile(`(?i)(` + pattern + `)`)

    // Check for disallowed keywords and patterns
    if match := regex.FindString(query); match != "" {
        return "", fmt.Errorf("disallowed keyword or pattern found: %s", match)
    }

    // Validate that the query starts with SELECT
    if !regexp.MustCompile(`(?i)^\s*SELECT\b`).MatchString(query) {
        return "", fmt.Errorf("query must start with SELECT")
    }

    // Basic structure validation
    if !regexp.MustCompile(`(?i)\bFROM\b`).MatchString(query) {
        return "", fmt.Errorf("invalid query structure: missing FROM clause")
    }

    return query, nil
}

func ParseRowsToString(rows map[string][]map[string]interface{}) (string, error) {
	var result strings.Builder

	for tableName, rows := range rows {
		result.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		for i, row := range rows {
			result.WriteString(fmt.Sprintf("  Row %d:\n", i+1))
			for key, value := range row {
				result.WriteString(fmt.Sprintf("    %s: %v\n", key, value))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

func ParseDocumentsToString(documents []database.Document) (string, error) {
	var result strings.Builder
	result.WriteString("Relevant Documents:\n")
	for i, doc := range documents {
		result.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		result.WriteString(fmt.Sprintf("  Collection Slug: %s\n", doc.CollectionSlug))
		result.WriteString(fmt.Sprintf("  CID: %s\n", doc.CID))
		result.WriteString(fmt.Sprintf("  Content: %s\n", doc.Content))
		result.WriteString("\n")
	}

	return result.String(), nil
}

func SourceData(model string, data_sources []string, question string, search_limit int) (string, error) {
	var data strings.Builder
	db, err := database.CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	for _, data_source := range data_sources {
		switch data_source {
		case "documents":
			fmt.Printf("DOCUMENTS FLOW!\n")
			embedding, err := CreateEmbedding(model, question)
			if err != nil {
				continue
			}
			similarDocuments, err := database.GetSimilaritySearchDocuments(db, embedding, search_limit)
			if err != nil {
				continue
			}

			parsedDocuments, err := ParseDocumentsToString(similarDocuments)
			data.WriteString(parsedDocuments + "\n")

		case "sql":
			fmt.Printf("Getting SQL Rows!\n")
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

			sql_request, err := QueryOllama(model, []OllamaChatMessage{
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

			parsedRows, err := ParseRowsToString(map[string][]map[string]interface{}{
				"sql_result": result,
			})
			data.WriteString(parsedRows + "\n")

		case "default":
			fmt.Printf("DEFAULT FLOW!\n")
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

			parsedRows, err := ParseRowsToString(allSimilarRows)
			data.WriteString(parsedRows + "\n")
			parsedDocuments, err := ParseDocumentsToString(similarDocuments)
			data.WriteString(parsedDocuments + "\n")

		case "NA":
			return QueryOllama(model, []OllamaChatMessage{{Role: "user", Content: question}})
		}
	}

	return data.String(), nil
}

// Helper function to truncate a string
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

func ParseRelevantData(relevantData database.RelevantData) (string, error) {
	var result strings.Builder

	// Parse similar rows
	result.WriteString("Relevant Data from Database Tables:\n")
	for tableName, rows := range relevantData.SimilarRows {
		result.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		for i, row := range rows {
			result.WriteString(fmt.Sprintf("  Row %d:\n", i+1))
			for key, value := range row {
				result.WriteString(fmt.Sprintf("    %s: %v\n", key, value))
			}
		}
		result.WriteString("\n")
	}

	// Parse similar documents
	result.WriteString("Relevant Documents:\n")
	for i, doc := range relevantData.SimilarDocuments {
		result.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		result.WriteString(fmt.Sprintf("  Collection Slug: %s\n", doc.CollectionSlug))
		result.WriteString(fmt.Sprintf("  CID: %s\n", doc.CID))
		result.WriteString(fmt.Sprintf("  Content: %s\n", doc.Content))
		result.WriteString("\n")
	}

	return result.String(), nil
}
