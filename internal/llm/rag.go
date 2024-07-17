package llm

import (
	"fmt"
	"sync"
)

func AgenticFlow(request LLMQueryRequest) (string, error) {

	decomposed_query_request, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(SubquestionInstruction)}, {Role: "user", Content: request.Input}})
	if err != nil {
		return "", err
	}
	decomposed_query, err := ParseSubQuestions(decomposed_query_request)
	if err != nil{
		return "", err
	}

	if len(decomposed_query) == 0 {
		decomposed_query = []string{request.Input}
	}

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

	if len(data_sources) == 1 && data_sources[0] == "sql" {
		return sqlFlow(request, question)
	}

	data, err := SourceData(request.Model, data_sources, question, request.SearchLimit)
	if err != nil {
		return "", err
	}
	answer, err := QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
	if err != nil {
		return "", err
	}

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
	}

	return answer, nil
}

func sqlFlow(request LLMQueryRequest, question string) (string, error) {
	data, err := SourceData(request.Model, []string{"sql"}, question, request.SearchLimit)

	if err != nil {
		return "", err
	}

	return QueryOllama(request.Model, []ChatMessage{{Role: "user", Content: string(GameFIGeniusInstruction)}, {Role: "user", Content: "DATA:\n" + data + "QUERY:\n" + question}})
}
