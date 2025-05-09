package api

import (
	alg "backend/internal/algorithm"
	"backend/model"
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) HandleBFS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bfs/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bfs/{elementName}?count=N&singlePath=true", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	singlePath := true
	if singlePathParam := r.URL.Query().Get("singlePath"); singlePathParam != "" {
		parsedValue, err := strconv.ParseBool(singlePathParam)
		if err == nil {
			singlePath = parsedValue
		} else {
		}
	}
	if _, exists := h.elements[elementName]; !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		return
	}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			element := h.elements[base]
			result := model.SearchResult{
				Paths:        [][]model.Node{{{Element: elementName, ImagePath: element.ImagePath}}},
				NodesVisited: 1,
				TimeElapsed:  0,
			}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}
	startTime := time.Now()
	var allPaths [][]model.Node
	var visited int
	if !singlePath && count > 1 {
		explorationCount := count * 15
		if explorationCount > 60 {
			explorationCount = 60
		}
		paths1, visited1 := alg.MultiThreadedBFS(h.elements, elementName, explorationCount, false)
		paths2, visited2 := alg.BFS(h.elements, elementName, count*3, false)
		allPaths = append(paths1, paths2...)
		visited = visited1 + visited2
	} else {
		allPaths, visited = alg.BFS(h.elements, elementName, 1, true)
	}
	var validPaths [][]model.Node
	targetTier := h.elements[elementName].Tier
	for i, path := range allPaths {
		valid := true
		for _, node := range path {
			if node.Element == elementName {
				continue
			}
			if ingredient, exists := h.elements[node.Element]; exists {
				if ingredient.Tier > targetTier {
					log.Printf("DEBUG: Path %d invalid: ingredient %s (tier %d) > target %s (tier %d)",
						i, node.Element, ingredient.Tier, elementName, targetTier)
					valid = false
					break
				}
			}
		}
		if valid {
			validPaths = append(validPaths, path)
		}
	}
	allPaths = validPaths
	timeElapsed := time.Since(startTime).Milliseconds()
	for i := range allPaths {
		for j := range allPaths[i] {
			elem := allPaths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && allPaths[i][j].ImagePath == "" {
				allPaths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}
	var finalPaths [][]model.Node
	if !singlePath && len(allPaths) > 1 {
		pathGroups := make(map[string][]model.Node)
		log.Printf("DEBUG: Grouping paths by base elements for diversity")
		for i, path := range allPaths {
			if len(path) < 2 {
				continue
			}
			var baseElementsUsed []string
			for _, node := range path {
				isBase := false
				for _, base := range baseElements {
					if node.Element == base {
						baseElementsUsed = append(baseElementsUsed, base)
						isBase = true
						break
					}
				}
				if !isBase && len(node.Ingredients) > 0 {
					if len(baseElementsUsed) < 5 {
						baseElementsUsed = append(baseElementsUsed, node.Element)
					}
				}
			}
			sort.Strings(baseElementsUsed)
			signature := strings.Join(baseElementsUsed, ",") + fmt.Sprintf("|len:%d", len(path))
			log.Printf("DEBUG: Path %d has signature: %s", i, signature)
			if _, exists := pathGroups[signature]; !exists {
				pathGroups[signature] = path
				log.Printf("DEBUG: Added path with unique signature: %s", signature)
			}
		}
		for _, path := range pathGroups {
			finalPaths = append(finalPaths, path)
			if len(finalPaths) >= count {
				log.Printf("DEBUG: Selected %d diverse paths, stopping", count)
				break
			}
		}
		if len(finalPaths) < count && len(allPaths) > len(finalPaths) {
			log.Printf("DEBUG: Still need more paths, adding from all paths")
			sort.Slice(allPaths, func(i, j int) bool {
				return len(allPaths[i]) < len(allPaths[j])
			})
			for _, path := range allPaths {
				if len(finalPaths) >= count {
					break
				}
				alreadyIncluded := false
				for _, existingPath := range finalPaths {
					if utils.GeneratePathFingerprint(existingPath) == utils.GeneratePathFingerprint(path) {
						alreadyIncluded = true
						break
					}
				}
				if !alreadyIncluded {
					finalPaths = append(finalPaths, path)
				}
			}
		}
	} else {
		finalPaths = allPaths
	}
	if len(finalPaths) == 0 && len(allPaths) > 0 {
		finalPaths = allPaths[:1]
		log.Printf("DEBUG: No diverse paths found, using first available path")
	} else if len(finalPaths) == 0 {
		element := h.elements[elementName]
		if len(element.Recipes) > 0 {
			log.Printf("DEBUG: Creating manual path from first recipe")
			recipe := element.Recipes[0]
			path := []model.Node{{Element: elementName, ImagePath: element.ImagePath}}
			for _, ing := range recipe.Ingredients {
				if ingElement, exists := h.elements[ing]; exists {
					if ingElement.Tier <= targetTier {
						path = append([]model.Node{{
							Element:   ing,
							ImagePath: ingElement.ImagePath,
						}}, path...)
					}
				}
			}
			finalPaths = [][]model.Node{path}
		} else {
			finalPaths = [][]model.Node{{{Element: elementName, ImagePath: element.ImagePath}}}
			log.Printf("DEBUG: No recipes available, returning just the target element")
		}
	}
	log.Printf("DEBUG: Final result contains %d paths", len(finalPaths))
	result := model.SearchResult{
		Paths:        finalPaths,
		NodesVisited: visited,
		TimeElapsed:  timeElapsed,
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
	log.Printf("DEBUG: Successfully sent BFS response with %d paths", len(finalPaths))
}

func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("DEBUG: Starting HandleSearch request")
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
	log.Printf("DEBUG: Searching for %s using %s algorithm (max results: %d, single path: %v)",
		config.TargetElement, config.Algorithm, config.MaxResults, config.SinglePath)

	startTime := time.Now()
	var result model.SearchResult
	switch config.Algorithm {
	case "bfs":
		paths, visited := alg.BFS(h.elements, config.TargetElement, config.MaxResults, true)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: BFS found %d paths after visiting %d nodes", len(paths), visited)
	case "dfs":
		paths, visited := alg.DFS(h.elements, config.TargetElement, config.MaxResults, false)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)
	case "bidirectional":
		paths, visited := alg.Bidirectional(h.elements, config.TargetElement, config.MaxResults)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: Bidirectional search found %d paths after visiting %d nodes", len(paths), visited)
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
	log.Printf("DEBUG: Successfully sent response in %d ms", time.Since(startTime).Milliseconds())
}
