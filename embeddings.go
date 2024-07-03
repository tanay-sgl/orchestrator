package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pgvector/pgvector-go"
	"github.com/tmc/langchaingo/llms/ollama"
)

func CreateEmbedding(requestModel string, content string) (pgvector.Vector, error) {
	fmt.Printf("Creating embedding...\n")
	fmt.Printf(requestModel + "\n")
	fmt.Printf(content + "\n")
	model, err := ollama.New(ollama.WithModel(requestModel))
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to create ollama model: %w", err)
	}

	contentSlice := []string{content}
	fmt.Printf("Content slice: %v\n", contentSlice)

	embeddingMatrix, err := model.CreateEmbedding(context.Background(), contentSlice)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Flatten the 2D embedding matrix into a 1D slice
	embedding := make([]float32, 0, len(embeddingMatrix)*len(embeddingMatrix[0]))
	for _, row := range embeddingMatrix {
		embedding = append(embedding, row...)
	}
	return pgvector.NewVector(embedding), nil
}

func ProcessDocumentEmbeddingsInChunks(request DocumentEmbeddingsRequest) error {
	db := ConnectToDatabase()
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

func ProcessRowEmbeddings(request RowEmbeddingsRequest) {
	db := ConnectToDatabase()
	defer db.Close()

	row, err := GetRowAsAString(db, request)

	if err != nil {
		log.Fatal(err)
	}

	embedding, err := CreateEmbedding(request.Model, row)
	if err != nil {
		log.Fatal(err)
	}
	InsertRowEmbedding(db, request, embedding)

	fmt.Println("Row embedding processed and updated successfully.")
}
