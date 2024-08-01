package llm

import (
	"context"
	"fmt"
	"orchestrator/internal/database"
	"orchestrator/internal/fileprocessing"
	"orchestrator/internal/models"

	"github.com/pgvector/pgvector-go"
	"github.com/tmc/langchaingo/llms/ollama"
)

func CreateEmbedding(requestModel string, content string) (pgvector.Vector, error) {
	model, err := ollama.New(ollama.WithModel(requestModel))
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to create ollama model: %w", err)
	}

	twoDimensionalEmbedding, err := model.CreateEmbedding(context.Background(), []string{content})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to create embedding: %w", err)
	}

	oneDimensionalEmbedding := make([]float32, 0, len(twoDimensionalEmbedding)*len(twoDimensionalEmbedding[0]))
	for _, row := range twoDimensionalEmbedding {
		oneDimensionalEmbedding = append(oneDimensionalEmbedding, row...)
	}
	return pgvector.NewVector(oneDimensionalEmbedding), nil
}

func ProcessDocumentEmbeddingsInChunks(request models.DocumentEmbeddingsRequest) error {
	db, err := database.CreateDatabaseConnectionFromEnv()
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}
	defer db.Close()
	content, err := fileprocessing.GetFileChunksFromCIDAsStrings(request.CID, 1000)

	if err != nil {
		return fmt.Errorf("error getting content for CID: %w", err)
	}

	for i, chunk := range content {
		if chunk == "" {
			fmt.Printf("Warning: Empty chunk at index %d\n", i)
			continue
		}

		embedding, err := CreateEmbedding(request.Model, chunk)
		if err != nil {
			fmt.Printf("Error creating embedding for chunk %d: %v\n", i, err)
			continue
		}

		err = database.InsertDocumentEmbedding(db, request, chunk, embedding)
		if err != nil {
			fmt.Printf("Error inserting document embedding for chunk %d: %v\n", i, err)
			continue
		}
	}
	return nil
}

func ProcessRowEmbeddings(request models.RowEmbeddingsRequest) error {
	row, err := database.GetRowAsAString(request)

	if err != nil {
		return err
	}

	embedding, err := CreateEmbedding(request.Model, row)
	if err != nil {
		return err
	}

	return database.InsertRowEmbedding(request, embedding)
}
