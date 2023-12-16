package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/zhu327/gemini-openai-proxy/pkg/gpt"
	"github.com/zhu327/gemini-openai-proxy/pkg/util"
)

func ChatHandler(c *gin.Context) {
	// Retrieve the Authorization header value
	authorizationHeader := c.GetHeader("Authorization")
	// Declare a variable to store the OPENAI_API_KEY
	var openaiAPIKey string
	// Use fmt.Sscanf to extract the Bearer token
	_, err := fmt.Sscanf(authorizationHeader, "Bearer %s", &openaiAPIKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req openai.ChatCompletionRequest
	// Bind the JSON data from the request to the struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request message must not be empty!"})
		return
	}

	ctx := c.Request.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(openaiAPIKey))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro")
	gpt.SetGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	gpt.SetGenaiChatByOpenaiRequest(cs, req)

	prompt := genai.Text(req.Messages[len(req.Messages)-1].Content)

	if !req.Stream {
		genaiResp, err := cs.SendMessage(ctx, prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		openaiResp := gpt.GenaiResponseToOpenaiResponse(genaiResp)
		c.JSON(http.StatusOK, openaiResp)
		return
	}

	iter := cs.SendMessageStream(ctx, prompt)
	dataChan := make(chan string)
	go func() {
		respID := util.GetUUID()
		created := time.Now().Unix()

		for {
			genaiResp, err := iter.Next()
			if err == iterator.Done {
				close(dataChan)
				break
			}

			if err != nil {
				dataChan <- fmt.Sprintf(`{"error": "%s"}`, err.Error())
				close(dataChan)
				break
			}

			openaiResp := gpt.GenaiResponseToStreamComplitionResponse(genaiResp, respID, created)
			resp, _ := json.Marshal(openaiResp)
			dataChan <- string(resp)
		}
	}()

	setEventStreamHeaders(c)
	c.Stream(func(w io.Writer) bool {
		if data, ok := <-dataChan; ok {
			c.Render(-1, gpt.Event{Data: "data: " + data})
			return true
		}
		c.Render(-1, gpt.Event{Data: "data: [DONE]"})
		return false
	})
}

func setEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}
