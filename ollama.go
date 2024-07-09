package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)


func GetRelevantDataWithoutAnalysisFromAllTables(request LLMQueryRequest) RelevantData {
	requestEmbedding, err := CreateEmbedding("llama3", request.Input)
	if err != nil {
		log.Fatal(err)
	}

	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		fmt.Errorf("Error connecting to database: %v", err)
	}
	defer db.Close()

	similarRows, err := GetAllSimilarRowsFromDB(db, TableNames, requestEmbedding, request.SearchLimit)
	if err != nil {
		log.Fatal(err)
	}

	similarDocumentContent, err := GetSimilarDocuments(db, requestEmbedding, request.SearchLimit)
	if err != nil {
		log.Fatal(err)
	}

	return RelevantData{
		SimilarRows:      similarRows,
		SimilarDocuments: similarDocumentContent,
	}
}


func FormatPromptWithContext(request LLMQueryRequest, relevantData RelevantData) ChatMessage {
	var contentBuilder strings.Builder

	// Add new user input
	contentBuilder.WriteString(fmt.Sprintf("%s\n\n", request.Input))

	// Add relevant data context
	contentBuilder.WriteString("Context from relevant documents:\n")
	for _, doc := range relevantData.SimilarDocuments {
		contentBuilder.WriteString(doc.Content)
		contentBuilder.WriteString("\n")
	}

	return ChatMessage{
		Role:    "user",
		Content: contentBuilder.String(),
	}
}

func FormatMessages(contextMessage ChatMessage, messages []Message) []ChatMessage {
	formattedMessages := make([]ChatMessage, 0, len(messages)+1)

	for _, msg := range messages {
		formattedMessages = append(formattedMessages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	formattedMessages = append(formattedMessages, contextMessage)

	return formattedMessages
}

func CreateCustomChatMessage(role string, content string) ChatMessage {
	return ChatMessage{
		Role:    role,
		Content: content,
	}
}

func QueryOllama(request LLMQueryRequest, chatMessages []ChatMessage) string {
	fmt.Printf("Querying Ollama...\n")
	url := "http://localhost:11434/api/chat"

	jsonQuery, err := json.Marshal(OllamaRequest{
		Model:    request.Model,
		Messages: chatMessages,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Request: %s\n", string(jsonQuery))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonQuery))
	if err != nil {
		log.Fatal(err)
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
			fmt.Print(ollamaResponse.Message.Content) // Print each chunk as it's received
		}

		if ollamaResponse.Done {
			fmt.Printf("\nResponse completed. Total duration: %d ns\n", ollamaResponse.TotalDuration)
			break // Exit the loop when we receive the "done" signal
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading response: %v", err)
	}

	if fullResponse.Len() == 0 {
		log.Print("No valid response found in the API output")
		return ""
	}

	return fullResponse.String()
}

func ProcessLLMQuery(request LLMQueryRequest) (string, error) {
	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		fmt.Errorf("Error connecting to database: %v", err)
	}
	defer db.Close()
	messageWithContext := FormatPromptWithContext(request, GetRelevantDataWithoutAnalysisFromAllTables(request))

	previousMessages, err := GetRecentMessages(db, request.ConversationID, request.SearchLimit)
	if err != nil {
		return "", err
	}
	messages := FormatMessages(messageWithContext, previousMessages)

	return QueryOllama(request, messages), nil
}

