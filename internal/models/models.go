package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AnthropicRequest struct {
	Model             string             `json:"model"`
	MaxTokensToSample int                `json:"max_tokens"`
	Messages          []AnthropicMessage `json:"messages"`
	Stream            bool               `json:"stream,omitempty"`
	Temperature       *float64           `json:"temperature,omitempty"`
	TopP              *float64           `json:"top_p,omitempty"`
	TopK              *int               `json:"top_k,omitempty"`
	StopSequences     []string           `json:"stop_sequences,omitempty"`
	PresencePenalty   *float64           `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float64           `json:"frequency_penalty,omitempty"`
}

type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type AnthropicResponse struct {
	Id           string             `json:"id"`
	Type         string             `json:"type"`
	Model        string             `json:"model"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	StopReason   string             `json:"stop_reason,omitempty"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage     `json:"usage"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type OpenAIRequest struct {
	Model            string         `json:"model"`
	MaxTokens        int            `json:"max_tokens"`
	Messages         []OpenAIMessage `json:"messages"`
	Stream           bool           `json:"stream,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	N                *int           `json:"n,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Id      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []OpenAIChoice   `json:"choices"`
	Usage   OpenAIUsage      `json:"usage"`
}

type OpenAIChoice struct {
	Index        int             `json:"index"`
	Message      OpenAIMessage   `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIStreamingChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Id    string `json:"id"`
	Model string `json:"model"`
}

type AnthropicStreamingChunk struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Delta   struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// Database models for logging
type RequestLog struct {
	gorm.Model
	RequestID       string    `gorm:"index"`
	Timestamp       time.Time // Start time
	EndTime         time.Time // End time
	ClientIP        string
	RequestHeaders  string // JSON string of headers
	RequestBody     string // JSON string of request body
	RequestType     string // "anthropic" or "openai"
	ModelName       string // Renamed from Model to avoid conflict with gorm.Model
	IsStreaming     bool
	ProcessingTime  int64 // in milliseconds
	ResponseStatus  int
	ResponseHeaders string // JSON string of headers
	ResponseBody    string // JSON string of response body
	AdditionalParams string // JSON string of additional parameters like temperature, topK, etc.
	Usage           string // JSON string of usage information (optional)
	Cost            float64 // Cost of the request in USD
}

// UsageData represents the parsed usage information
type UsageData struct {
	// OpenAI usage fields
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`

	// Anthropic usage fields
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

// Model pricing information
type ModelPrice struct {
	gorm.Model
	ModelName   string  `gorm:"index;unique"`
	InputPrice  float64 // Price per input token in USD
	OutputPrice float64 // Price per output token in USD
}

// ParsedRequestLog extends RequestLog with parsed usage data
type ParsedRequestLog struct {
	RequestLog
	ParsedUsage *UsageData
}
