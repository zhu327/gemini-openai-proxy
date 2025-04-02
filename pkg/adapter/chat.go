package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/zhu327/gemini-openai-proxy/pkg/util"
)

const (
	genaiRoleUser  = "user"
	genaiRoleModel = "model"
)

type GeminiAdapter struct {
	client *genai.Client
	model  string
}

func NewGeminiAdapter(client *genai.Client, model string) *GeminiAdapter {
	return &GeminiAdapter{
		client: client,
		model:  model,
	}
}

func (g *GeminiAdapter) GenerateContent(
	ctx context.Context,
	req *ChatCompletionRequest,
	messages []*genai.Content,
) (*openai.ChatCompletionResponse, error) {
	// Add 'models/' prefix if not already present
	modelName := g.model
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}
	model := g.client.GenerativeModel(modelName)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatHistory(cs, messages)

	genaiResp, err := cs.SendMessage(ctx, messages[len(messages)-1].Parts...)
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) {
			if apiErr.Code == http.StatusTooManyRequests {
				return nil, errors.Wrap(&openai.APIError{
					Code:    http.StatusTooManyRequests,
					Message: err.Error(),
				}, "genai send message error")
			}
		} else {
			log.Printf("Error is not of type *googleapi.Error: %v\n", err)
		}
		return nil, errors.Wrap(err, "genai send message error")
	}
	openaiResp := genaiResponseToOpenaiResponse(g.model, genaiResp)
	return &openaiResp, nil
}

func (g *GeminiAdapter) GenerateStreamContent(
	ctx context.Context,
	req *ChatCompletionRequest,
	messages []*genai.Content,
) (<-chan string, error) {
	modelName := g.model
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}
	model := g.client.GenerativeModel(modelName)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatHistory(cs, messages)

	iter := cs.SendMessageStream(ctx, messages[len(messages)-1].Parts...)

	dataChan := make(chan string)
	go handleStreamIter(g.model, iter, dataChan)

	return dataChan, nil
}

func handleStreamIter(model string, iter *genai.GenerateContentResponseIterator, dataChan chan string) {
	defer close(dataChan)

	respID := util.GetUUID()
	created := time.Now().Unix()

	// For character-by-character streaming
	var textBuffer string

	// Counter for character-by-character streaming - increased for better performance
	sentenceLength := 1000
	charCount := 0

	// Function to send a single character with proper formatting
	sendCharacter := func(char string) {
		openaiResp := &CompletionResponse{
			ID:      fmt.Sprintf("chatcmpl-%s", respID),
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   GetMappedModel(model),
			Choices: []CompletionChoice{
				{
					Index: 0,
					Delta: struct {
						Content   string            `json:"content,omitempty"`
						Role      string            `json:"role,omitempty"`
						ToolCalls []openai.ToolCall `json:"tool_calls,omitempty"`
					}{
						Content: char,
					},
				},
			},
		}
		resp, _ := json.Marshal(openaiResp)
		dataChan <- string(resp)
	}

	// Function to send entire text at once (for finish conditions)
	sendFullText := func(text string) {
		if text == "" {
			return
		}
		openaiResp := &CompletionResponse{
			ID:      fmt.Sprintf("chatcmpl-%s", respID),
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   GetMappedModel(model),
			Choices: []CompletionChoice{
				{
					Index: 0,
					Delta: struct {
						Content   string            `json:"content,omitempty"`
						Role      string            `json:"role,omitempty"`
						ToolCalls []openai.ToolCall `json:"tool_calls,omitempty"`
					}{
						Content: text,
					},
				},
			},
		}
		resp, _ := json.Marshal(openaiResp)
		dataChan <- string(resp)
	}

	for {
		genaiResp, err := iter.Next()
		if err == iterator.Done {
			// Send any remaining text when done - all at once
			if len(textBuffer) > 0 {
				// Send all remaining text at once when done
				sendFullText(textBuffer)
			}
			break
		}

		if err != nil {
			log.Printf("genai get stream message error %v\n", err)

			// Check for context cancellation
			if errors.Is(err, context.Canceled) {
				log.Printf("Context was canceled by client")
				apiErr := openai.APIError{
					Code:    http.StatusRequestTimeout,
					Message: "Request was canceled",
					Type:    "canceled_error",
				}
				resp, _ := json.Marshal(apiErr)
				dataChan <- string(resp)
				break
			}

			// Check for rate limit errors
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests {
				log.Printf("Rate limit exceeded: %v\n", err)
				rateLimitErr := openai.APIError{
					Code:    http.StatusTooManyRequests,
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				}
				resp, _ := json.Marshal(rateLimitErr)
				dataChan <- string(resp)
				break
			}

			// Handle other errors
			generalErr := openai.APIError{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
				Type:    "internal_server_error",
			}
			resp, _ := json.Marshal(generalErr)
			dataChan <- string(resp)
			break
		}

		// Process each candidate's text content
		for _, candidate := range genaiResp.Candidates {
			if candidate.Content == nil {
				continue
			}

			// Check if this is the last message with a finish reason
			isLastMessage := candidate.FinishReason > genai.FinishReasonStop

			for _, part := range candidate.Content.Parts {
				switch pp := part.(type) {
				case genai.Text:
					text := string(pp)
					if isLastMessage {
						// If this is the last message, collect the text in buffer
						textBuffer += text
					} else if charCount < sentenceLength {
						// Stream character by character until we reach sentenceLength
						for i, char := range text {
							if charCount < sentenceLength {
								sendCharacter(string(char))
								// No delay between characters for faster streaming
								charCount++
							} else {
								// Once we've reached sentenceLength, send the rest of this text at once
								remaining := text[i:]
								if remaining != "" {
									sendFullText(remaining)
								}
								break
							}
						}

					} else {
						// For subsequent chunks after sentenceLength, send the entire text at once
						sendFullText(text)
					}
				case genai.FunctionCall:
					// Handle function calls as before
					openaiResp := genaiResponseToStreamCompletionResponse(model, genaiResp, respID, created)
					resp, _ := json.Marshal(openaiResp)
					dataChan <- string(resp)
				}
			}
		}

		// Send finish reason if present
		if len(genaiResp.Candidates) > 0 && genaiResp.Candidates[0].FinishReason > genai.FinishReasonStop {
			// Send any accumulated text all at once
			if len(textBuffer) > 0 {
				sendFullText(textBuffer)
			}

			// Send the finish reason
			for _, candidate := range genaiResp.Candidates {
				if candidate.FinishReason > genai.FinishReasonStop {
					openaiFinishReason := string(convertFinishReason(candidate.FinishReason))
					openaiResp := &CompletionResponse{
						ID:      fmt.Sprintf("chatcmpl-%s", respID),
						Object:  "chat.completion.chunk",
						Created: created,
						Model:   GetMappedModel(model),
						Choices: []CompletionChoice{
							{
								Index: 0,
								Delta: struct {
									Content   string            `json:"content,omitempty"`
									Role      string            `json:"role,omitempty"`
									ToolCalls []openai.ToolCall `json:"tool_calls,omitempty"`
								}{
									// Empty content for finish reason message
								},
								FinishReason: &openaiFinishReason,
							},
						},
					}
					resp, _ := json.Marshal(openaiResp)
					dataChan <- string(resp)
					break
				}
			}
			break
		}
	}
}

