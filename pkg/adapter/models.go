package adapter

import (
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	Gemini1Dot5Pro   = "gemini-1.5-pro-latest"
	Gemini1Dot5Flash = "gemini-1.5-flash-002"
	Gemini1Dot5ProV  = "gemini-1.0-pro-vision-latest" // Converted to one of the above models in struct::ToGenaiModel
	Gemini2FlashExp  = "gemini-2.0-flash-exp"
	TextEmbedding004 = "text-embedding-004"
)

var USE_MODEL_MAPPING bool = os.Getenv("DISABLE_MODEL_MAPPING") != "1"

func GetOwner() string {
	if USE_MODEL_MAPPING {
		return "openai"
	} else {
		return "google"
	}
}

func GetModel(openAiModelName string) string {
	if USE_MODEL_MAPPING {
		return openAiModelName
	} else {
		return ConvertModel(openAiModelName)
	}
}

func GetMappedModel(geminiModelName string) string {
	if !USE_MODEL_MAPPING {
		return geminiModelName
	}
	switch {
	case geminiModelName == Gemini1Dot5Pro:
		return openai.GPT4TurboPreview
	case geminiModelName == Gemini1Dot5Flash:
		return openai.GPT4
	case geminiModelName == Gemini2FlashExp:
		return openai.GPT4o
	case geminiModelName == TextEmbedding004:
		return string(openai.AdaEmbeddingV2)
	default:
		return openai.GPT3Dot5Turbo
	}
}

func ConvertModel(openAiModelName string) string {
	switch {
	case openAiModelName == openai.GPT4VisionPreview:
		return Gemini1Dot5ProV
	case openAiModelName == openai.GPT4TurboPreview || openAiModelName == openai.GPT4Turbo1106 || openAiModelName == openai.GPT4Turbo0125:
		return Gemini1Dot5Pro
	case strings.HasPrefix(openAiModelName, openai.GPT4):
		return Gemini1Dot5Flash
	case openAiModelName == string(openai.AdaEmbeddingV2):
		return TextEmbedding004
	case openAiModelName == openai.GPT4o:
		return Gemini2FlashExp
	default:
		return Gemini1Dot5Flash
	}
}

func (req *ChatCompletionRequest) ToGenaiModel() string {
	if USE_MODEL_MAPPING {
		return req.ParseModelWithMapping()
	} else {
		return req.ParseModelWithoutMapping()
	}
}

func (req *ChatCompletionRequest) ParseModelWithoutMapping() string {
	switch {
	case req.Model == Gemini1Dot5ProV:
		if os.Getenv("GPT_4_VISION_PREVIEW") == Gemini1Dot5Pro {
			return Gemini1Dot5Pro
		}

		return Gemini1Dot5Flash
	default:
		return req.Model
	}
}

func (req *ChatCompletionRequest) ParseModelWithMapping() string {
	switch {
	case req.Model == openai.GPT4VisionPreview:
		if os.Getenv("GPT_4_VISION_PREVIEW") == Gemini1Dot5Pro {
			return Gemini1Dot5Pro
		}

		return Gemini1Dot5Flash
	default:
		return ConvertModel(req.Model)
	}
}

func (req *EmbeddingRequest) ToGenaiModel() string {
	if USE_MODEL_MAPPING {
		return ConvertModel(req.Model)
	} else {
		return req.Model
	}
}
