package api

import (
	alg "backend/internal/algorithm"
	"backend/model"
	"backend/utils"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) HandleElementTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tree/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL format. Use /api/tree/{algorithm}/{elementName}", http.StatusBadRequest)
		return
	}
	algorithm := pathParts[0]
	elementName := strings.Join(pathParts[1:], "/")
	targetElement, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		return
	}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			result := map[string]interface{}{
				"name":          elementName,
				"imagePath":     targetElement.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}
	startTime := time.Now()
	var result map[string]interface{}
	var visitedNodes int
	g := utils.CreateElementGraph(h.elements)
	switch strings.ToLower(algorithm) {
	case "bfs":
		result, visitedNodes = alg.GetElementTreeBFS(g, elementName)
	case "dfs":
		result, visitedNodes = alg.GetElementTreeDFS(g, elementName)
	default:
		http.Error(w, "Invalid algorithm. Use 'bfs' or 'dfs'", http.StatusBadRequest)
		return
	}
	finalResult := map[string]interface{}{
		"tree":         result,
		"nodesVisited": visitedNodes,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
	}
	if err := json.NewEncoder(w).Encode(finalResult); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

func (h *Handler) HandleBestRecipes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("DEBUG: Starting HandleBestRecipes request")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/best-recipes/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/best-recipes/{elementName}?count=N", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)
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
	maxResults := count + 5
	if maxResults > 20 {
		maxResults = 20
		log.Printf("DEBUG: Limiting exploration to 20 paths")
	}
	paths, visited := alg.DFS(h.elements, elementName, maxResults, false)
	log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)

	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j])
	})
	log.Printf("DEBUG: Sorted paths by length (shortest first)")
	if len(paths) > count {
		paths = paths[:count]
		log.Printf("DEBUG: Taking only the top %d shortest paths", count)
	}

	for i := range paths {
		for j := range paths[i] {
			elem := paths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && paths[i][j].ImagePath == "" {
				paths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}

	result := model.SearchResult{
		Paths:        paths,
		NodesVisited: visited,
		TimeElapsed:  time.Since(startTime).Milliseconds(),
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response with %d recipes", len(paths))
}

func (h *Handler) HandleMultipleRecipesTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleMultipleRecipesTree request")

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/multiple-recipes-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/multiple-recipes-tree/{elementName}?count=N&algorithm=algo", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	count := 3
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe tree count: %d", count)

	algorithm := "dfs"
	if algoParam := r.URL.Query().Get("algorithm"); algoParam != "" {
		algorithm = strings.ToLower(algoParam)
	}
	log.Printf("DEBUG: Using algorithm: %s", algorithm)

	if count > 10 {
		count = 10
		log.Printf("DEBUG: Limiting count to maximum of 10 for tree format")
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
			result := map[string]interface{}{
				"trees": []map[string]interface{}{{
					"name":          elementName,
					"imagePath":     element.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}},
				"nodesVisited": 1,
				"timeElapsed":  0,
				"algorithm":    algorithm,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	startTime := time.Now()
	g := utils.CreateElementGraph(h.elements)

	var finalTrees []map[string]interface{}
	var totalVisitedNodesCount int

	if algorithm == "bfs" {
		paths, visited := alg.MultiThreadedBFS(h.elements, elementName, count*3, false)
		totalVisitedNodesCount = visited

		uniqueTrees := make([]map[string]interface{}, 0)
		uniqueSignatures := make(map[string]bool)

		for _, path := range paths {
			if len(path) < 2 {
				continue
			}

			pathVisitCount := 0
			tree := utils.ConvertPathToCompleteTree(path, h.elements, &pathVisitCount, algorithm)

			signature := utils.GenerateDetailedTreeSignature(tree)
			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				uniqueTrees = append(uniqueTrees, tree)
				totalVisitedNodesCount += pathVisitCount

				if len(uniqueTrees) >= count {
					break
				}
			}
		}

		finalTrees = uniqueTrees
	} else {
		trees, visited := utils.GenerateAllRecipeVariations(g, elementName, element.ImagePath, count)
		finalTrees = trees
		totalVisitedNodesCount = visited
		log.Printf("DEBUG: Generated %d unique recipe trees after visiting %d nodes",
			len(trees), visited)
	}

	if len(finalTrees) == 0 {
		visited := make(map[string]bool)
		visitCount := 0
		var tree map[string]interface{}

		if algorithm == "bfs" {
			tree = utils.BuildElementTreeBFS(g, elementName, visited, &visitCount)
		} else {
			tree = utils.BuildElementTreeBFS(g, elementName, visited, &visitCount)
		}

		finalTrees = []map[string]interface{}{tree}
		totalVisitedNodesCount += visitCount
		log.Printf("DEBUG: Added fallback element tree using %s (nodes visited: %d)",
			strings.ToUpper(algorithm), visitCount)
	}

	result := map[string]interface{}{
		"trees":        finalTrees,
		"nodesVisited": totalVisitedNodesCount,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
		"algorithm":    algorithm,
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

func (h *Handler) HandleBestRecipesTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleBestRecipesTree request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/best-recipes-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/best-recipes-tree/{elementName}?count=N&algorithm=algo", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)
	algorithm := "bfs"
	if algoParam := r.URL.Query().Get("algorithm"); algoParam != "" {
		algorithm = strings.ToLower(algoParam)
	}
	log.Printf("DEBUG: Using algorithm: %s", algorithm)
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
			result := map[string]interface{}{
				"trees": []map[string]interface{}{{
					"name":          elementName,
					"imagePath":     element.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}},
				"nodesVisited": 1,
				"timeElapsed":  0,
				"algorithm":    algorithm,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}
	startTime := time.Now()
	g := utils.CreateElementGraph(h.elements)
	recipeTrees := make([]map[string]interface{}, 0, count)
	visitedNodesCount := 0
	node := g.Nodes[elementName]
	if len(node.RecipesToMakeThisElement) == 0 {
		tree := map[string]interface{}{
			"name":        elementName,
			"imagePath":   element.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
		recipeTrees = append(recipeTrees, tree)
	} else {
		for _, recipe := range node.RecipesToMakeThisElement {
			if len(recipeTrees) >= count {
				break
			}
			localVisitCount := 0

			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   element.ImagePath,
				"ingredients": []interface{}{},
			}

			ingredients := make([]interface{}, 0, len(recipe.Ingredients))
			for _, ingredientName := range recipe.Ingredients {
				ingredientVisited := make(map[string]bool)
				ingredientVisitCount := 0

				var ingredientTree map[string]interface{}
				if algorithm == "bfs" {
					ingredientTree = utils.BuildElementTreeBFS(g, ingredientName, ingredientVisited, &ingredientVisitCount)
				} else {
					ingredientTree = utils.BuildElementTreeDFS(g, ingredientName, ingredientVisited, &ingredientVisitCount)
				}

				ingredients = append(ingredients, ingredientTree)
				localVisitCount += ingredientVisitCount
			}

			tree["ingredients"] = ingredients
			visitedNodesCount += localVisitCount

			isUnique := true
			for _, existingTree := range recipeTrees {
				if utils.CompareTreeIngredients(existingTree, tree) {
					isUnique = false
					break
				}
			}

			if isUnique {
				recipeTrees = append(recipeTrees, tree)
				log.Printf("")
				log.Printf("DEBUG: Added recipe tree using %s algorithm with recipe containing %d ingredients", algorithm, len(recipe.Ingredients))
			}
		}

		if len(recipeTrees) < count {
			maxResults := count * 2
			if maxResults > 10 {
				maxResults = 10
			}

			var paths [][]model.Node
			var visited int

			switch algorithm {
			case "bfs":
				// Dari:
				paths, visited = alg.BFS(h.elements, elementName, maxResults, false)
			case "dfs":
				// Default to DFS
				paths, visited = alg.DFS(h.elements, elementName, maxResults, false)
			}

			log.Printf("DEBUG: %s found %d paths after visiting %d nodes",
				strings.ToUpper(algorithm), len(paths), visited)
			visitedNodesCount += visited

			sort.Slice(paths, func(i, j int) bool {
				return len(paths[i]) < len(paths[j])
			})

			for i, path := range paths {
				if len(recipeTrees) >= count {
					break
				}

				if len(path) < 2 {
					continue
				}

				g := utils.CreateElementGraph(h.elements)

				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": []interface{}{},
				}

				ingredientSet := make(map[string]bool)
				for i := 1; i < len(path); i++ {
					if path[i].Element != elementName {
						ingredientSet[path[i].Element] = true
					}
				}

				ingredients := make([]interface{}, 0)
				for ingredient := range ingredientSet {
					isDirectIngredient := false
					for _, recipe := range node.RecipesToMakeThisElement {
						for _, ing := range recipe.Ingredients {
							if ing == ingredient {
								isDirectIngredient = true
								break
							}
						}
						if isDirectIngredient {
							break
						}
					}

					if isDirectIngredient {
						ingredientVisited := make(map[string]bool)
						ingredientVisitCount := 0

						var ingredientTree map[string]interface{}
						if algorithm == "bfs" {
							ingredientTree = utils.BuildElementTreeBFS(g, ingredient, ingredientVisited, &ingredientVisitCount)
						} else {
							ingredientTree = utils.BuildElementTreeDFS(g, ingredient, ingredientVisited, &ingredientVisitCount)
						}

						ingredients = append(ingredients, ingredientTree)
						visitedNodesCount += ingredientVisitCount
					}
				}
				if len(ingredients) > 0 {
					tree["ingredients"] = ingredients
					isUnique := true
					for _, existingTree := range recipeTrees {
						if utils.CompareTreeIngredients(existingTree, tree) {
							isUnique = false
							break
						}
					}
					if isUnique {
						recipeTrees = append(recipeTrees, tree)
						log.Printf("DEBUG: Added alternative recipe tree from path %d", i+1)
					}
				}
			}
		}
	}
	if len(recipeTrees) == 0 {
		visited := make(map[string]bool)
		visitCount := 0
		var mainTree map[string]interface{}
		if algorithm == "bfs" {
			mainTree = utils.BuildElementTreeBFS(g, elementName, visited, &visitCount)
		} else {
			mainTree = utils.BuildElementTreeDFS(g, elementName, visited, &visitCount)
		}
		recipeTrees = append(recipeTrees, mainTree)
		visitedNodesCount += visitCount
		log.Printf("DEBUG: Added fallback element tree using %s", strings.ToUpper(algorithm))
	}
	result := map[string]interface{}{
		"trees":        recipeTrees,
		"nodesVisited": visitedNodesCount,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
		"algorithm":    algorithm,
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
	log.Printf("DEBUG: Successfully sent response with %d recipe trees", len(recipeTrees))
}
