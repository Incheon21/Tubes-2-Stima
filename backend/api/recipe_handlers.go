package api

import (
	alg "backend/internal/algorithm"
	"backend/model"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) HandleMultipleRecipes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("DEBUG: Starting HandleMultipleRecipes request")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/multiple-recipes/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/multiple-recipes/{elementName}?count=N", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)
	count := 5
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)
	if count > 10 {
		count = 10
		log.Printf("DEBUG: Limiting count to maximum of 10")
	}
	element, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			result := model.SearchResult{
				Paths: [][]model.Node{{{
					Element:   elementName,
					ImagePath: element.ImagePath,
				}}},
				NodesVisited: 0,
				TimeElapsed:  0,
			}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}
	log.Printf("DEBUG: Starting DFS search for element '%s'", elementName)
	startTime := time.Now()
	explorationLimit := count * 2
	if explorationLimit > 20 {
		explorationLimit = 20
		log.Printf("DEBUG: Limiting exploration to 20 paths")
	} else {
		log.Printf("DEBUG: Setting exploration limit to %d paths", explorationLimit)
	}
	paths, visited := alg.DFS(h.elements, elementName, explorationLimit, true)
	log.Printf("DEBUG: DFS visited %d nodes", visited)
	log.Printf("DEBUG: DFS found %d paths", len(paths))
	log.Printf("DEBUG: Grouping paths by base elements used")
	pathGroups := make(map[string][]model.Node)

	for i, path := range paths {
		if len(path) < 3 {
			log.Printf("DEBUG: Skipping path %d (too short, only %d nodes)", i, len(path))
			continue
		}

		var baseElementsUsed []string
		for _, node := range path {
			isBaseElement := false
			for _, base := range baseElements {
				if node.Element == base {
					baseElementsUsed = append(baseElementsUsed, base)
					isBaseElement = true
				}
			}
			if !isBaseElement && len(node.Ingredients) == 0 {
				baseElementsUsed = append(baseElementsUsed, node.Element)
			}
		}

		sort.Strings(baseElementsUsed)
		fingerprint := strings.Join(baseElementsUsed, ",")
		log.Printf("DEBUG: Path %d has fingerprint: %s", i, fingerprint)

		if _, exists := pathGroups[fingerprint]; !exists {
			pathGroups[fingerprint] = path
			log.Printf("DEBUG: Added path with unique fingerprint: %s", fingerprint)
		}
	}

	log.Printf("DEBUG: Found %d unique path groups", len(pathGroups))

	diversePaths := make([][]model.Node, 0)
	for fingerprint, path := range pathGroups {
		diversePaths = append(diversePaths, path)
		log.Printf("DEBUG: Selected path with fingerprint: %s", fingerprint)
		if len(diversePaths) >= count {
			log.Printf("DEBUG: Reached requested count of %d diverse paths", count)
			break
		}
	}
	if len(diversePaths) < count && len(paths) > len(diversePaths) {
		log.Printf("DEBUG: Not enough diverse paths (%d/%d), adding more from original paths",
			len(diversePaths), count)
		sort.Slice(paths, func(i, j int) bool {
			return len(paths[i]) < len(paths[j])
		})
		log.Printf("DEBUG: Sorted original paths by length (shortest first)")
		for i, path := range paths {
			if len(diversePaths) >= count {
				break
			}
			isIncluded := false
			for _, dp := range diversePaths {
				if len(path) > 0 && len(dp) > 0 &&
					path[0].Element == dp[0].Element &&
					path[len(path)-1].Element == dp[len(dp)-1].Element {
					isIncluded = true
					break
				}
			}
			if !isIncluded {
				diversePaths = append(diversePaths, path)
				log.Printf("DEBUG: Added additional path %d (length: %d)", i, len(path))
			}
		}
	}
	log.Printf("DEBUG: Final diverse path count: %d", len(diversePaths))
	for i := range diversePaths {
		for j := range diversePaths[i] {
			elem := diversePaths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && diversePaths[i][j].ImagePath == "" {
				diversePaths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}
	log.Printf("DEBUG: Processing completed in %d ms", time.Since(startTime).Milliseconds())
	result := model.SearchResult{
		Paths:        diversePaths,
		NodesVisited: visited,
		TimeElapsed:  time.Since(startTime).Milliseconds(),
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
	log.Printf("DEBUG: Successfully sent response with %d recipes", len(diversePaths))
}

func (h *Handler) HandleRecipePath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("DEBUG: Starting HandleRecipePath request")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/recipes/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL format. Use /api/recipes/{algorithm}/{elementName}", http.StatusBadRequest)
		return
	}
	algorithm := pathParts[0]
	elementName := strings.Join(pathParts[1:], "/") // In case element name has slashes
	log.Printf("DEBUG: Requested algorithm: %s, element: %s", algorithm, elementName)
	if _, exists := h.elements[elementName]; !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			result := model.SearchResult{
				Paths:        [][]model.Node{{{Element: elementName}}},
				NodesVisited: 1,
				TimeElapsed:  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}
	config := model.SearchConfig{
		MaxResults: 1,
		SinglePath: true,
	}
	if maxResults := r.URL.Query().Get("maxResults"); maxResults != "" {
		var err error
		if config.MaxResults, err = strconv.Atoi(maxResults); err != nil {
			config.MaxResults = 1 // Default to 1 if invalid
		}
	}
	startTime := time.Now()
	var result model.SearchResult
	log.Printf("DEBUG: Starting search with algorithm: %s for element: %s (max results: %d)",
		algorithm, elementName, config.MaxResults)

	switch strings.ToLower(algorithm) {
	case "bfs":
		paths, visited := alg.BFS(h.elements, elementName, config.MaxResults, true)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: BFS found %d paths after visiting %d nodes", len(paths), visited)
	case "dfs":
		paths, visited := alg.DFS(h.elements, elementName, config.MaxResults, false)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)
	case "bidirectional":
		paths, visited := alg.Bidirectional(h.elements, elementName, config.MaxResults)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: Bidirectional search found %d paths after visiting %d nodes", len(paths), visited)
	default:
		http.Error(w, "Invalid algorithm. Use 'bfs', 'dfs', or 'bidirectional'", http.StatusBadRequest)
		return
	}
	result.TimeElapsed = time.Since(startTime).Milliseconds()
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
	log.Printf("DEBUG: Successfully sent response in %d ms", time.Since(startTime).Milliseconds())
}
