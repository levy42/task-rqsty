package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vitali/ai-gateway/internal/config"
	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/handlers"
)

func main() {
	cfg := config.ParseFlags()

	var err error
	db.DB, err = db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Printf("Database initialized successfully")

	isEmpty, err := db.IsModelPriceTableEmpty()
	if err != nil {
		log.Printf("Error checking if ModelPrice table is empty: %v", err)
	} else if isEmpty {
		log.Printf("ModelPrice table is empty, fetching model pricing...")
		if err := db.FetchAndStoreModelPricing(cfg.TargetURL); err != nil {
			log.Printf("Error fetching and storing model pricing: %v", err)
		} else {
			log.Printf("Model pricing fetched and stored successfully")
		}
	} else {
		log.Printf("ModelPrice table already has data, skipping fetch")
	}

	http.HandleFunc("/", handlers.HandleLogsPage)

	http.HandleFunc("/prices", handlers.HandlePricesPage)

	http.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleMessages(w, r, cfg)
	})

	http.HandleFunc("/v1/models", handlers.HandleModels)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
