package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func createPostgresDSN() string {
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

func GetRowAsAString(request RowEmbeddingsRequest) (string, error) {
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

func InsertRowEmbedding(request RowEmbeddingsRequest, embedding pgvector.Vector) error {
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

func constructSimilarDocumentsQuery(queryEmbedding pgvector.Vector, limit int) string {
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

func GetAllSimilarRowsFromDB(request LLMQueryRequest) (map[string][]map[string]interface{}, error) {
	//fmt.Printf("GetAllSimilarRowsFromDB\n")
	requestEmbedding, err := CreateEmbedding("llama3", request.Input)
	if err != nil {
		//log.Fatal(err)
	}

	results := make(map[string][]map[string]interface{})

	for _, tableName := range TableNames {
		rows, err := GetSimilarRowsFromTable(tableName, requestEmbedding, request.SearchLimit)
		if err != nil {
			fmt.Printf("error searching table %s: %w", tableName, err)
		}
		results[tableName] = rows
	}

	return results, nil
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

func GetSimilaritySearchDocuments(request LLMQueryRequest) ([]Document, error) {
	//fmt.Printf("Creating Embedding for: " + request.Input + "\n")
	requestEmbedding, err := CreateEmbedding(request.Model, request.Input)
	if err != nil {
		return nil, fmt.Errorf("error creating embedding: %w", err)
	}

	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	var documents []Document

	query := constructSimilarDocumentsQuery(requestEmbedding, request.SearchLimit)
	_, err = db.Query(&documents, query)

	return documents, err
}

func GetSimilaritySearchAll(request LLMQueryRequest) (RelevantData, error) {
	similarRows, err := GetAllSimilarRowsFromDB(request)

	similarDocumentContent, err := GetSimilaritySearchDocuments(request)
	if err != nil {
		//log.Fatal(err)
	}

	return RelevantData{
		SimilarRows:      similarRows,
		SimilarDocuments: similarDocumentContent,
	}, nil
}

func SourceData(model string, data_sources []string, question string, search_limit int) (string, error) {
	//fmt.Printf("QUESTON: %s\n", question)
	relevant_data := RelevantData{}
	db, err := CreateDatabaseConnectionFromEnv()
	if err != nil {
		return "", fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	for _, data_source := range data_sources {
		switch data_source {
		case "documents":
			similarDocuments, err := GetSimilaritySearchDocuments(LLMQueryRequest{
				Model:       model,
				Input:       question,
				SearchLimit: search_limit,
			})
			if err != nil {
				continue
			}
			//fmt.Printf("similarDocuments: %v\n", similarDocuments)
			relevant_data.SimilarDocuments = similarDocuments
		case "sql":
			fmt.Printf("SQL MODE")
			//Identify collection_slug(s)
			requestEmbedding, err := CreateEmbedding(model, question)
			if err != nil {
				//log.Fatal(err)
			}

			collection_string, err := GetSimilarRowsFromTable("collection", requestEmbedding, search_limit)
			if err != nil {
				//log.Default().Println(err)
			}

			// Convert the result to a JSON string
			resultRow, err := json.Marshal(collection_string)
			if err != nil {
				return "", fmt.Errorf("failed to marshal result: %v", err)
			}

			sql_request, err := QueryOllama(model, []ChatMessage{
				{Role: "user", Content: string(SQLInstruction)},
				{Role: "user", Content: "METADATA:\n" + string(resultRow) + "\nQUERY:\n"},
				{Role: "user", Content: question}})
			if err != nil {
				//log.Default().Println(err)
			}

			sanitized_sql_request, err := SanitizeAndParseSQLQuery(sql_request)
			if err != nil {
				//log.Default().Println(err)
			}
			fmt.Println("Sanitized SQL instruction: " + sanitized_sql_request)
			var result []map[string]interface{}
			_, err = db.Query(&result, sanitized_sql_request)
			if err != nil {
				return "", fmt.Errorf("error executing SQL query: %w", err)
			}

			// Add the SQL result to the relevant_data
			relevant_data.SimilarRows = map[string][]map[string]interface{}{
				"sql_result": result,
			}
		case "default":

			all_similar, err := GetSimilaritySearchAll(LLMQueryRequest{
				Model:       model,
				Input:       question,
				SearchLimit: search_limit,
			})

			if err != nil {
				return "", fmt.Errorf("error getting default data: %w", err)
			}

			if all_similar.SimilarDocuments != nil {
				relevant_data.SimilarDocuments = append(relevant_data.SimilarDocuments, all_similar.SimilarDocuments...)
			}
			for table, rows := range all_similar.SimilarRows {
				if relevant_data.SimilarRows == nil {
					relevant_data.SimilarRows = make(map[string][]map[string]interface{})
				}
				relevant_data.SimilarRows[table] = append(relevant_data.SimilarRows[table], rows...)
			}
		case "NA":
			return QueryOllama(model, []ChatMessage{{Role: "user", Content: question}})
		}
	}
	return ParseRelevantData(relevant_data)
}
