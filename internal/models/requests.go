package models

import "encoding/json"

// Document Embeddings are sent to vector database
type DocumentEmbeddingsRequest struct {
	CID            string `json:"cid"`
	CollectionSlug string `json:"collection_slug"`
	Model          string `json:"model,omitempty" default:"nomic-embed-text"`
}

// Async requests to vectorize rows as data trickles in
// Primary keys ordered by database order
type RowEmbeddingsRequest struct {
	RowPrimaryKey json.RawMessage `json:"row_primary_key"`
	Model         string          `json:"model,omitempty" default:"nomic-embed-text"`
	Table         string          `json:"table"`
}

// LLMSimpleQueryRequest represents a simple LLM query without RAG
type LLMSimpleQueryRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model,omitempty" default:"default-model"`
	ConversationID int64  `json:"conversation_id,omitempty"`
}

// LLMRAGQueryRequest represents an LLM query with RAG
type LLMRAGQueryRequest struct {
	Input          string   `json:"input"`
	Model          string   `json:"model,omitempty" default:"default-model"`
	SearchLimit    int      `json:"search_limit,omitempty" default:"5"`
	DataSources    []string `json:"data_sources,omitempty"`
	ConversationID int64    `json:"conversation_id,omitempty"`
}

// LLMSQLQueryRequest represents an LLM query for SQL generation
type LLMSQLQueryRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model,omitempty" default:"default-model"`
	ConversationID int64  `json:"conversation_id,omitempty"`
}