func genaiResponseToStreamCompletionResponse(model string, genaiResp *genai.GenerateContentResponse, respID string, created int64) *CompletionResponse {
	resp := CompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", respID),
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   GetMappedModel(model),
		Choices: make([]CompletionChoice, 0, len(genaiResp.Candidates)),
	}

	count := 0
	toolCalls := make([]openai.ToolCall, 0)

	for _, candidate := range genaiResp.Candidates {
		parts := candidate.Content.Parts
		for _, part := range parts {
			index := count
			switch pp := part.(type) {
			case genai.Text:
				choice := CompletionChoice{
					Index: index,
				}
				choice.Delta.Content = string(pp)

				if candidate.FinishReason > genai.FinishReasonStop {
					log.Printf("genai message finish reason %s\n", candidate.FinishReason.String())
					openaiFinishReason := string(convertFinishReason(candidate.FinishReason))
					choice.FinishReason = &openaiFinishReason
				}

				resp.Choices = append(resp.Choices, choice)
			case genai.FunctionCall:
				args, _ := json.Marshal(pp.Args)
				toolCalls = append(toolCalls, openai.ToolCall{
					Index:    genai.Ptr(int(index)),
					ID:       fmt.Sprintf("%s-%d", pp.Name, index),
					Type:     openai.ToolTypeFunction,
					Function: openai.FunctionCall{Name: pp.Name, Arguments: string(args)},
				})
			}
			count++
		}
	}

	if len(toolCalls) > 0 {
		choice := CompletionChoice{
			Index: 0,
		}
		// For tool calls, we need to set a special finish reason
		openaiFinishReason := string(openai.FinishReasonToolCalls)
		choice.FinishReason = &openaiFinishReason

		// Add the tool calls to the response
		toolCallsJSON, _ := json.Marshal(toolCalls)
		choice.Delta.Content = string(toolCallsJSON)

		resp.Choices = append(resp.Choices, choice)
	}

	return &resp
}

