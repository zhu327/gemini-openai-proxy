package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/iterator"

	"github.com/zhu327/gemini-openai-proxy/pkg/util"
)

const (
	Gemini1Pro       = "gemini-1.0-pro-latest"
	Gemini1Dot5Pro   = "gemini-1.5-pro-latest"
	Gemini1ProVision = "gemini-1.0-pro-vision-latest"
	Gemini1Ultra     = "gemini-1.0-ultra-latest"

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
	model := g.client.GenerativeModel(g.model)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatHistory(cs, messages)

	genaiResp, err := cs.SendMessage(ctx, messages[len(messages)-1].Parts...)
	if err != nil {
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
	model := g.client.GenerativeModel(g.model)
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

	for {
		genaiResp, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Printf("genai get stream message error %v\n", err)
			apiErr := openai.APIError{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			}

			resp, _ := json.Marshal(apiErr)
			dataChan <- string(resp)
			break
		}

		openaiResp := genaiResponseToStreamCompletionResponse(model, genaiResp, respID, created)
		resp, _ := json.Marshal(openaiResp)
		dataChan <- string(resp)

		if len(openaiResp.Choices) > 0 && openaiResp.Choices[0].FinishReason != nil {
			break
		}
	}
}

func genaiResponseToStreamCompletionResponse(
	model string,
	genaiResp *genai.GenerateContentResponse,
	respID string,
	created int64,
) *CompletionResponse {
	resp := CompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", respID),
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: make([]CompletionChoice, 0, len(genaiResp.Candidates)),
	}

	for i, candidate := range genaiResp.Candidates {
		var content string
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			if s, ok := candidate.Content.Parts[0].(genai.Text); ok {
				content = string(s)
			}
		}

		choice := CompletionChoice{
			Index: i,
		}
		choice.Delta.Content = content

		if candidate.FinishReason > genai.FinishReasonStop {
			log.Printf("genai message finish reason %s\n", candidate.FinishReason.String())
			openaiFinishReason := string(convertFinishReason(candidate.FinishReason))
			choice.FinishReason = &openaiFinishReason
		}

		resp.Choices = append(resp.Choices, choice)
	}
	return &resp
}

func genaiResponseToOpenaiResponse(
	model string, genaiResp *genai.GenerateContentResponse,
) openai.ChatCompletionResponse {
	resp := openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", util.GetUUID()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: make([]openai.ChatCompletionChoice, 0, len(genaiResp.Candidates)),
	}

	for i, candidate := range genaiResp.Candidates {
		var content string
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			if s, ok := candidate.Content.Parts[0].(genai.Text); ok {
				content = string(s)
			}
		}

		choice := openai.ChatCompletionChoice{
			Index: i,
			Message: openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: content,
			},
			FinishReason: convertFinishReason(candidate.FinishReason),
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
