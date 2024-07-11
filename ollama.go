package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

func QueryOllama(model string, chatMessages []ChatMessage) (string, error) {
	//fmt.Printf("Querying Ollama...\n")
	url := "http://localhost:11434/api/chat"

	jsonQuery, err := json.Marshal(OllamaRequest{
		Model:    model,
		Messages: chatMessages,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	//fmt.Printf("Request: %s\n", string(jsonQuery))

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
			//fmt.Print(ollamaResponse.Message.Content) // Print each chunk as it's received
		}

		if ollamaResponse.Done {
			//fmt.Printf("\nResponse completed. Total duration: %d ns\n", ollamaResponse.TotalDuration)
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
	decomposedQueriesAndAnswers, err := AgenticFlow(request)
	if err != nil {
		return "", err
	}
	return QueryOllama(request.Model,
		[]ChatMessage{{Role: "user", Content: string(SnythesizeInstruction)},
			{Role: "user", Content: "QUERY:\n" + request.Input},
			{Role: "user", Content: "SUB QUERIES AND ANSWERS:\n" + decomposedQueriesAndAnswers}})
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

	var subQuestionAnswers []string
	answersChan := make(chan string, len(decomposed_query))
	var wg sync.WaitGroup

	// Loop through each entry in decomposed_query
	for _, question := range decomposed_query {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()
			// Call AnswerSubQuestion for each question
			answer, err := AnswerSubQuestion(request, q)
			if err != nil {
				// Handle the error if needed
				answersChan <- fmt.Sprintf("Error answering sub-question: %v", err)
			} else {
				// Format the sub-question and answer
				formattedResult := fmt.Sprintf("Sub-question: %s\nAnswer: %s", q, answer)
				answersChan <- formattedResult
			}
		}(question)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(answersChan)

	// Collect answers
	for answer := range answersChan {
		subQuestionAnswers = append(subQuestionAnswers, answer)
	}

	return FormatSubQuestionAnswers(subQuestionAnswers), nil
}

func AnswerSubQuestion(request LLMQueryRequest, question string) (string, error) {
	data_source_request, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(DataSourceInstruction)}, {Role: "user", Content: "QUERY TO ANALYZE: \n" + question}})
	if err != nil {
		return "", err
	}
	data_sources, err := ParseDataSources(data_source_request)
	if err != nil {
		return "", err
	}
	data, err := SourceData(request.Model, data_sources, question, request.SearchLimit)
	if err != nil {
		return "", err
	}
	answer, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
	if err != nil {
		return "", err
	}

	// Hallucination check
	for i := 0; i < 2; i++ {
		hallucination_check, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(HallucinationDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})
		if err != nil {
			return "", err
		}
		if AnalyzeYesNoResponse(hallucination_check) {
			break
		}
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
		if err != nil {
			return "", err
		}
	}

	// Correctness check
	correctness_check, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(CorrectnessDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})
	if err != nil {
		return "", err
	}
	if !AnalyzeYesNoResponse(correctness_check) {
		// Let's try again but with the default search as a fall back
		data, err = SourceData(request.Model, []string{"default"}, question, request.SearchLimit)
		if err != nil {
			return "", err
		}
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
		if err != nil {
			return "", err
		}

		// Repeat hallucination check
		for i := 0; i < 2; i++ {
			hallucination_check, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(HallucinationDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})
			if err != nil {
				return "", err
			}
			if AnalyzeYesNoResponse(hallucination_check) {
				break
			}
			answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
			if err != nil {
				return "", err
			}
		}

		correctness_check, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(CorrectnessDetectiveInstruction)}, {Role: "user", Content: "\n QUESTION: " + question + "\n ANSWER: " + answer}})
		if err != nil {
			return "", err
		}
	}

	if !AnalyzeYesNoResponse(correctness_check) {
		// At this point we give up on sourcing data and move on to the next question
		answer, err = QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "QUERY:\n" + question}})
		if err != nil {
			return "", err
		}
	}

	return answer, nil
}
