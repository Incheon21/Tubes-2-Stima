package api

import (
	"backend/internal/algorithm"
	"backend/model"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	elements map[string]model.Element
}

func NewHandler(elements map[string]model.Element) *Handler {
	return &Handler{elements: elements}
}

func (h *Handler) HandleGetElements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	elementList := make([]model.Element, 0, len(h.elements))
	for _, elem := range h.elements {
		elementList = append(elementList, elem)
	}

	if err := json.NewEncoder(w).Encode(elementList); err != nil {
		http.Error(w, "Failed to encode elements", http.StatusInternalServerError)
		log.Printf("Error encoding elements: %v", err)
		return
	}
}

func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config model.SearchConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request: %v", err)
		return
	}

	if config.TargetElement == "" {
		http.Error(w, "Target element is required", http.StatusBadRequest)
		return
	}

	if config.Algorithm == "" {
		config.Algorithm = "bfs"
	}
	if config.MaxResults <= 0 {
		config.MaxResults = 1
	}

	startTime := time.Now()
	var result model.SearchResult

	switch config.Algorithm {
	case "bfs":
		paths, visited := algorithm.BFS(h.elements, config.TargetElement, config.MaxResults, config.SinglePath)
		result.Paths = paths
		result.NodesVisited = visited
	case "dfs":
		paths, visited := algorithm.DFS(h.elements, config.TargetElement, config.MaxResults, config.SinglePath)
		result.Paths = paths
		result.NodesVisited = visited
	case "bidirectional":
		paths, visited := algorithm.Bidirectional(h.elements, config.TargetElement, config.MaxResults)
		result.Paths = paths
		result.NodesVisited = visited
	default:
		http.Error(w, "Invalid algorithm", http.StatusBadRequest)
		return
	}

	result.TimeElapsed = time.Since(startTime).Milliseconds()

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}
