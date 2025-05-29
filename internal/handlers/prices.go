package handlers

import (
	"html/template"
	"net/http"

	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/models"
)

// HandlePricesPage renders a page with model pricing information
func HandlePricesPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all models from the database
	var modelPrices []models.ModelPrice
	result := db.DB.Find(&modelPrices)
	if result.Error != nil {
		http.Error(w, "Error fetching models: "+result.Error.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare data for template
	data := struct {
		Models     []models.ModelPrice
		TotalCount int
	}{
		Models:     modelPrices,
		TotalCount: len(modelPrices),
	}

	// Load HTML template from file
	tmplFile := "templates/prices.html"

	// Parse the template
	t, err := template.New("prices").ParseFiles(tmplFile)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template
	w.Header().Set("Content-Type", "text/html")
	if err := t.ExecuteTemplate(w, "prices.html", data); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}