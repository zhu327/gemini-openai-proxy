package adapter

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
)

type ChatCompletionMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (m *ChatCompletionMessage) StringContent() string {
	return strings.Trim(string(m.Content), "\"")
}

func (m *ChatCompletionMessage) MultiContent() (parts []openai.ChatMessagePart, err error) {
	err = json.Unmarshal(m.Content, &parts)
	if err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	return
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
