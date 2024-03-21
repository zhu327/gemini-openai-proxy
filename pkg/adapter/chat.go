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
	GeminiPro       = "gemini-pro"
	GeminiProVision = "gemini-pro-vision"

	genaiRoleUser  = "user"
	genaiRoleModel = "model"
)

type GenaiModelAdapter interface {
	GenerateContent(ctx context.Context, req *ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	GenerateStreamContent(ctx context.Context, req *ChatCompletionRequest) (<-chan string, error)
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

	prompt := genai.Text(req.Messages[len(req.Messages)-1].StringContent())
	genaiResp, err := cs.SendMessage(ctx, prompt)
	if err != nil {
		return nil, errors.Wrap(err, "genai send message error")
	}

	openaiResp := genaiResponseToOpenaiResponse(genaiResp)
	return &openaiResp, nil
}

func (g *GeminiProAdapter) GenerateStreamContent(
	ctx context.Context,
	req *ChatCompletionRequest,
) (<-chan string, error) {
	model := g.client.GenerativeModel(GeminiPro)
	setGenaiModelByOpenaiRequest(model, req)

	cs := model.StartChat()
	setGenaiChatByOpenaiRequest(cs, req)

	prompt := genai.Text(req.Messages[len(req.Messages)-1].StringContent())
	iter := cs.SendMessageStream(ctx, prompt)

	dataChan := make(chan string)
	go handleStreamIter(iter, dataChan)

	return dataChan, nil
}

type GeminiProVisionAdapter struct {
	client *genai.Client
}

func NewGeminiProVisionAdapter(client *genai.Client) GenaiModelAdapter {
	return &GeminiProVisionAdapter{
		client: client,
	}
}

func (g *GeminiProVisionAdapter) GenerateContent(
	ctx context.Context,
	req *ChatCompletionRequest,
) (*openai.ChatCompletionResponse, error) {
	model := g.client.GenerativeModel(GeminiProVision)
	setGenaiModelByOpenaiRequest(model, req)

	// NOTE: use last message as prompt, gemini pro vision does not support context
	// https://ai.google.dev/tutorials/go_quickstart#multi-turn-conversations-chat
	prompt, err := g.openaiMessageToGenaiPrompt(req.Messages[len(req.Messages)-1])
	if err != nil {
		return nil, errors.Wrap(err, "genai generate prompt error")
	}

	genaiResp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, errors.Wrap(err, "genai send message error")
	}

	openaiResp := genaiResponseToOpenaiResponse(genaiResp)
	return &openaiResp, nil
}

func (*GeminiProVisionAdapter) openaiMessageToGenaiPrompt(msg ChatCompletionMessage) ([]genai.Part, error) {
	parts, err := msg.MultiContent()
	if err != nil {
		return nil, err
	}

	prompt := make([]genai.Part, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case openai.ChatMessagePartTypeText:
			prompt = append(prompt, genai.Text(part.Text))
		case openai.ChatMessagePartTypeImageURL:
			data, format, err := parseImageURL(part.ImageURL.URL)
			if err != nil {
				return nil, errors.Wrap(err, "parse image url error")
			}

			prompt = append(prompt, genai.ImageData(format, data))
		}
	}
	return prompt, nil
}

func (g *GeminiProVisionAdapter) GenerateStreamContent(
	ctx context.Context,
	req *ChatCompletionRequest,
) (<-chan string, error) {
	model := g.client.GenerativeModel(GeminiProVision)
	setGenaiModelByOpenaiRequest(model, req)

	// NOTE: use last message as prompt, gemini pro vision does not support context
	// https://ai.google.dev/tutorials/go_quickstart#multi-turn-conversations-chat
	prompt, err := g.openaiMessageToGenaiPrompt(req.Messages[len(req.Messages)-1])
	if err != nil {
		return nil, errors.Wrap(err, "genai generate prompt error")
	}

	iter := model.GenerateContentStream(ctx, prompt...)

	dataChan := make(chan string)
	go handleStreamIter(iter, dataChan)

	return dataChan, nil
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
			openaiFinishReason := string(convertFinishReason(candidate.FinishReason))
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

func setGenaiChatByOpenaiRequest(cs *genai.ChatSession, req *ChatCompletionRequest) {
	cs.History = make([]*genai.Content, 0, len(req.Messages))
	if len(req.Messages) > 1 {
		for _, message := range req.Messages[:len(req.Messages)-1] {
			switch message.Role {
			case openai.ChatMessageRoleSystem:
				cs.History = append(cs.History, []*genai.Content{
					{
						Parts: []genai.Part{
							genai.Text(message.StringContent()),
						},
						Role: genaiRoleUser,
					},
					{
						Parts: []genai.Part{
							genai.Text(""),
						},
						Role: genaiRoleModel,
					},
				}...)
			case openai.ChatMessageRoleAssistant:
				cs.History = append(cs.History, &genai.Content{
					Parts: []genai.Part{
						genai.Text(message.StringContent()),
					},
					Role: genaiRoleModel,
				})
			case openai.ChatMessageRoleUser:
				cs.History = append(cs.History, &genai.Content{
					Parts: []genai.Part{
						genai.Text(message.StringContent()),
					},
					Role: genaiRoleUser,
				})
			}
		}
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
