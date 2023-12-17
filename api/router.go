package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine) {
	// Configure CORS to allow all methods and all origins
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST"}
	router.Use(cors.New(config))

	// Define a route and its handler
	// openai model
	router.GET("/v1/models", ModelListHandler)
	router.GET("/v1/models/gpt-3.5-turbo", ModelRetrieveHandler)

	// openai chat
	router.POST("/v1/chat/completions", ChatProxyHandler)
}
