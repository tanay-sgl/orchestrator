package database

import (
	"context"
	"encoding/json"
	"fmt"
	"orchestrator/internal/models"
	"os"
	"reflect"
	"strings"

	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pgvector/pgvector-go"
)

func CreateDatabaseConnectionFromEnv() (*pg.DB, error) {
	db := pg.Connect(&pg.Options{
		Addr:     os.Getenv("TIMESCALE_ADDRESS"),
		User:     os.Getenv("TIMESCALE_USER"),
		Password: os.Getenv("TIMESCALE_PASSWORD"),
		Database: os.Getenv("TIMESCALE_DATABASE"),
	})

	err := db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func CreatePostgresDSN() string {
    return fmt.Sprintf("postgres://%s:%s@%s/%s",
        os.Getenv("TIMESCALE_USER"),
        os.Getenv("TIMESCALE_PASSWORD"),
        os.Getenv("TIMESCALE_ADDRESS"),
        os.Getenv("TIMESCALE_DATABASE"))
}

func GetTableSchemaAsString() (string, error) {

	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", err
	}
	defer db.Close()

    var tables []struct {
        TableName string
        Columns   string
    }

    _, err = db.Query(&tables, `
        SELECT table_name, 
               string_agg(column_name || ' ' || data_type, ', ' ORDER BY ordinal_position) AS columns
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        GROUP BY table_name
        ORDER BY table_name
    `)
    if err != nil {
        return "", err
    }

    var schema strings.Builder
    for _, table := range tables {
        schema.WriteString(fmt.Sprintf("%s: %s\n", table.TableName, table.Columns))
    }
    return schema.String(), nil
}

func GetRowAsAString(request models.RowEmbeddingsRequest) (string, error) {
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
	var db *pg.DB
	db, err = CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", err
	}
	defer db.Close()

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

func InsertDocumentEmbedding(db *pg.DB, request models.DocumentEmbeddingsRequest, content string, embedding pgvector.Vector) error {
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

func InsertRowEmbedding(request models.RowEmbeddingsRequest, embedding pgvector.Vector) error {
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

	var db *pg.DB
	db, err = CreateDatabaseConnectionFromEnv()
	if err != nil {
		return err
	}
	defer db.Close()
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

func ConstructSimilarDocumentsQuery(queryEmbedding pgvector.Vector, limit int) string {
	embeddingStr := fmt.Sprintf("%v", queryEmbedding)

	query := fmt.Sprintf(`
        SELECT collection_slug, content
        FROM documents
        ORDER BY embedding <=> '%s'::vector
        LIMIT %d
    `, embeddingStr, limit)

	return query
}

// TODO handle no results gracefully
func GetSimilarRowsFromTable(tableName string, queryEmbedding pgvector.Vector, limit int) ([]map[string]interface{}, error) {
	//fmt.Printf("GetSimilarRowsFromTable: %s\n", tableName)
	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {

	}
	defer db.Close()

	var rows []json.RawMessage
	_, err = db.Query(&rows, fmt.Sprintf(`
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



func GetRecentMessages(db *pg.DB, conversationID int64, limit int) ([]Message, error) {
	//fmt.Printf("GetRecentMessages\n")
	var messages []Message
	err := db.Model(&messages).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Select()
	return messages, err
}





