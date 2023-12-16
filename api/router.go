package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine) {
	// Configure CORS to allow all methods and all origins
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"POST"}
	router.Use(cors.New(config))

	// Define a route and its handler
	router.POST("/v1/chat/completions", ChatHandler)
}
