package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/models"
)

// HandleModels handles the /v1/models endpoint
func HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all models from the database
	var modelPrices []models.ModelPrice
	result := db.DB.Find(&modelPrices)
	if result.Error != nil {
		log.Printf("Error fetching models: %v", result.Error)
		http.Error(w, "Error fetching models", http.StatusInternalServerError)
		return
	}

	// Convert to the expected response format
	type ModelData struct {
		ID          string  `json:"id"`
		Object      string  `json:"object"`
		Created     int64   `json:"created"`
		InputPrice  float64 `json:"input_price"`
		OutputPrice float64 `json:"output_price"`
	}

	response := struct {
		Object string      `json:"object"`
		Data   []ModelData `json:"data"`
	}{
		Object: "list",
		Data:   make([]ModelData, 0, len(modelPrices)),
	}

	for _, mp := range modelPrices {
		response.Data = append(response.Data, ModelData{
			ID:          mp.ModelName,
			Object:      "model",
			Created:     mp.CreatedAt.Unix(),
			InputPrice:  mp.InputPrice,
			OutputPrice: mp.OutputPrice,
		})
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding models response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}