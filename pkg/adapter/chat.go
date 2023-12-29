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
	GeminiPro = "gemini-pro"

	genaiRoleUser  = "user"
	genaiRoleModel = "model"
)

type GenaiModelAdapter interface {
	GenerateContent(ctx context.Context, req *ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	GenerateStreamContent(ctx context.Context, req *ChatCompletionRequest) <-chan string
}

type GeminiProAdapter struct {
	client *genai.Client
}

func NewGeminiProAdapter(client *genai.Client) GenaiModelAdapter {
	return &GeminiProAdapter{
		client: client,
	}
}

func (g *GeminiProAdapter) GenerateContent(
	ctx context.Context,
	req *ChatCompletionRequest,
) (*openai.ChatCompletionResponse, error) {
	model := g.client.GenerativeModel(GeminiPro)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatByOpenaiRequest(cs, req)

	prompt := genai.Text(req.Messages[len(req.Messages)-1].Content)
	genaiResp, err := cs.SendMessage(ctx, prompt)
	if err != nil {
		log.Printf("genai send message error %v\n", err)
		return nil, errors.Wrap(err, "genai send message error")
	}

	openaiResp := genaiResponseToOpenaiResponse(genaiResp)
	return &openaiResp, nil
}

func (g *GeminiProAdapter) GenerateStreamContent(
	ctx context.Context,
	req *ChatCompletionRequest,
) <-chan string {
	model := g.client.GenerativeModel(GeminiPro)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatByOpenaiRequest(cs, req)

	prompt := genai.Text(req.Messages[len(req.Messages)-1].Content)
	iter := cs.SendMessageStream(ctx, prompt)

	dataChan := make(chan string)
	go handleStreamIter(iter, dataChan)

	return dataChan
}

func handleStreamIter(iter *genai.GenerateContentResponseIterator, dataChan chan string) {
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

		openaiResp := genaiResponseToStreamCompletionResponse(genaiResp, respID, created)
		resp, _ := json.Marshal(openaiResp)
		dataChan <- string(resp)

		if len(openaiResp.Choices) > 0 && openaiResp.Choices[0].FinishReason != nil {
			break
		}
	}
}

func genaiResponseToStreamCompletionResponse(
	genaiResp *genai.GenerateContentResponse,
	respID string,
	created int64,
) *CompletionResponse {
	resp := CompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", respID),
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   GeminiPro,
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

			var openaiFinishReason string = string(openai.FinishReasonStop)
			if candidate.FinishReason == genai.FinishReasonMaxTokens {
				openaiFinishReason = string(openai.FinishReasonLength)
			}
			choice.FinishReason = &openaiFinishReason
		}

		resp.Choices = append(resp.Choices, choice)
	}
	return &resp
}

func genaiResponseToOpenaiResponse(
	genaiResp *genai.GenerateContentResponse,
) openai.ChatCompletionResponse {
	resp := openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", util.GetUUID()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   GeminiPro,
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
			FinishReason: openai.FinishReasonStop,
		}
		resp.Choices = append(resp.Choices, choice)
	}
	return resp
}

func setGenaiChatByOpenaiRequest(cs *genai.ChatSession, req *ChatCompletionRequest) {
	cs.History = make([]*genai.Content, 0, len(req.Messages))
	if len(req.Messages) > 1 {
		for _, message := range req.Messages[:len(req.Messages)-1] {
			switch message.Role {
			case openai.ChatMessageRoleSystem:
				cs.History = append(cs.History, []*genai.Content{
					{
						Parts: []genai.Part{
							genai.Text(message.Content),
						},
						Role: genaiRoleUser,
					},
					{
						Parts: []genai.Part{
							genai.Text("ok."),
						},
						Role: genaiRoleModel,
					},
				}...)
			case openai.ChatMessageRoleAssistant:
				cs.History = append(cs.History, &genai.Content{
					Parts: []genai.Part{
						genai.Text(message.Content),
					},
					Role: genaiRoleModel,
				})
			case openai.ChatMessageRoleUser:
				cs.History = append(cs.History, &genai.Content{
					Parts: []genai.Part{
						genai.Text(message.Content),
					},
					Role: genaiRoleUser,
				})
			}
		}
	}

	if len(cs.History) != 0 && cs.History[len(cs.History)-1].Role != genaiRoleModel {
		cs.History = append(cs.History, &genai.Content{
			Parts: []genai.Part{
				genai.Text("ok."),
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
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockOnlyHigh,
		},
	}
}
