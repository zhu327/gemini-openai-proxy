package adapter

import (
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
)

// convertOpenAIToolsToGenAI converts OpenAI tools to Gemini tools
func convertOpenAIToolsToGenAI(tools []openai.Tool) []*genai.Tool {
	var result []*genai.Tool

	for _, tool := range tools {
		if tool.Type != openai.ToolTypeFunction {
			continue // Only support function tools for now
		}

		// Convert parameters to Gemini schema
		paramsMap, ok := tool.Function.Parameters.(map[string]interface{})
		if !ok {
			// If it's not already a map, try to convert it using json
			paramsBytes, err := json.Marshal(tool.Function.Parameters)
			if err != nil {
				continue // Skip this tool if we can't convert parameters
			}

			var convertedParams map[string]interface{}
			if err := json.Unmarshal(paramsBytes, &convertedParams); err != nil {
				continue // Skip this tool if we can't convert parameters
			}
			paramsMap = convertedParams
		}

		schema := convertJSONSchemaToGenAISchema(paramsMap)

		item := &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  schema,
			}},
		}
		result = append(result, item)
	}

	return result
}

// convertJSONSchemaToGenAISchema converts a JSON schema to Gemini schema
func convertJSONSchemaToGenAISchema(params map[string]interface{}) *genai.Schema {
	genaiSchema := &genai.Schema{
		Type: genai.TypeObject,
	}

	// Extract required fields
	if required, ok := params["required"].([]interface{}); ok {
		genaiSchema.Required = make([]string, 0, len(required))
		for _, r := range required {
			if s, ok := r.(string); ok {
				genaiSchema.Required = append(genaiSchema.Required, s)
			}
		}
	}

	// Extract properties
	if properties, ok := params["properties"].(map[string]interface{}); ok {
		genaiSchema.Properties = make(map[string]*genai.Schema, len(properties))
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				genaiSchema.Properties[name] = convertPropertyToGenAISchema(propMap)
			}
		}
	}

	return genaiSchema
}

// convertPropertyToGenAISchema converts a property to Gemini schema
func convertPropertyToGenAISchema(prop map[string]interface{}) *genai.Schema {
	schema := &genai.Schema{}

	// Set type
	if t, ok := prop["type"].(string); ok {
		schema.Type = convertJSONTypeToGenAIType(t)
	}

	// Set description
	if desc, ok := prop["description"].(string); ok {
		schema.Description = desc
	}

	// Set enum values
	if enum, ok := prop["enum"].([]interface{}); ok {
		schema.Enum = make([]string, 0, len(enum))
		for _, e := range enum {
			switch v := e.(type) {
			case string:
				schema.Enum = append(schema.Enum, v)
			default:
				schema.Enum = append(schema.Enum, fmt.Sprintf("%v", v))
			}
		}
	}

	// Handle items for array type
	if schema.Type == genai.TypeArray {
		if items, ok := prop["items"].(map[string]interface{}); ok {
			schema.Items = convertPropertyToGenAISchema(items)
		}
	}

	// Handle properties for object type
	if schema.Type == genai.TypeObject {
		if properties, ok := prop["properties"].(map[string]interface{}); ok {
			schema.Properties = make(map[string]*genai.Schema, len(properties))
			for name, p := range properties {
				if propMap, ok := p.(map[string]interface{}); ok {
					schema.Properties[name] = convertPropertyToGenAISchema(propMap)
				}
			}
		}

		// Extract required fields for nested objects
		if required, ok := prop["required"].([]interface{}); ok {
			schema.Required = make([]string, 0, len(required))
			for _, r := range required {
				if s, ok := r.(string); ok {
					schema.Required = append(schema.Required, s)
				}
			}
		}
	}

	return schema
}

// convertJSONTypeToGenAIType converts JSON schema type to Gemini type
func convertJSONTypeToGenAIType(t string) genai.Type {
	switch t {
	case "string":
		return genai.TypeString
	case "integer":
		return genai.TypeInteger
	case "number":
		return genai.TypeNumber
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeUnspecified
	}
}
