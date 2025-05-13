package api

import (
	"backend/model"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (h *Handler) HandleGetElements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/elements/")
	if path != "" && path != "elements" {
		elementName := strings.TrimSpace(path)
		element, exists := h.elements[elementName]
		if !exists {
			http.Error(w, "Element not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(element); err != nil {
			http.Error(w, "Failed to encode element", http.StatusInternalServerError)
			log.Printf("Error encoding element: %v", err)
		}
		return
	}
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
