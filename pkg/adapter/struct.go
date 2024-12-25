package adapter

import (
	"encoding/json"

	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
)

type ChatCompletionMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ChatCompletionRequest represents a request structure for chat completion API.
type ChatCompletionRequest struct {
	Model       string                  `json:"model" binding:"required"`
	Messages    []ChatCompletionMessage `json:"messages" binding:"required,min=1"`
	MaxTokens   int32                   `json:"max_tokens" binding:"omitempty"`
	Temperature float32                 `json:"temperature" binding:"omitempty"`
	TopP        float32                 `json:"top_p" binding:"omitempty"`
	N           int32                   `json:"n" binding:"omitempty"`
	Stream      bool                    `json:"stream" binding:"omitempty"`
	Stop        []string                `json:"stop,omitempty"`
}

func (req *ChatCompletionRequest) ToGenaiMessages() ([]*genai.Content, error) {
	if req.Model == TextEmbedding004 || req.Model == string(openai.AdaEmbeddingV2) {
		return nil, errors.New("Chat Completion is not supported for embedding model")
	}

	return req.toVisionGenaiContent()
}

func (req *ChatCompletionRequest) toVisionGenaiContent() ([]*genai.Content, error) {
	content := make([]*genai.Content, 0, len(req.Messages))
	for _, message := range req.Messages {
		var parts []openai.ChatMessagePart

		// Attempt to unmarshal into a slice of parts
		if err := json.Unmarshal(message.Content, &parts); err != nil {
			// If it fails, try unmarshalling into a single string
			var singleString string
			if err := json.Unmarshal(message.Content, &singleString); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal message content")
			}
			// Convert single string to a part
			parts = []openai.ChatMessagePart{
				{Type: openai.ChatMessagePartTypeText, Text: singleString},
			}
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

		switch message.Role {
		case openai.ChatMessageRoleSystem:
			content = append(content, []*genai.Content{
				{
					Parts: prompt,
					Role:  genaiRoleUser,
				},
				{
					Parts: []genai.Part{
						genai.Text(" "),
					},
					Role: genaiRoleModel,
				},
			}...)
		case openai.ChatMessageRoleAssistant:
			content = append(content, &genai.Content{
				Parts: prompt,
				Role:  genaiRoleModel,
			})
		case openai.ChatMessageRoleUser:
			content = append(content, &genai.Content{
				Parts: prompt,
				Role:  genaiRoleUser,
			})
		}
	}
	return content, nil
}

type CompletionChoice struct {
	Index int `json:"index"`
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
}

type StringArray []string

// UnmarshalJSON implements the json.Unmarshaler interface for StringArray.
func (s *StringArray) UnmarshalJSON(data []byte) error {
	// Check if the data is a JSON array
	if data[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*s = arr
		return nil
	}

	// Check if the data is a JSON string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = StringArray{str} // Wrap the string in a slice
	return nil
}

// EmbeddingRequest represents a request structure for embeddings API.
type EmbeddingRequest struct {
	Model    string      `json:"model" binding:"required"`
	Messages StringArray `json:"input" binding:"required,min=1"`
}

func (req *EmbeddingRequest) ToGenaiMessages() ([]*genai.Content, error) {
	if req.Model != TextEmbedding004 && req.Model != string(openai.AdaEmbeddingV2) {
		return nil, errors.New("Embedding is not supported for chat model " + req.Model)
	}

	content := make([]*genai.Content, 0, len(req.Messages))
	for _, message := range req.Messages {
		embedString := []genai.Part{
			genai.Text(message),
		}
		content = append(content, &genai.Content{
			Parts: embedString,
		})
	}

	return content, nil
}
