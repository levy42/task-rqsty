package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/vitali/ai-gateway/internal/db"
	"github.com/vitali/ai-gateway/internal/models"
)

// HandleLogsPage renders a page with database logs and pagination
func HandleLogsPage(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	pageSize := 10

	// Get page from query parameters
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	// Get page size from query parameters
	pageSizeParam := r.URL.Query().Get("pageSize")
	if pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query logs with pagination
	var logs []models.RequestLog
	var totalCount int64

	// Get total count
	db.DB.Model(&models.RequestLog{}).Count(&totalCount)

	// Get paginated logs
	result := db.DB.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)
	if result.Error != nil {
		http.Error(w, "Error retrieving logs: "+result.Error.Error(), http.StatusInternalServerError)
		return
	}

	// Parse usage data for each log
	var parsedLogs []models.ParsedRequestLog
	for _, log := range logs {
		parsedLog := models.ParsedRequestLog{
			RequestLog: log,
		}

		// Parse usage data if available
		if log.Usage != "" {
			var usageData models.UsageData
			if err := json.Unmarshal([]byte(log.Usage), &usageData); err == nil {
				parsedLog.ParsedUsage = &usageData
			}
		}

		parsedLogs = append(parsedLogs, parsedLog)
	}

	// Calculate total pages
	totalPages := (int(totalCount) + pageSize - 1) / pageSize

	// Prepare data for template
	data := struct {
		Logs       []models.ParsedRequestLog
		Page       int
		PageSize   int
		TotalPages int
		TotalCount int64
		NextPage   int
		PrevPage   int
	}{
		Logs:       parsedLogs,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		TotalCount: totalCount,
		NextPage:   page + 1,
		PrevPage:   page - 1,
	}

	// Load HTML template from file
	tmplFile := "templates/logs.html"

	// Create a template function map for the sequence function
	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			if end < start {
				return nil
			}
			// Limit the number of page links to avoid too many links
			if end-start > 10 {
				if page > 5 {
					start = page - 5
				}
				if end > start+10 {
					end = start + 10
				}
			}
			s := make([]int, end-start+1)
			for i := range s {
				s[i] = start + i
			}
			return s
		},
	}

	// Parse the template with the function map
	t, err := template.New("logs").Funcs(funcMap).ParseFiles(tmplFile)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template
	w.Header().Set("Content-Type", "text/html")
	if err := t.ExecuteTemplate(w, "logs.html", data); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
