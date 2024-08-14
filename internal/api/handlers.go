package api

import (
	"fmt"
	"log"
	"net/http"
	"orchestrator/internal/llm"
	"orchestrator/internal/models"

	"github.com/gin-gonic/gin"
)

func handlePing(c *gin.Context) {
	c.String(http.StatusOK, "pong! orchestrator is at your command")
}

func handleUserProfile(db map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.Param("name")
		value, ok := db[user]
		if ok {
			c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
		}
	}
}

func handleAdminEndpoint(db map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)
		var json struct {
			Value string `json:"value" binding:"required"`
		}

		if c.Bind(&json) == nil {
			db[user] = json.Value
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	}
}

func handleGenerateRowEmbeddings(c *gin.Context) {
	user := c.MustGet(gin.AuthUserKey).(string)
	if user != "foo" {
		c.JSON(http.StatusOK, gin.H{"status": "error"})
		return
	}

	var request models.RowEmbeddingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go llm.ProcessRowEmbeddings(request)

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Processing started"})
}

func handleGenerateDocumentEmbeddings(c *gin.Context) {
	fmt.Printf("handleGenerateDocumentEmbeddings\n")
	user := c.MustGet(gin.AuthUserKey).(string)
	if user != "foo" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}

	var request models.DocumentEmbeddingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	go func() {
		err := llm.ProcessDocumentEmbeddingsInChunks(request)
		if err != nil {
			log.Printf("Error processing document embeddings: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Processing started"})
}

func handleLLMSimpleQuery(c *gin.Context) {
	var request models.LLMSimpleQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := llm.ProcessLLMSimpleQuery(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleLLMSQLQuery(c *gin.Context) {
	var request models.LLMSQLQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, err := llm.QueryUserRequestAsSQL(request.Model, request.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleLLMRAGQuerySingleNode(c *gin.Context) {
	var request models.LLMRAGQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := llm.ProcessLLMRAGQuerySingleNode(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleLLMRAGQueryMultiNode(c *gin.Context) {
	var request models.LLMRAGQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := llm.ProcessLLMRAGQueryMultiNode(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}
