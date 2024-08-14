package llm

import (
	"fmt"
	"orchestrator/internal/database"
	"orchestrator/internal/models"
	"strings"
)

func QueryUserRequestForSimilarDocuments(request models.LLMRAGQueryRequest) (string, error) {
	db, err := database.CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", err
	}
	defer db.Close()
	var result strings.Builder
	query_embedding, err := CreateEmbedding(request.Model, request.Input)
	if err != nil {
		return "", err
	}
	similarDocuments, err := database.GetSimilaritySearchDocuments(db, query_embedding, request.SearchLimit)
	if err != nil {
		return "", err
	}
	for i, doc := range similarDocuments {
		result.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		result.WriteString(fmt.Sprintf("Collection Slug: %s\n", doc.CollectionSlug))
		result.WriteString(fmt.Sprintf("CID: %s\n", doc.CID))
		result.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		result.WriteString("\n")
	}

	return result.String(), nil
}
