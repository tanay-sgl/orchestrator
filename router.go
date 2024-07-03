package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	var db = make(map[string]string)
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong! orchestrator is at your command")
	})

	router.GET("/user/:name", func(c *gin.Context) {
		user := c.Params.ByName("name")
		value, ok := db[user]
		if ok {
			c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
		}
	})

	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"foo":  "bar", // user:foo password:bar
		"manu": "123", // user:manu password:123
	}))

	authorized.POST("admin", func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)

		// Parse JSON
		var json struct {
			Value string `json:"value" binding:"required"`
		}

		if c.Bind(&json) == nil {
			db[user] = json.Value
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	authorized.POST("generateRowEmbeddings", func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)

		//TODO change this
		if user != "foo" {
			c.JSON(http.StatusOK, gin.H{"status": "error"})
		}

		var request RowEmbeddingsRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process the request asynchronously
		go ProcessRowEmbeddings(request)

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Processing started"})
	})

	authorized.POST("generateDocumentEmbeddings", func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)

		// TODO: Implement proper authentication
		if user != "foo" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
			return
		}

		var request DocumentEmbeddingsRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
			return
		}

		// Process the request asynchronously
		go func() {
			err := ProcessDocumentEmbeddingsInChunks(request)
			if err != nil {
				// Log the error, as we can't return it to the client in an asynchronous operation
				log.Printf("Error processing document embeddings: %v", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Processing started"})
	})

	
		authorized.POST("llmQuery", func(c *gin.Context) {
			user := c.MustGet(gin.AuthUserKey).(string)
	
			// TODO: Implement proper authentication
			if user != "foo" {
				c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
				return
			}
	
			var request LLMQueryRequest
			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
				return
			}
	
			fmt.Print("Received request: \n")
			fmt.Printf("%+v\n", request)
	
			// Process the request synchronously
			response, err := ProcessLLMQuery(request)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
				return
			}
	
			c.JSON(http.StatusOK, gin.H{"status": "success", "response": response})
		})

	return router
}