func genaiResponseToOpenaiResponse(model string, genaiResp *genai.GenerateContentResponse) openai.ChatCompletionResponse {
	resp := openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", util.GetUUID()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   GetMappedModel(model),
		Choices: make([]openai.ChatCompletionChoice, 0, len(genaiResp.Candidates)),
	}

	if genaiResp.UsageMetadata != nil {
		resp.Usage = openai.Usage{
			PromptTokens:     int(genaiResp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(genaiResp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(genaiResp.UsageMetadata.TotalTokenCount),
		}
	}

	for i, candidate := range genaiResp.Candidates {
		toolCalls := make([]openai.ToolCall, 0)
		var content string

		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			for j, part := range candidate.Content.Parts {
				switch pp := part.(type) {
				case genai.Text:
					content = string(pp)
				case genai.FunctionCall:
					args, _ := json.Marshal(pp.Args)
					toolCalls = append(toolCalls, openai.ToolCall{
						Index:    genai.Ptr(j),
						ID:       fmt.Sprintf("%s-%d", pp.Name, j),
						Type:     openai.ToolTypeFunction,
						Function: openai.FunctionCall{Name: pp.Name, Arguments: string(args)},
					})
				}
			}
		}

		choice := openai.ChatCompletionChoice{
			Index:        i,
			FinishReason: convertFinishReason(candidate.FinishReason),
		}

		if len(toolCalls) > 0 {
			choice.Message = openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				ToolCalls: toolCalls,
			}
			choice.FinishReason = openai.FinishReasonToolCalls
		} else {
			choice.Message = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: content,
			}
		}

		resp.Choices = append(resp.Choices, choice)
	}
	return resp
}

func convertFinishReason(reason genai.FinishReason) openai.FinishReason {
	openaiFinishReason := openai.FinishReasonStop
	switch reason {
	case genai.FinishReasonMaxTokens:
		openaiFinishReason = openai.FinishReasonLength
	case genai.FinishReasonSafety, genai.FinishReasonRecitation:
		openaiFinishReason = openai.FinishReasonContentFilter
	}
	return openaiFinishReason
}

func setGenaiChatHistory(cs *genai.ChatSession, messages []*genai.Content) {
	cs.History = make([]*genai.Content, 0, len(messages))
	if len(messages) > 1 {
		cs.History = append(cs.History, messages[:len(messages)-1]...)
	}

	if len(cs.History) != 0 && cs.History[len(cs.History)-1].Role != genaiRoleModel {
		cs.History = append(cs.History, &genai.Content{
			Parts: []genai.Part{
				genai.Text(""),
			},
			Role: genaiRoleModel,
		})
	}
}

func setGenaiModelByOpenaiRequest(model *genai.GenerativeModel, req *ChatCompletionRequest) {
	if req.MaxTokens != 0 {
		model.MaxOutputTokens = &req.MaxTokens
	}
	if req.Temperature != 0 {
		model.Temperature = &req.Temperature
	}
	if req.TopP != 0 {
		model.TopP = &req.TopP
	}
	if len(req.Stop) != 0 {
		model.StopSequences = req.Stop
	}

	// Set response format if specified
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json" {
		model.ResponseMIMEType = "application/json"
	}

	// Configure tools if provided
	if len(req.Tools) > 0 {
		tools := convertOpenAIToolsToGenAI(req.Tools)
		model.Tools = tools

		// Configure tool choice/function calling mode
		model.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{},
		}

		switch v := req.ToolChoice.(type) {
		case string:
			if v == "none" {
				model.ToolConfig.FunctionCallingConfig.Mode = genai.FunctionCallingNone
			} else if v == "auto" {
				model.ToolConfig.FunctionCallingConfig.Mode = genai.FunctionCallingAuto
			}
		case map[string]interface{}:
			if funcObj, ok := v["function"]; ok {
				if funcMap, ok := funcObj.(map[string]interface{}); ok {
					if name, ok := funcMap["name"].(string); ok {
						model.ToolConfig.FunctionCallingConfig.Mode = genai.FunctionCallingAny
						model.ToolConfig.FunctionCallingConfig.AllowedFunctionNames = []string{name}
					}
				}
			}
		}
	}

	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
	}
}

func (g *GeminiAdapter) GenerateEmbedding(
	ctx context.Context,
	messages []*genai.Content,
) (*openai.EmbeddingResponse, error) {
	// Add 'models/' prefix if not already present
	modelName := g.model
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}
	model := g.client.EmbeddingModel(modelName)

	batchEmbeddings := model.NewBatch()
	for _, message := range messages {
		batchEmbeddings = batchEmbeddings.AddContent(message.Parts...)
	}

	genaiResp, err := model.BatchEmbedContents(ctx, batchEmbeddings)
	if err != nil {
		return nil, errors.Wrap(err, "genai generate embeddings error")
	}

	openaiResp := openai.EmbeddingResponse{
		Object: "list",
		Data:   make([]openai.Embedding, 0, len(genaiResp.Embeddings)),
		Model:  openai.EmbeddingModel(GetMappedModel(g.model)),
	}

	for i, genaiEmbedding := range genaiResp.Embeddings {
		embedding := openai.Embedding{
			Object:    "embedding",
			Embedding: genaiEmbedding.Values,
			Index:     i,
		}
		openaiResp.Data = append(openaiResp.Data, embedding)
	}

	return &openaiResp, nil
}
