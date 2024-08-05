package llm

import (
	"fmt"
	"orchestrator/internal/models"
	"sync"
)

func ProcessLLMRAGQuery(request models.LLMRAGQueryRequest) (string, error) {
    // Generate sub-questions
    decomposed_query_request, err := QueryOllama(request.Model, []OllamaChatMessage{
        {Role: "user", Content: string(SubquestionInstruction)},
        {Role: "user", Content: request.Input},
    })
    if err != nil {
        return "", err
    }
    
    decomposed_query, err := ParseSubQuestions(decomposed_query_request)
    if err != nil {
        return "", err
    }

    // Process sub-questions concurrently
    var subQuestionAnswers []string
    answersChan := make(chan string, len(decomposed_query))
    var wg sync.WaitGroup

    for _, question := range decomposed_query {
        wg.Add(1)
        go func(q string) {
            defer wg.Done()
            answer, err := AnswerSubQuestion(request, q)
            if err != nil {
                answersChan <- fmt.Sprintf("Error answering sub-question: %v", err)
            } else {
                answersChan <- fmt.Sprintf("Sub-question: %s\nAnswer: %s", q, answer)
            }
        }(question)
    }

    wg.Wait()
    close(answersChan)

    for answer := range answersChan {
        subQuestionAnswers = append(subQuestionAnswers, answer)
    }

    return  QueryOllama(request.Model, []OllamaChatMessage{
        {Role: "user", Content: string(SynthesizeInstruction)},
        {Role: "user", Content: "Original Query: " + request.Input},
        {Role: "user", Content: "Sub-questions and Answers:\n" + FormatSubQuestionAnswers(subQuestionAnswers)},
    })
}

func AnswerSubQuestion(request models.LLMRAGQueryRequest, question string) (string, error) {
	data_source_request, err := QueryOllama(request.Model, []OllamaChatMessage{{Role: "user", Content: string(DataSourceInstruction)}, {Role: "user", Content: "QUERY TO ANALYZE: \n" + question}})
	if err != nil {
		return "", err
	}

	if data_source_request == "sql" {
		fmt.Printf("GETTING SQL ONLY\n")
		return QueryUserRequestAsSQL(request.Model, question)
	}

	var data string

	if data_source_request == "documents" {
		fmt.Printf("GETTING DOCUMENTS ONLY\n")
		data, err = QueryUserRequestForSimilarDocuments(request, question)
		if err != nil {
			return "", err
		}
	} else {
		fmt.Printf("GETTING BOTH\n")
		document_data, err := QueryUserRequestForSimilarDocuments(request, question)
		if err != nil {
			return "", err
		}
		sql_data, err := QueryUserRequestAsSQL(request.Model, question)
		if err != nil {
			return "", err
		}
		data = document_data + "\n" + sql_data
	}

	return QueryOllama(request.Model, []OllamaChatMessage{
		{Role: "user", Content: string(GameFIGeniusInstruction)},
		{Role: "user", Content: "DATA:\n" + data},
		{Role: "user", Content: "QUERY:\n" + question},
	})

}
