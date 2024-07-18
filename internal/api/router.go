package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	db := make(map[string]string)

	router.GET("/ping", handlePing)
	router.GET("/user/:name", handleUserProfile(db))

	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"foo":  "bar",
		"manu": "123",
	}))

	authorized.POST("admin", handleAdminEndpoint(db))
	authorized.POST("generateRowEmbeddings", handleGenerateRowEmbeddings)
	authorized.POST("generateDocumentEmbeddings", handleGenerateDocumentEmbeddings)
	authorized.POST("llmQuery", handleLLMQuery)
	authorized.POST("searchDocuments", searchDocuments)

	return router
}

