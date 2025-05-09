package main

import (
	"backend/api"
	"backend/internal"
	"log"
	"net/http"
)

func main() {
	elements, elementGraph, err := internal.LoadElements()
	if err != nil {
		log.Fatalf("Failed to load elements: %v", err)
	}

	log.Printf("Successfully loaded %d elements", len(elements))
	log.Printf("Graph built with %d nodes and %d base elements",
		len(elementGraph.Nodes), len(elementGraph.BaseElements))

	handler := api.NewHandler(elements)

	// Create the CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Create a router and wrap all handlers with CORS middleware
	mux := http.NewServeMux()

	// Apply CORS middleware to all routes
	mux.Handle("/api/elements/", corsMiddleware(http.HandlerFunc(handler.HandleGetElements)))
	mux.Handle("/api/search", corsMiddleware(http.HandlerFunc(handler.HandleSearch)))
	mux.Handle("/api/elements", corsMiddleware(http.HandlerFunc(handler.HandleGetElements)))
	mux.Handle("/api/recipes/", corsMiddleware(http.HandlerFunc(handler.HandleRecipePath)))
	mux.Handle("/api/tree/", corsMiddleware(http.HandlerFunc(handler.HandleElementTree)))
	mux.Handle("/api/best-recipes/", corsMiddleware(http.HandlerFunc(handler.HandleBestRecipes)))
	mux.Handle("/api/multiple-recipes/", corsMiddleware(http.HandlerFunc(handler.HandleMultipleRecipes)))
	mux.Handle("/api/best-recipes-tree/", corsMiddleware(http.HandlerFunc(handler.HandleBestRecipesTree)))
	mux.Handle("/api/multiple-recipes-tree/", corsMiddleware(http.HandlerFunc(handler.HandleMultipleRecipesTree)))
	mux.Handle("/api/bfs/", corsMiddleware(http.HandlerFunc(handler.HandleBFS)))
	mux.Handle("/api/mt-bfs-recipes-tree/", corsMiddleware(http.HandlerFunc(handler.HandleMultiThreadedBFSRecipesTree)))
	port := ":8080"
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
