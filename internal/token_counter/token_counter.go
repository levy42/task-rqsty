package token_counter

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/vitali/ai-gateway/internal/models"
)

// CountTokens counts the tokens in the given text using the specified model
func CountTokens(text string, model string) (int, error) {
	encoding := getEncodingForModel(model)
	if encoding == "" {
		return 0, fmt.Errorf("unsupported model: %s", model)
	}

	tkm, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("error getting encoding: %v", err)
	}

	tokens := tkm.Encode(text, nil, nil)
	return len(tokens), nil
}

// CountTokensInRequest counts the tokens in the request body
func CountTokensInRequest(requestBody string, requestType string) (int, error) {
	if requestType == "anthropic" {
		var anthropicReq models.AnthropicRequest
		if err := json.Unmarshal([]byte(requestBody), &anthropicReq); err != nil {
			return 0, fmt.Errorf("error parsing Anthropic request: %v", err)
		}

		totalTokens := 0
		for _, message := range anthropicReq.Messages {
			var contentStr string

			if err := json.Unmarshal(message.Content, &contentStr); err == nil {
				tokens, err := CountTokens(contentStr, anthropicReq.Model)
				if err != nil {
					return 0, err
				}
				totalTokens += tokens
			} else {
				var contentBlocks []models.AnthropicContent
				if err := json.Unmarshal(message.Content, &contentBlocks); err != nil {
					return 0, fmt.Errorf("error parsing message content: %v", err)
				}

				for _, block := range contentBlocks {
					if block.Type == "text" {
						tokens, err := CountTokens(block.Text, anthropicReq.Model)
						if err != nil {
							return 0, err
						}
						totalTokens += tokens
					}
				}
			}
		}

		return totalTokens, nil
	} else if requestType == "openai" {
		var openaiReq models.OpenAIRequest
		if err := json.Unmarshal([]byte(requestBody), &openaiReq); err != nil {
			return 0, fmt.Errorf("error parsing OpenAI request: %v", err)
		}

		totalTokens := 0
		for _, message := range openaiReq.Messages {
			tokens, err := CountTokens(message.Content, openaiReq.Model)
			if err != nil {
				return 0, err
			}
			totalTokens += tokens
		}

		return totalTokens, nil
	}

	return 0, fmt.Errorf("unsupported request type: %s", requestType)
}

// CountTokensInResponse counts the tokens in the response text
func CountTokensInResponse(responseText string, model string) (int, error) {
	return CountTokens(responseText, model)
}

// CreateUsageJSON creates a JSON string with the token usage information
func CreateUsageJSON(inputTokens int, outputTokens int, requestType string) (string, error) {
	var usageData models.UsageData

	if requestType == "anthropic" {
		usageData = models.UsageData{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}
	} else if requestType == "openai" {
		usageData = models.UsageData{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		}
	} else {
		return "", fmt.Errorf("unsupported request type: %s", requestType)
	}

	usageJSON, err := json.Marshal(usageData)
	if err != nil {
		return "", fmt.Errorf("error marshaling usage data: %v", err)
	}

	return string(usageJSON), nil
}

// getEncodingForModel returns the encoding name for the given model
func getEncodingForModel(model string) string {
	modelLower := strings.ToLower(model)

	if strings.Contains(modelLower, "gpt-4") {
		return "cl100k_base"
	}
	if strings.Contains(modelLower, "gpt-3.5-turbo") {
		return "cl100k_base"
	}

	if strings.Contains(modelLower, "claude") {
		return "cl100k_base"  // Using the same encoding as GPT-4 for Claude models
	}

	log.Printf("Unknown model: %s, using cl100k_base encoding", model)
	return "cl100k_base"
}
