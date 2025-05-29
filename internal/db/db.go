package db

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vitali/ai-gateway/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// InitDB initializes the database connection and creates tables
func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

 // Auto migrate the schema
	err = db.AutoMigrate(&models.RequestLog{}, &models.ModelPrice{})
	if err != nil {
		return nil, err
	}

	log.Printf("Database initialized at %s", dbPath)
	return db, nil
}

// IsModelPriceTableEmpty checks if the ModelPrice table is empty
func IsModelPriceTableEmpty() (bool, error) {
	var count int64
	result := DB.Model(&models.ModelPrice{}).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count == 0, nil
}

// FetchAndStoreModelPricing fetches model pricing from the /v1/models endpoint and stores it in the database
func FetchAndStoreModelPricing(targetURL string) error {
	// Construct the URL for the models endpoint
	url := fmt.Sprintf("%s/models", strings.TrimSuffix(targetURL, "/"))

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching models: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error fetching models: status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the response
	var modelsResponse struct {
		Data []struct {
			ID          string  `json:"id"`
			InputPrice  float64 `json:"input_price"`
			OutputPrice float64 `json:"output_price"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &modelsResponse); err != nil {
		return fmt.Errorf("error parsing models response: %v", err)
	}

	// Store the model pricing in the database
	for _, model := range modelsResponse.Data {
		modelPrice := models.ModelPrice{
			ModelName:   model.ID,
			InputPrice:  model.InputPrice,
			OutputPrice: model.OutputPrice,
		}

		// Use Upsert to handle existing models
		result := DB.Where("model_name = ?", model.ID).FirstOrCreate(&modelPrice)
		if result.Error != nil {
			log.Printf("Error storing model pricing for %s: %v", model.ID, result.Error)
			continue
		}

		// If the model exists but prices have changed, update them
		if result.RowsAffected == 0 {
			DB.Model(&modelPrice).Updates(models.ModelPrice{
				InputPrice:  model.InputPrice,
				OutputPrice: model.OutputPrice,
			})
		}

		log.Printf("Stored pricing for model %s: input=%f, output=%f", model.ID, model.InputPrice, model.OutputPrice)
	}

	return nil
}

// GetModelPricing gets the pricing information for a specific model
func GetModelPricing(modelName string) (*models.ModelPrice, error) {
	var modelPrice models.ModelPrice
	result := DB.Where("model_name = ?", modelName).First(&modelPrice)
	if result.Error != nil {
		return nil, result.Error
	}
	return &modelPrice, nil
}

// CalculateCost calculates the cost of a request based on token usage and model pricing
func CalculateCost(modelName string, usageJSON string) (float64, error) {
	if usageJSON == "" {
		return 0, nil
	}

	// Parse the usage data
	var usageData models.UsageData
	if err := json.Unmarshal([]byte(usageJSON), &usageData); err != nil {
		return 0, fmt.Errorf("error parsing usage data: %v", err)
	}

	// Get the model pricing
	modelPrice, err := GetModelPricing(modelName)
	if err != nil {
		return 0, fmt.Errorf("error getting model pricing: %v", err)
	}

	// Calculate the cost based on the provider
	var inputTokens, outputTokens int

	// Check if it's OpenAI usage data
	if usageData.PromptTokens > 0 || usageData.CompletionTokens > 0 {
		inputTokens = usageData.PromptTokens
		outputTokens = usageData.CompletionTokens
	} else {
		// Assume it's Anthropic usage data
		inputTokens = usageData.InputTokens
		outputTokens = usageData.OutputTokens
	}

	// Calculate the cost
	cost := float64(inputTokens) * modelPrice.InputPrice + float64(outputTokens) * modelPrice.OutputPrice

	return cost, nil
}

// GenerateRandomID generates a random ID for request logging
func GenerateRandomID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Printf("Error generating random ID: %v", err)
		return "fallback-id-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(bytes)
}

// LogRequest creates a new request log entry
func LogRequest(r *http.Request, requestType string, requestBody string, model string, isStreaming bool, additionalParams string) (*models.RequestLog, error) {
	// Convert headers to JSON
	headerMap := make(map[string][]string)
	for k, v := range r.Header {
		headerMap[k] = v
	}
	headerJSON, err := json.Marshal(headerMap)
	if err != nil {
		return nil, err
	}

	requestLog := &models.RequestLog{
		RequestID:        GenerateRandomID(),
		Timestamp:        time.Now(),
		ClientIP:         r.RemoteAddr,
		RequestHeaders:   string(headerJSON),
		RequestBody:      requestBody,
		RequestType:      requestType,
		ModelName:        model,
		IsStreaming:      isStreaming,
		AdditionalParams: additionalParams,
	}

	result := DB.Create(requestLog)
	if result.Error != nil {
		return nil, result.Error
	}

	return requestLog, nil
}

// UpdateResponseLog updates the response information in the request log
func UpdateResponseLog(requestLog *models.RequestLog, status int, responseHeaders http.Header, responseBody string, processingTime int64, usage ...string) error {
	// Convert headers to JSON
	headerMap := make(map[string][]string)
	for k, v := range responseHeaders {
		headerMap[k] = v
	}
	headerJSON, err := json.Marshal(headerMap)
	if err != nil {
		return err
	}

	requestLog.ResponseStatus = status
	requestLog.ResponseHeaders = string(headerJSON)
	requestLog.ResponseBody = responseBody
	requestLog.ProcessingTime = processingTime
	requestLog.EndTime = time.Now()

	// Set usage if provided
	if len(usage) > 0 && usage[0] != "" {
		requestLog.Usage = usage[0]

		// Calculate cost for requests with usage data (both streaming and non-streaming)
		cost, err := CalculateCost(requestLog.ModelName, usage[0])
		if err != nil {
			log.Printf("Error calculating cost: %v", err)
		} else {
			requestLog.Cost = cost
			log.Printf("Calculated cost for request %s: $%.6f", requestLog.RequestID, cost)
		}
	}

	result := DB.Save(requestLog)
	return result.Error
}
