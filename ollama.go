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

func QueryOllama(model string, chatMessages []ChatMessage) (string, error) {
	fmt.Printf("Querying Ollama...\n")
	url := "http://localhost:11434/api/chat"

	jsonQuery, err := json.Marshal(OllamaRequest{
		Model:    model,
		Messages: chatMessages,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	fmt.Printf("Request: %s\n", string(jsonQuery))

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
			fmt.Print(ollamaResponse.Message.Content) // Print each chunk as it's received
		}

		if ollamaResponse.Done {
			fmt.Printf("\nResponse completed. Total duration: %d ns\n", ollamaResponse.TotalDuration)
			break // Exit the loop when we receive the "done" signal
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

func ProcessLLMQuery(request LLMQueryRequest) (string, error) {
	// db, err := CreateDatabaseConnectionFromEnv()
	// if err != nil {
	// 	fmt.Errorf("Error connecting to database: %v", err)
	// }
	// defer db.Close()
	// messageWithContext := FormatPromptWithContext(request, GetRelevantDataWithoutAnalysisFromAllTables(request))

	// previousMessages, err := GetRecentMessages(db, request.ConversationID, request.SearchLimit)
	// if err != nil {
	// 	return "", err
	// }
	// messages := FormatMessages(messageWithContext, previousMessages)

	// return QueryOllama(request.Model, messages)
}

func AgenticFlow(request LLMQueryRequest) (string, error) {
	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		fmt.Printf("Error connecting to database: %v", err)
	}
	defer db.Close()

	decomposed_query_request, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(SubquestionInstruction)}, {Role: "user", Content: request.Input}})
	if err != nil {
		return "", err
	}
	decomposed_query, err := ParseSubQuestions(decomposed_query_request)
	if err != nil {
		return "", err
	}

	return AnswerSubQuestionsRecursively(request, decomposed_query, "")

}



func AnswerSubQuestionsRecursively(request LLMQueryRequest, sub_questions []string, previous_answer string) (string, error) {

	if len(sub_questions) == 0 {
		return previous_answer, nil
	}

	question := sub_questions[0]

	data_source_request, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(DataSourceInstruction)}, {Role: "user", Content: question}})
	if err != nil {
		return previous_answer, err
	}

	data_sources, err := ParseDataSources(data_source_request)
	if err != nil {
		return previous_answer, err
	}

	data, err := SourceData(request.Model, data_sources, question)
	if err != nil {
		return previous_answer, err
	}

	answer, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})

	if err != nil {
		return previous_answer, err
	}

	hallucination_check, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(HallucinationDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}}) 

	if err != nil {
		return previous_answer, err
	}

	if hallucination_check == "YES" {
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})
		hallucination_check, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(HallucinationDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}}) 

	}

	if hallucination_check == "YES" {
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})
	}

	correctness_check, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(CorrectnessDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})

	if err != nil {
		return previous_answer, err
	}

	if correctness_check == "NO" {
		// Let's try again but with the default search as a fall back
		data, err = SourceData(request.Model, []string{"default"}, question)
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})
		if hallucination_check == "YES" {
			answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})
			hallucination_check, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(HallucinationDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}}) 
	
		}
	
		if hallucination_check == "YES" {
			answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer + "\n" + data + "\n" + question}})
		}
	
		correctness_check, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(CorrectnessDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})
	
			
	}

	if correctness_check == "NO" {
		//At this point we give up on sourcing data and move on to the next question
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: previous_answer +"\n" + question}})
	}

	return AnswerSubQuestionsRecursively(request , sub_questions[1:], answer)
}

