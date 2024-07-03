package main

import "encoding/json"

// Document Embeddings are sent to vector database; response unnecessary
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

type LLMQueryRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model,omitempty" default:"default-model"`
	SearchLimit    int    `json:"search_limit,omitempty" default:"5"`
	ConversationID int64  `json:"conversation_id,omitempty"`
}

type LLMQueryResponse struct {
	Result       string
	RelevantData RelevantData
}
