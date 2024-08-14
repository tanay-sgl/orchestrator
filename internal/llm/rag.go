package llm

import (
	"fmt"
	"orchestrator/internal/models"
	"sync"
)

func ProcessLLMRAGQuerySingleNode(request models.LLMRAGQueryRequest) (string, error) {
    data, err := QueryUserRequestForSimilarDocuments(request)
    if err != nil {
        return "", err
    }

	return QueryOllama(request.Model, []OllamaChatMessage{
		{Role: "user", Content: string(GameFIGeniusInstruction)},
		{Role: "user", Content: "DATA:\n" + data},
		{Role: "user", Content: "QUERY:\n" + request.Input},
	})
}

func ProcessLLMRAGQueryMultiNode(request models.LLMRAGQueryRequest) (string, error) {
	fmt.Println("RAGGING!\n")
	// Generate sub-questions
	fmt.Println("making sub questions\n")
	decomposed_query_request, err := QueryOllama(request.Model, []OllamaChatMessage{
		{Role: "user", Content: string(SubquestionInstruction)},
		{Role: "user", Content: request.Input},
	})
	if err != nil {
		return "", err
	}
	fmt.Println("sub questions generated\n")
	fmt.Println(decomposed_query_request)
	fmt.Println("parsing sub questions\n")
	decomposed_query, err := ParseSubQuestions(decomposed_query_request)
	if err != nil {
		return "", err
	}
	fmt.Println(decomposed_query)
	fmt.Println("threading\n")
	// Process sub-questions concurrently
	var subQuestionAnswers []string
	answersChan := make(chan string, len(decomposed_query))
	var wg sync.WaitGroup

	for _, question := range decomposed_query {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()
			fmt.Println("doing sub question!\n")
			answer, err := ProcessLLMRAGQuerySingleNode(
				models.LLMRAGQueryRequest{Model: request.Model,
					Input:          q,
					SearchLimit:    request.SearchLimit,
					DataSources:    request.DataSources,
					ConversationID: request.ConversationID})
			if err != nil {
				answersChan <- fmt.Sprintf("Error answering sub-question: %v", err)
			} else {
				answersChan <- fmt.Sprintf("Sub-question: %s\nAnswer: %s", q, answer)
			}
		}(question)
		fmt.Println("moving to next sub question!\n")
	}

	wg.Wait()
	close(answersChan)

	for answer := range answersChan {
		fmt.Println("collating result!\n")
		subQuestionAnswers = append(subQuestionAnswers, answer)
	}

	fmt.Println("returning result!\n")
	return QueryOllama(request.Model, []OllamaChatMessage{
		{Role: "user", Content: string(SynthesizeInstruction)},
		{Role: "user", Content: "Original Query: " + request.Input},
		{Role: "user", Content: "Sub-questions and Answers:\n" + FormatSubQuestionAnswers(subQuestionAnswers)},
	})
}
