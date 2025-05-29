package converter

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/vitali/ai-gateway/internal/models"
)

func ConvertToOpenAI(anthropicReq models.AnthropicRequest) models.OpenAIRequest {
	openaiMessages := make([]models.OpenAIMessage, len(anthropicReq.Messages))
	for i, msg := range anthropicReq.Messages {
		var content string

		err := json.Unmarshal(msg.Content, &content)
		if err == nil {
			openaiMessages[i] = models.OpenAIMessage{
				Role:    msg.Role,
				Content: content,
			}
		} else {
			var structuredContent []map[string]interface{}
			err = json.Unmarshal(msg.Content, &structuredContent)
			if err == nil {
				combinedContent := ""
				for _, block := range structuredContent {
					if blockType, ok := block["type"].(string); ok && blockType == "text" {
						if text, ok := block["text"].(string); ok {
							combinedContent += text
						}
					}
				}
				openaiMessages[i] = models.OpenAIMessage{
					Role:    msg.Role,
					Content: combinedContent,
				}
			} else {
				log.Printf("Error parsing message content: %v", err)
				openaiMessages[i] = models.OpenAIMessage{
					Role:    msg.Role,
					Content: "",
				}
			}
		}
	}
	model := anthropicReq.Model
	if !strings.Contains(model, "/") {
		model = "openai/" + model
	}

	openaiReq := models.OpenAIRequest{
		Model:     model,
		MaxTokens: anthropicReq.MaxTokensToSample,
		Messages:  openaiMessages,
		Stream:    anthropicReq.Stream,
	}

	if anthropicReq.Temperature != nil {
		openaiReq.Temperature = anthropicReq.Temperature
	}
	if anthropicReq.TopP != nil {
		openaiReq.TopP = anthropicReq.TopP
	}
	if anthropicReq.StopSequences != nil && len(anthropicReq.StopSequences) > 0 {
		openaiReq.Stop = anthropicReq.StopSequences
	}
	if anthropicReq.PresencePenalty != nil {
		openaiReq.PresencePenalty = anthropicReq.PresencePenalty
	}
	if anthropicReq.FrequencyPenalty != nil {
		openaiReq.FrequencyPenalty = anthropicReq.FrequencyPenalty
	}

	// Map TopK to N if present (approximate mapping)
	if anthropicReq.TopK != nil {
		n := *anthropicReq.TopK
		if n > 0 {
			openaiReq.N = &n
		}
	}

	return openaiReq
}

func ConvertToAnthropic(openaiResp models.OpenAIResponse) models.AnthropicResponse {
	anthropicResp := models.AnthropicResponse{
		Id:    openaiResp.Id,
		Type:  "message",
		Model: openaiResp.Model,
		Role:  "assistant",
		Usage: models.AnthropicUsage{
			InputTokens:  openaiResp.Usage.PromptTokens,
			OutputTokens: openaiResp.Usage.CompletionTokens,
		},
	}

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]
		anthropicResp.Content = []models.AnthropicContent{
			{
				Type: "text",
				Text: choice.Message.Content,
			},
		}

		if choice.FinishReason != "" {
			anthropicResp.StopReason = choice.FinishReason
		}
	}

	return anthropicResp
}
