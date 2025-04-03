package adapter

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	Gemini1Dot5Pro   = "gemini-1.5-pro-latest"
	Gemini1Dot5Flash = "gemini-1.5-flash-002"
	Gemini1Dot5ProV  = "gemini-1.0-pro-vision-latest" // Converted to one of the above models in struct::ToGenaiModel
	Gemini2FlashExp  = "gemini-2.0-flash-exp"
	TextEmbedding004 = "text-embedding-004"
)

// GeminiModels stores the available models from Gemini API
var (
	GeminiModels     []string
	geminiModelsOnce sync.Once
	geminiModelsLock sync.RWMutex
)

var USE_MODEL_MAPPING bool = os.Getenv("DISABLE_MODEL_MAPPING") != "1"

// FetchGeminiModels fetches available models from Gemini API
func FetchGeminiModels(ctx context.Context, apiKey string) ([]string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	models := []string{}
	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Strip the 'models/' prefix from model names
		modelName := m.Name
		modelName = strings.TrimPrefix(modelName, "models/")
		models = append(models, modelName)
	}

	return models, nil
}

// InitGeminiModels initializes the GeminiModels slice with available models
func InitGeminiModels(apiKey string) error {
	var initErr error
	geminiModelsOnce.Do(func() {
		ctx := context.Background()
		models, err := FetchGeminiModels(ctx, apiKey)
		if err != nil {
			log.Printf("Failed to fetch Gemini models: %v\n", err)
			// Fallback to default models
			geminiModelsLock.Lock()
			GeminiModels = []string{
				Gemini1Dot5Pro,
				Gemini1Dot5Flash,
				Gemini1Dot5ProV,
				Gemini2FlashExp,
				TextEmbedding004,
			}
			geminiModelsLock.Unlock()
			initErr = err
			return
		}
		geminiModelsLock.Lock()
		GeminiModels = models
		geminiModelsLock.Unlock()
		log.Printf("Initialized Gemini models: %v\n", GeminiModels)
	})
	return initErr
}

// GetAvailableGeminiModels returns the available Gemini models
func GetAvailableGeminiModels() []string {
	geminiModelsLock.RLock()
	defer geminiModelsLock.RUnlock()

	if len(GeminiModels) == 0 {
		return []string{Gemini1Dot5Pro, Gemini1Dot5Flash, Gemini1Dot5ProV, Gemini2FlashExp, TextEmbedding004}
	}

	return GeminiModels
}

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

// IsValidGeminiModel checks if the model is a valid Gemini model
func IsValidGeminiModel(modelName string) bool {
	if len(GeminiModels) == 0 {
		// If models haven't been fetched yet, use the default list
		return modelName == Gemini1Dot5Pro ||
			modelName == Gemini1Dot5Flash ||
			modelName == Gemini1Dot5ProV ||
			modelName == Gemini2FlashExp ||
			modelName == TextEmbedding004
	}

	geminiModelsLock.RLock()
	defer geminiModelsLock.RUnlock()

	for _, model := range GeminiModels {
		if model == modelName {
			return true
		}
	}

	return false
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
		// Check if the model is valid
		if IsValidGeminiModel(req.Model) {
			return req.Model
		}

		// Fallback to default model if not valid
		log.Printf("Invalid model: %s, falling back to %s\n", req.Model, Gemini1Dot5Flash)
		return Gemini1Dot5Flash
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
		// Check if the model is valid
		if IsValidGeminiModel(req.Model) {
			return req.Model
		}

		// Fallback to default embedding model if not valid
		log.Printf("Invalid embedding model: %s, falling back to %s\n", req.Model, TextEmbedding004)
		return TextEmbedding004
	}
}
