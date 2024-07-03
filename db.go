package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	var documents []Document
	_, err := db.Query(&documents, `
        SELECT collection_slug, cid, content
        FROM documents
        ORDER BY embedding <=> ?
        LIMIT ?
    `, queryEmbedding, limit)

	return documents, err
}

func GetSimilarRowsFromTable(db *pg.DB, tableName string, queryEmbedding pgvector.Vector, limit int) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	_, err := db.Query(&rows, fmt.Sprintf(`
        SELECT (r.*)::jsonb - 'embedding' as row
        FROM (
            SELECT *
            FROM %s
            ORDER BY embedding <=> ?
            LIMIT ?
        ) r
    `, tableName), queryEmbedding, limit)

	return rows, err
}


func GetAllSimilarRowsFromDB(db *pg.DB, tables []string, queryEmbedding pgvector.Vector, limitPerTable int) (map[string][]map[string]interface{}, error) {
	results := make(map[string][]map[string]interface{})

	for _, tableName := range tables {
		rows, err := GetSimilarRowsFromTable(db, tableName, queryEmbedding, limitPerTable)
		if err != nil {
			return nil, fmt.Errorf("error searching table %s: %w", tableName, err)
		}
		results[tableName] = rows
	}

	return results, nil
}

func PrintDatabaseSchema(db *pg.DB) {
	var tables []struct {
		TableName  string
		ColumnName string
		DataType   string
	}

	_, err := db.Query(&tables, `
        SELECT 
            table_name AS table_name,
            column_name AS column_name,
            data_type AS data_type
        FROM 
            information_schema.columns
        WHERE 
            table_schema = 'public'
        ORDER BY 
            table_name, ordinal_position
    `)
	if err != nil {
		log.Fatal(err)
	}

	currentTable := ""
	for _, table := range tables {
		if table.TableName != currentTable {
			if currentTable != "" {
				fmt.Println()
			}
			currentTable = table.TableName
			fmt.Printf("Table: %s\n", currentTable)
		}
		fmt.Printf("  Column: %s, Data Type: %s\n", table.ColumnName, table.DataType)
	}
}
