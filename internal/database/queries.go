package database

import (
	"encoding/json"
	"fmt"
	"orchestrator/internal/models"
	"reflect"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pgvector/pgvector-go"
)


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


func GetOrCreateConversation(db *pg.DB, conversationID int64, title string) (*Conversation, error) {
    conversation := &Conversation{ID: conversationID}
    err := db.Model(conversation).WherePK().Select()
    if err == pg.ErrNoRows {
        // Conversation doesn't exist, create a new one
        conversation = &Conversation{
            Title: title,
        }
        _, err = db.Model(conversation).Insert()
        if err != nil {
            return nil, fmt.Errorf("error creating new conversation: %w", err)
        }
    } else if err != nil {
        return nil, fmt.Errorf("error retrieving conversation: %w", err)
    }
    return conversation, nil
}

func SaveMessages(db *pg.DB, conversationID int64, messages []Message, title string) error {
    conversation, err := GetOrCreateConversation(db, conversationID, title)
    if err != nil {
        return err
    }

    for _, msg := range messages {
        msg.ConversationID = conversation.ID
        msg.Conversation = conversation
        _, err := db.Model(&msg).Insert()
        if err != nil {
            return fmt.Errorf("error inserting message: %w", err)
        }
    }
    return nil
}


func GetSimilaritySearchDocuments(db *pg.DB, embedding pgvector.Vector, searchLimit int) ([]Document, error) {
    var documents []Document
    query := ConstructSimilarDocumentsQuery(embedding, searchLimit)
    _, err := db.Query(&documents, query)
    return documents, err
}


func GetAllSimilarRowsFromDB(db *pg.DB, embedding pgvector.Vector, searchLimit int) (map[string][]map[string]interface{}, error) {
    results := make(map[string][]map[string]interface{})
    for _, tableName := range TableNames {
        rows, err := GetSimilarRowsFromTable(tableName, embedding, searchLimit)
        if err != nil {
            return nil, fmt.Errorf("error searching table %s: %w", tableName, err)
        }
        results[tableName] = rows
    }
    return results, nil
}

func ExecuteSQLQuery(db *pg.DB, query string) ([]map[string]interface{}, error) {
    var result []map[string]interface{}
    _, err := db.Query(&result, query)
    if err != nil {
        return nil, fmt.Errorf("error executing SQL query: %w", err)
    }
    return result, nil
}


func SaveConversationAsMessages(db *pg.DB, conversationID int64, userInput, assistantResponse string) error {
    if conversationID == 0 {
        return nil
    }

    title := fmt.Sprintf("Query: %s", userInput)
    err := SaveMessages(db, conversationID, []Message{
        {Role: "user", Content: userInput},
        {Role: "assistant", Content: assistantResponse},
    }, title)
    if err != nil {
        return fmt.Errorf("error saving conversation: %w", err)
    }

    return nil
}