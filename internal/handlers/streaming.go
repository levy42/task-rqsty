package handlers

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/models"
	"github.com/vitali/ai-gateway/internal/token_counter"
)

// HandleStreamingResponse handles streaming responses from the API
func HandleStreamingResponse(w http.ResponseWriter, req *http.Request, client *http.Client, requestLog *models.RequestLog, model string, provider string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Printf("Starting streaming request to API")
	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		if requestLog != nil {
			processingTime := time.Since(startTime).Milliseconds()
			db.UpdateResponseLog(requestLog, http.StatusInternalServerError, nil, err.Error(), processingTime, "")
		}
		return
	}
	defer resp.Body.Close()

	log.Printf("Streaming response from API: status=%d, headers=%v", resp.StatusCode, resp.Header)

	if requestLog != nil {
		// We'll update the processing time at the end of streaming
		db.UpdateResponseLog(requestLog, resp.StatusCode, resp.Header, "Streaming response", 0, "")
	}

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageID := db.GenerateRandomID()

	initialEvent := struct {
		Type  string `json:"type"`
		Message struct {
			Content []string `json:"content"`
		} `json:"message"`
		Id    string `json:"id"`
		Model string `json:"model"`
		Role  string `json:"role"`
	}{
		Type:    "message_start",
		Id:      messageID,
		Model:   model,
		Role:    "assistant",
		Message: struct {
			Content []string `json:"content"`
		    }{
		    Content: []string{},
		    },
	}

	initialJSON, err := json.Marshal(initialEvent)
	if err != nil {
		log.Printf("Error marshaling initial event: %v", err)
		http.Error(w, "Error creating streaming response", http.StatusInternalServerError)
		return
	}

	_, writeErr := w.Write([]byte("event: message_start\ndata: " + string(initialJSON) + "\n\n"))
	if writeErr != nil {
		log.Printf("Error writing initial event: %v", writeErr)
		return
	}
	flusher.Flush()

	contentStartEvent := struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		ContentBlock struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content_block"`
	}{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			Type: "text",
			Text: "",
		},
	}

	contentStartJSON, err := json.Marshal(contentStartEvent)
	if err != nil {
		log.Printf("Error marshaling content start event: %v", err)
		http.Error(w, "Error creating streaming response", http.StatusInternalServerError)
		return
	}

	_, writeErr = w.Write([]byte("event: content_block_start\ndata: " + string(contentStartJSON) + "\n\n"))
	if writeErr != nil {
		log.Printf("Error writing content start event: %v", writeErr)
		return
	}
	flusher.Flush()

	var fullTextOutput strings.Builder
	var usageJSON string

	var inputTokens int
	if requestLog != nil {
		var err error
		inputTokens, err = token_counter.CountTokensInRequest(requestLog.RequestBody, provider)
		if err != nil {
			log.Printf("Error counting tokens in request: %v", err)
		} else {
			log.Printf("Counted %d tokens in request", inputTokens)
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			log.Printf("Received [DONE] from OpenAI")
			contentStopEvent := struct {
				Type  string `json:"type"`
				Index int    `json:"index"`
			}{
				Type:  "content_block_stop",
				Index: 0,
			}

			contentStopJSON, err := json.Marshal(contentStopEvent)
			if err != nil {
				log.Printf("Error marshaling content stop event: %v", err)
				continue
			}

			_, writeErr := w.Write([]byte("event: content_block_stop\ndata: " + string(contentStopJSON) + "\n\n"))
			if writeErr != nil {
				log.Printf("Error writing content stop event: %v", writeErr)
			}
			flusher.Flush()

			var outputTokens int
			if fullTextOutput.Len() > 0 {
				var err error
				outputTokens, err = token_counter.CountTokensInResponse(fullTextOutput.String(), model)
				if err != nil {
					log.Printf("Error counting tokens in response: %v", err)
				} else {
					log.Printf("Counted %d tokens in response at [DONE]", outputTokens)
				}
			}

			messageStopEvent := struct {
				Type  string `json:"type"`
				Id    string `json:"id"`
				Role  string `json:"role"`
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}{
				Type: "message_stop",
				Id:   messageID,
				Role: "assistant",
				Usage: struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				}{
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
				},
			}

			usageData, err := token_counter.CreateUsageJSON(inputTokens, outputTokens, provider)
			if err != nil {
				log.Printf("Error creating usage JSON: %v", err)
			} else {
				usageJSON = usageData
				log.Printf("Created usage JSON for [DONE] event: %s", usageJSON)
			}

			messageStopJSON, err := json.Marshal(messageStopEvent)
			if err != nil {
				log.Printf("Error marshaling message stop event: %v", err)
				continue
			}

			_, writeErr = w.Write([]byte("event: message_stop\ndata: " + string(messageStopJSON) + "\n\n"))
			if writeErr != nil {
				log.Printf("Error writing message stop event: %v", writeErr)
			}
			flusher.Flush()

			_, writeErr = w.Write([]byte("event: done\ndata: [DONE]\n\n"))
			if writeErr != nil {
				log.Printf("Error writing [DONE] message: %v", writeErr)
			}
			flusher.Flush()
			continue
		}

		var openaiChunk models.OpenAIStreamingChunk

		log.Printf("Received OpenAI chunk: %s", data)
		if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
			log.Printf("Error parsing OpenAI chunk: %v", err)
			continue
		}

		if len(openaiChunk.Choices) == 0 || openaiChunk.Choices[0].Delta.Content == "" {
			continue
		}

		fullTextOutput.WriteString(openaiChunk.Choices[0].Delta.Content)

		anthropicChunk := models.AnthropicStreamingChunk{
			Type:  "content_block_delta",
			Index: 0, // Keep index at 0 for all text deltas
			Delta: struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				Type: "text_delta",
				Text: openaiChunk.Choices[0].Delta.Content,
			},
		}

		anthropicJSON, err := json.Marshal(anthropicChunk)
		log.Printf("Sending chunk: %s", string(anthropicJSON))
		if err != nil {
			log.Printf("Error marshaling Anthropic chunk: %v", err)
			continue
		}

		_, writeErr := w.Write([]byte("event: content_block_delta\ndata: " + string(anthropicJSON) + "\n\n"))
		if writeErr != nil {
			log.Printf("Error writing to response: %v", writeErr)
			break
		}
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from response: %v", err)
		if requestLog != nil {
			processingTime := time.Since(startTime).Milliseconds()
			db.UpdateResponseLog(requestLog, http.StatusInternalServerError, nil, "Error reading from response: "+err.Error(), processingTime, "")
		}
	} else {
		log.Printf("Completed streaming response")
		if requestLog != nil {
			processingTime := time.Since(startTime).Milliseconds()

			outputTokens, err := token_counter.CountTokensInResponse(fullTextOutput.String(), model)
			if err != nil {
				log.Printf("Error counting tokens in response: %v", err)
			} else {
				log.Printf("Counted %d tokens in response", outputTokens)

				usageJSON, err = token_counter.CreateUsageJSON(inputTokens, outputTokens, provider)
				if err != nil {
					log.Printf("Error creating usage JSON: %v", err)
				} else {
					log.Printf("Created usage JSON: %s", usageJSON)
				}
			}

			db.UpdateResponseLog(requestLog, http.StatusOK, resp.Header, fullTextOutput.String(), processingTime, usageJSON)
		}
	}
}
