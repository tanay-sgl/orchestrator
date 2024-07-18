package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"orchestrator/internal/database"
	"orchestrator/internal/models"
	"strings"
)

func QueryOllama(model string, chatMessages []ChatMessage) (string, error) {
	url := "http://localhost:11434/api/chat"

	jsonQuery, err := json.Marshal(OllamaRequest{
		Model:    model,
		Messages: chatMessages,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonQuery))
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var ollamaResponse OllamaResponse
		err = json.Unmarshal([]byte(line), &ollamaResponse)
		if err != nil {
			fmt.Printf("Error unmarshaling JSON: %v\n", err)
			continue
		}

		if ollamaResponse.Message.Content != "" {
			fullResponse.WriteString(ollamaResponse.Message.Content)
			}

		if ollamaResponse.Done {
			break 
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if fullResponse.Len() == 0 {
		log.Print("No valid response found in the API output")
		return "", nil
	}

	return fullResponse.String(), nil
}

func ProcessLLMQuery(request models.LLMQueryRequest) (string, error) {
	decomposedQueriesAndAnswers, err := AgenticFlow(request)
	if err != nil {
		return "", err
	}
	return QueryOllama(request.Model,
		[]ChatMessage{{Role: "user", Content: string(SnythesizeInstruction)},
			{Role: "user", Content: "QUERY:\n" + request.Input},
			{Role: "user", Content: "SUB QUERIES AND ANSWERS:\n" + decomposedQueriesAndAnswers}})
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