package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ConversionRequest struct {
	SourceLanguage string `json:"sourceLanguage"`
	SourceCode     string `json:"sourceCode"`
}

type ConversionResponse struct {
	ID             string     `json:"id"`
	SourceLanguage string     `json:"sourceLanguage"`
	SourceCode     string     `json:"sourceCode"`
	FlutterCode    string     `json:"flutterCode,omitempty"`
	Status         string     `json:"status"`
	AssignedAgent  string     `json:"assignedAgent,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Error          string     `json:"error,omitempty"`
}

var (
	conversions = make(map[string]*ConversionResponse)
	convMutex   sync.RWMutex
)

func HandleConversions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		submitConversion(w, r)
	case http.MethodGet:
		listConversions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleConversionByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := parts[3]

	switch r.Method {
	case http.MethodGet:
		getConversion(w, r, id)
	case http.MethodPatch:
		updateConversion(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func submitConversion(w http.ResponseWriter, r *http.Request) {
	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.SourceLanguage != "swift" && req.SourceLanguage != "kotlin" {
		http.Error(w, "Invalid source language", http.StatusBadRequest)
		return
	}

	if req.SourceCode == "" {
		http.Error(w, "Source code required", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	now := time.Now()

	conversion := &ConversionResponse{
		ID:             id,
		SourceLanguage: req.SourceLanguage,
		SourceCode:     req.SourceCode,
		Status:         "pending",
		CreatedAt:      now,
	}

	convMutex.Lock()
	conversions[id] = conversion
	convMutex.Unlock()

	// Convert code synchronously (Ollama handles the async work)
	flutterCode, err := convertCodeWithOllama(req.SourceLanguage, req.SourceCode)

	convMutex.Lock()
	if err != nil {
		conversion.Status = "failed"
		conversion.Error = err.Error()
	} else {
		conversion.Status = "completed"
		conversion.FlutterCode = flutterCode
		now := time.Now()
		conversion.CompletedAt = &now
		conversion.AssignedAgent = "converter-ollama"
	}
	conversions[id] = conversion
	convMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conversion)
}

func getConversion(w http.ResponseWriter, r *http.Request, id string) {
	convMutex.RLock()
	conversion, exists := conversions[id]
	convMutex.RUnlock()

	if !exists {
		http.Error(w, "Conversion not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversion)
}

func updateConversion(w http.ResponseWriter, r *http.Request, id string) {
	convMutex.RLock()
	conversion, exists := conversions[id]
	convMutex.RUnlock()

	if !exists {
		http.Error(w, "Conversion not found", http.StatusNotFound)
		return
	}

	var update ConversionResponse
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	convMutex.Lock()
	if update.Status != "" {
		conversion.Status = update.Status
	}
	if update.AssignedAgent != "" {
		conversion.AssignedAgent = update.AssignedAgent
	}
	if update.FlutterCode != "" {
		conversion.FlutterCode = update.FlutterCode
	}
	if update.Error != "" {
		conversion.Error = update.Error
	}

	if update.Status == "completed" || update.Status == "failed" {
		now := time.Now()
		conversion.CompletedAt = &now
	}

	conversions[id] = conversion
	convMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversion)
}

func listConversions(w http.ResponseWriter, r *http.Request) {
	convMutex.RLock()
	var convList []*ConversionResponse
	for _, c := range conversions {
		convList = append(convList, c)
	}
	convMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversions": convList,
		"count":       len(convList),
	})
}

func processConversionAsync(id, sourceLanguage, sourceCode string) {
	// Add small delay to ensure conversion is in map
	time.Sleep(500 * time.Millisecond)

	// Update status to converting
	convMutex.Lock()
	conversion, exists := conversions[id]
	if !exists {
		convMutex.Unlock()
		return
	}
	conversion.Status = "converting"
	conversion.AssignedAgent = "converter-ollama"
	conversions[id] = conversion
	convMutex.Unlock()

	// Call Ollama for conversion
	flutterCode, err := convertCodeWithOllama(sourceLanguage, sourceCode)

	convMutex.Lock()
	conversion = conversions[id]
	if err != nil {
		conversion.Status = "failed"
		conversion.Error = err.Error()
	} else {
		conversion.Status = "completed"
		conversion.FlutterCode = flutterCode
		now := time.Now()
		conversion.CompletedAt = &now
	}
	conversions[id] = conversion
	convMutex.Unlock()
}
