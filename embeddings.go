package main

import (
	"context"
	"fmt"

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

func ProcessDocumentEmbeddingsInChunks(request DocumentEmbeddingsRequest) error {
	fmt.Printf("ProcessDocumentEmbeddingsInChunks\n")
	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}
	defer db.Close()
	content, err := GetFileChunksFromCIDAsStrings(request.CID, 1000)
	fmt.Printf("Content: %v\n", content)
	if err != nil {
		return fmt.Errorf("error getting content for CID: %w", err)
	}

	for i, chunk := range content {
		if chunk == "" {
			fmt.Printf("Warning: Empty chunk at index %d\n", i)
			continue
		}

		fmt.Printf("Processing chunk %d, length: %d\n", i, len(chunk))

		embedding, err := CreateEmbedding(request.Model, chunk)
		if err != nil {
			fmt.Printf("Error creating embedding for chunk %d: %v\n", i, err)
			continue
		}

		err = InsertDocumentEmbedding(db, request, chunk, embedding)
		if err != nil {
			fmt.Printf("Error inserting document embedding for chunk %d: %v\n", i, err)
			continue
		}

		fmt.Printf("Successfully processed chunk %d\n", i)
	}

	fmt.Println("Document was chunked and embeddings processed and inserted successfully.")
	return nil
}

func ProcessRowEmbeddings(request RowEmbeddingsRequest) error {
	row, err := GetRowAsAString(request)

	if err != nil {
		return err
	}

	embedding, err := CreateEmbedding(request.Model, row)
	if err != nil {
		return err
	}

	return InsertRowEmbedding(request, embedding)
}
