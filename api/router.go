package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine) {
	// Configure CORS to allow all methods and all origins
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"*"}
	config.AllowCredentials = true
	config.OptionsResponseStatusCode = http.StatusOK
	router.Use(cors.New(config))

	// Define a route and its handler
	router.GET("/", IndexHandler)
	// openai model
	router.GET("/v1/models", ModelListHandler)
	router.GET("/v1/models/:model", ModelRetrieveHandler)

	// openai chat
	router.POST("/v1/chat/completions", ChatProxyHandler)

	// openai embeddings
	router.POST("/v1/embeddings", EmbeddingProxyHandler)
}
