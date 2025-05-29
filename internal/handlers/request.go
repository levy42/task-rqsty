package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vitali/ai-gateway/internal/config"
	"github.com/vitali/ai-gateway/internal/converter"
	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/models"
)

// HandleMessages handles the /v1/messages endpoint
func HandleMessages(w http.ResponseWriter, r *http.Request, config config.Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var anthropicReq models.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		http.Error(w, "Error parsing request JSON", http.StatusBadRequest)
		return
	}

	// Debug print for Anthropic request
	anthropicDebug, _ := json.MarshalIndent(anthropicReq, "", "  ")
	log.Printf("Incoming Anthropic request: %s", string(anthropicDebug))

	// Extract additional parameters for logging
	additionalParams := map[string]interface{}{
		"temperature": anthropicReq.Temperature,
		"top_p": anthropicReq.TopP,
		"top_k": anthropicReq.TopK,
		"stop_sequences": anthropicReq.StopSequences,
		"presence_penalty": anthropicReq.PresencePenalty,
		"frequency_penalty": anthropicReq.FrequencyPenalty,
	}

	additionalParamsJSON, err := json.Marshal(additionalParams)
	if err != nil {
		log.Printf("Error marshaling additional params: %v", err)
		additionalParamsJSON = []byte("{}")
	}

	// Log the request to the database
	requestLog, err := db.LogRequest(r, "anthropic", string(body), anthropicReq.Model, anthropicReq.Stream, string(additionalParamsJSON))
	if err != nil {
		log.Printf("Error logging request: %v", err)
		// Continue processing even if logging fails
	}

	openaiReq := converter.ConvertToOpenAI(anthropicReq)

	ForwardRequest(w, r, openaiReq, config, requestLog, "anthropic")
}

// ForwardRequest forwards the request to the target API
func ForwardRequest(w http.ResponseWriter, r *http.Request, openaiReq models.OpenAIRequest, config config.Config, requestLog *models.RequestLog, provider string) {
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		http.Error(w, "Error creating forwarded request", http.StatusInternalServerError)
		return
	}

	openaiDebug, _ := json.MarshalIndent(openaiReq, "", "  ")
	// TargetURL doesn't have "/" in the end as it's trimmed in config.go
	url := config.TargetURL + "/chat/completions"
	log.Printf("Forwarding OpenAI request to %s: %s", url, string(openaiDebug))

	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		http.Error(w, "Error creating forwarded request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	if xAPIKey := r.Header.Get("x-api-key"); xAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+xAPIKey)
	}

	client := &http.Client{}

	startTime := time.Now()

	if openaiReq.Stream {
		HandleStreamingResponse(w, req, client, requestLog, openaiReq.Model, provider)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// Log error response if we have a requestLog
		if requestLog != nil {
			processingTime := time.Since(startTime).Milliseconds()
			db.UpdateResponseLog(requestLog, http.StatusInternalServerError, nil, err.Error(), processingTime, "")
		}
		return
	}
	defer resp.Body.Close()

	// Debug print for response status and headers
	log.Printf("Response from API: status=%d, headers=%v", resp.StatusCode, resp.Header)

	// Calculate processing time
	processingTime := time.Since(startTime).Milliseconds()

	// If the response is not successful, just pass it through
	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Read the response body for logging
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return
		}

		// Log the error response
		if requestLog != nil {
			db.UpdateResponseLog(requestLog, resp.StatusCode, resp.Header, string(responseBody), processingTime, "")
		}

		// Write the response body to the client
		if _, err := w.Write(responseBody); err != nil {
			log.Printf("Error copying response: %v", err)
		}
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response body", http.StatusInternalServerError)
		// Log the error if we have a requestLog
		if requestLog != nil {
			db.UpdateResponseLog(requestLog, http.StatusInternalServerError, nil, "Error reading response body", processingTime, "")
		}
		return
	}

	// Parse the OpenAI response
	var openaiResp models.OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		log.Printf("Error parsing OpenAI response: %v", err)
		// If we can't parse the response, just pass it through
		w.WriteHeader(resp.StatusCode)
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		// Log the unparseable response
		if requestLog != nil {
			db.UpdateResponseLog(requestLog, resp.StatusCode, resp.Header, string(body), processingTime, "")
		}
		w.Write(body)
		return
	}

	// Convert the OpenAI response to Anthropic format
	anthropicResp := converter.ConvertToAnthropic(openaiResp)

	// Debug print for Anthropic response
	anthropicDebug, _ := json.MarshalIndent(anthropicResp, "", "  ")
	log.Printf("Converted Anthropic response: %s", string(anthropicDebug))

	// Set the content type header
	w.Header().Set("Content-Type", "application/json")

	// Write the response
	w.WriteHeader(http.StatusOK)

	// Encode the Anthropic response to JSON for both logging and response
	responseJSON, err := json.Marshal(anthropicResp)
	if err != nil {
		log.Printf("Error encoding Anthropic response: %v", err)
		return
	}

	// Extract usage information if available
	var usageJSON string
	log.Printf("%s", openaiResp.Usage)
	if openaiResp.Usage.TotalTokens > 0 {
		usageData, err := json.Marshal(openaiResp.Usage)
		if err != nil {
			log.Printf("Error encoding usage information: %v", err)
			usageJSON = ""
		} else {
			usageJSON = string(usageData)
		}
	}

	// Log the successful response
	if requestLog != nil {
		db.UpdateResponseLog(requestLog, http.StatusOK, resp.Header, string(responseJSON), processingTime, usageJSON)
	}

	// Write the response to the client
	if _, err := w.Write(responseJSON); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
