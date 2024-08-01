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

func QueryOllama(model string, chatMessages []OllamaChatMessage) (string, error) {
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


func ProcessLLMSimpleQuery(request models.LLMSimpleQueryRequest) (string, error) {
    db,err := database.CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", fmt.Errorf("error creating database connection: %w", err)
	}
    defer db.Close()

    var conversationHistory []OllamaChatMessage
    if request.ConversationID != 0 {
        messages, err :=database.GetRecentMessages(db, request.ConversationID, 10)
        if err != nil {
            return "", fmt.Errorf("error retrieving conversation history: %w", err)
        }
        

        for _, msg := range messages {
            conversationHistory = append(conversationHistory, OllamaChatMessage{
                Role:    msg.Role,
                Content: msg.Content,
            })
        }
    }

    conversationHistory = append(conversationHistory, OllamaChatMessage{
        Role:    "user",
        Content: request.Input,
    })

    response, err := QueryOllama(request.Model, conversationHistory)
    if err != nil {
        return "", fmt.Errorf("error querying Ollama: %w", err)
    }

    if request.ConversationID != 0 {
        title := fmt.Sprintf("Simple Query: %s", truncateString(request.Input, 50))
        err = database.SaveMessages(db, request.ConversationID, []database.Message{
            {Role: "user", Content: request.Input},
            {Role: "assistant", Content: response},
        }, title)
        if err != nil {
            return "", fmt.Errorf("error saving conversation: %w", err)
        }
    }

    return response, nil
}

func ProcessLLMRAGQuery(request models.LLMRAGQueryRequest) (string, error) {

	decomposedQueriesAndAnswers, err := AgenticFlow(request)
	if err != nil {
		return "", err
	}
	return QueryOllama(request.Model,
		[]OllamaChatMessage{{Role: "user", Content: string(SnythesizeInstruction)},
			{Role: "user", Content: "QUERY:\n" + request.Input},
			{Role: "user", Content: "SUB QUERIES AND ANSWERS:\n" + decomposedQueriesAndAnswers}})
}

