package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"os"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pgvector/pgvector-go"
)

func ConnectToDatabase() *pg.DB {
	return pg.Connect(&pg.Options{
		Addr:     os.Getenv("TIMESCALE_ADDRESS"),
		User:     os.Getenv("TIMESCALE_USER"),
		Password: os.Getenv("TIMESCALE_PASSWORD"),
		Database: os.Getenv("TIMESCALE_DATABASE"),
	})
}

func GetRowAsAString(db *pg.DB, request RowEmbeddingsRequest) (string, error) {
	// Get the struct type for the table
	structType := GetTableStruct(request.Table)
	if structType == nil {
		return "", fmt.Errorf("unknown table: %s", request.Table)
	}

	// Create a new instance of the struct
	row := reflect.New(structType).Interface()

	// Unmarshal the JSON primary key
	var primaryKeys map[string]interface{}
	err := json.Unmarshal(request.RowPrimaryKey, &primaryKeys)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal primary keys: %v", err)
	}

	// Create a query
	query := db.Model(row).ExcludeColumn("embedding")

	// Add WHERE clauses for each primary key
	for key, value := range primaryKeys {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	// Execute the query
	err = query.Select()
	if err != nil {
		return "", fmt.Errorf("failed to query %s: %v", request.Table, err)
	}

	// Convert the result to a JSON string
	result, err := json.Marshal(row)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %v", err)
	}

	return string(result), nil
}

func InsertDocumentEmbedding(db *pg.DB, request DocumentEmbeddingsRequest, content string, embedding pgvector.Vector) error {
	embeddingFloat32 := embedding.Slice()

	doc := &Document{
		CollectionSlug: request.CollectionSlug,
		CID:            request.CID,
		Content:        content,
		Embedding:      embeddingFloat32,
		EventTimestamp: time.Now().UTC(),
	}

	_, err := db.Model(doc).Insert()
	if err != nil {
		return fmt.Errorf("failed to insert document embedding: %w", err)
	}

	return nil
}

func InsertRowEmbedding(db *pg.DB, request RowEmbeddingsRequest, embedding pgvector.Vector) error {
	// Get the struct type for the table
	structType := GetTableStruct(request.Table)
	if structType == nil {
		return fmt.Errorf("unknown table: %s", request.Table)
	}

	// Create a new instance of the struct
	row := reflect.New(structType).Interface()

	// Unmarshal the JSON primary key
	var primaryKeys map[string]interface{}
	err := json.Unmarshal(request.RowPrimaryKey, &primaryKeys)
	if err != nil {
		return fmt.Errorf("failed to unmarshal primary keys: %v", err)
	}

	// Create a query
	query := db.Model(row).Set("embedding = ?", embedding)

	// Add WHERE clauses for each primary key
	for key, value := range primaryKeys {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	// Execute the update
	_, err = query.Update()
	if err != nil {
		return fmt.Errorf("failed to update %s: %v", request.Table, err)
	}

	return nil
}

func GetSimilarDocuments(db *pg.DB, queryEmbedding pgvector.Vector, limit int) ([]Document, error) {
	fmt.Printf("GetSimilarDocuments\n")
	var documents []Document
	_, err := db.Query(&documents, `
        SELECT collection_slug, cid, content
        FROM documents
        ORDER BY embedding <=> ?
        LIMIT ?
    `, queryEmbedding, limit)

	return documents, err
}

// TODO handle no results gracefully
func GetSimilarRowsFromTable(db *pg.DB, tableName string, queryEmbedding pgvector.Vector, limit int) ([]map[string]interface{}, error) {
	fmt.Printf("GetSimilarRowsFromTable: %s\n", tableName)

	// Whitelist of allowed table names to prevent SQL injection
	allowedTables := map[string]bool{"collection": true, "nft": true, "contract": true} // Add all your table names here
	if !allowedTables[tableName] {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var rows []json.RawMessage
	_, err := db.Query(&rows, fmt.Sprintf(`
        SELECT jsonb_object_agg(
            key,
            CASE 
                WHEN key = 'embedding' THEN NULL 
                ELSE value
            END
        ) as row
        FROM (
            SELECT *
            FROM %s
            ORDER BY embedding <=> ?::vector
            LIMIT ?
        ) r,
        LATERAL jsonb_each(to_jsonb(r))
        GROUP BY r
    `, tableName), queryEmbedding.Slice(), limit)

	if err != nil {
		return nil, fmt.Errorf("error querying similar rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no similar rows found in table %s", tableName)
	}

	// Unmarshal the JSON in each row
	var result []map[string]interface{}
	for _, row := range rows {
		var unmarshalled map[string]interface{}
		if err := json.Unmarshal(row, &unmarshalled); err != nil {
			return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
		}
		result = append(result, unmarshalled)
	}

	return result, nil
}

func GetAllSimilarRowsFromDB(db *pg.DB, tables []string, queryEmbedding pgvector.Vector, limitPerTable int) (map[string][]map[string]interface{}, error) {
	fmt.Printf("GetAllSimilarRowsFromDB\n")
	results := make(map[string][]map[string]interface{})

	for _, tableName := range tables {
		rows, err := GetSimilarRowsFromTable(db, tableName, queryEmbedding, limitPerTable)
		if err != nil {
			fmt.Printf("error searching table %s: %w", tableName, err)
		}
		results[tableName] = rows
	}

	return results, nil
}

func GetRecentMessages(db *pg.DB, conversationID int64, limit int) ([]Message, error) {
	fmt.Printf("GetRecentMessages\n")
	var messages []Message
	err := db.Model(&messages).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Select()
	return messages, err
}
