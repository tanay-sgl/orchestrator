package main

import (
	"context"
	"log"
)

type RelevantData struct {
    SimilarRows     map[string][]map[string]interface{}
    SimilarDocuments []Document
}


func ProcessLLMQuery(ctx context.Context, pm *PostgresMemory, request LLMQueryRequest) (LLMQueryResponse, error) {
    // Get relevant data
    relevantData := GetRelevantData(request)

    // Load conversation history
    memoryVars, err := pm.LoadMemoryVariables(ctx, map[string]interface{}{
        "conversation_id": request.ConversationID,
    })
    if err != nil {
        return LLMQueryResponse{}, err
    }

    // Prepare the prompt
    prompt := preparePrompt(request.Input, memoryVars["history"].([]map[string]string), relevantData)

    // Call LLM
    response, err := callLLM(prompt, request.Model)
    if err != nil {
        return LLMQueryResponse{}, err
    }

    // Save the context
    err = pm.SaveContext(ctx, map[string]interface{}{
        "conversation_id": request.ConversationID,
        "input":           request.Input,
    }, map[string]interface{}{
        "output": response,
    })
    if err != nil {
        return LLMQueryResponse{}, err
    }

    return LLMQueryResponse{
        Result:       response,
        RelevantData: relevantData,
    }, nil
}


func GetRelevantData(request LLMQueryRequest) RelevantData {
    requestEmbedding, err := CreateEmbedding("nomic-embed-text", request.Input)
    if err != nil {
        log.Fatal(err)
    }

    db := ConnectToDatabase()
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
        SimilarRows: similarRows,
        SimilarDocuments: similarDocumentContent,
    }
}

