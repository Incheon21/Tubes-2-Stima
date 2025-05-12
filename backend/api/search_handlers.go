package api

import (
	alg "backend/internal/algorithm"
	"backend/internal/graph"
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

func (h *Handler) HandleBFSTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bfs-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bfs-tree/{elementName}?count=N", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")

	log.Printf("DEBUG: BFS Tree request for element: %s", elementName)

	count := 3
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
			log.Printf("DEBUG: Requested tree count: %d", count)
		}
	}

	useMultithreaded := false
	if mtParam := r.URL.Query().Get("multithreaded"); mtParam != "" {
		useMultithreaded = mtParam == "true"
		log.Printf("DEBUG: Multithreaded mode: %v", useMultithreaded)
	} else if count > 1 {
		useMultithreaded = true
		log.Printf("DEBUG: Count > 1, automatically using multithreaded BFS")
	}

	algoName := "bfs"
	if useMultithreaded {
		algoName = "multithreaded-bfs"
	}

	// Validate element exists
	_, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// Handle base elements quickly
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	isBaseElement := false
	for _, base := range baseElements {
		if elementName == base {
			isBaseElement = true
			break
		}
	}

	if isBaseElement {
		log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
		elementData := h.elements[elementName]
		result := map[string]interface{}{
			"trees": []map[string]interface{}{{
				"name":          elementName,
				"imagePath":     elementData.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}},
			"nodesVisited": 1,
			"timeElapsed":  0,
			"algorithm":    algoName,
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Printf("Error encoding response: %v", err)
		}
		return
	}

	log.Printf("DEBUG: Starting %s tree search for element '%s'", algoName, elementName)
	startTime := time.Now()

	var paths [][]model.Node
	var visitedCount int

	if useMultithreaded {
		paths, visitedCount = alg.MultiThreadedBFS(h.elements, elementName, count*2, false)
	} else {
		paths, visitedCount = alg.BFS(h.elements, elementName, count*2, false)
	}

	log.Printf("DEBUG: %s found %d paths after visiting %d nodes", algoName, len(paths), visitedCount)

	trees := make([]map[string]interface{}, 0, len(paths))
	uniqueSignatures := make(map[string]bool)

	for _, path := range paths {
		tree := convertPathToTree(path, elementName, h.elements, baseElements)

		ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

		if tree != nil {
			signature := generateDetailedTreeSignature(tree)

			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				trees = append(trees, tree)
			}
		}
	}

	log.Printf("DEBUG: Generated %d unique trees from %d paths", len(trees), len(paths))

	if len(trees) > count {
		trees = trees[:count]
	}
	totalNodeCount := 0
	for _, tree := range trees {
		totalNodeCount += countNodesInTree(tree)
	}

	log.Printf("DEBUG: Total nodes in final trees: %d", totalNodeCount)

	timeElapsed := time.Since(startTime).Milliseconds()

	result := map[string]interface{}{
		"trees":          trees,
		"nodesVisited":   visitedCount,
		"totalTreeNodes": totalNodeCount,
		"timeElapsed":    timeElapsed,
		"algorithm":      algoName,
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent %s tree response with %d trees in %d ms",
		algoName, len(trees), timeElapsed)
}

func countNodesInTree(tree map[string]interface{}) int {
	if tree == nil {
		return 0
	}

	count := 1 // Count the current node

	// Count nodes in ingredient subtrees
	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok {
		return count
	}

	for _, ing := range ingredients {
		ingTree, ok := ing.(map[string]interface{})
		if ok {
			count += countNodesInTree(ingTree)
		}
	}

	return count
}

func ensureIngredientsExpanded(tree map[string]interface{}, elements map[string]model.Element, baseElements []string, visited map[string]bool) {
	if tree == nil {
		return
	}

	elementName, ok := tree["name"].(string)
	if !ok || visited[elementName] {
		return
	}

	visited[elementName] = true
	defer delete(visited, elementName)

	isBase := false
	for _, base := range baseElements {
		if elementName == base {
			isBase = true
			break
		}
	}

	if isBase {
		tree["isBaseElement"] = true
		return
	}

	ingredients, ok := tree["ingredients"].([]interface{})

	if (!ok || len(ingredients) == 0) && !isBase {
		if elemData, exists := elements[elementName]; exists && len(elemData.Recipes) > 0 {
			recipe := elemData.Recipes[0]
			newIngredients := make([]interface{}, 0, len(recipe.Ingredients))

			for _, ingName := range recipe.Ingredients {
				ingIsBase := false
				for _, base := range baseElements {
					if ingName == base {
						ingIsBase = true
						break
					}
				}

				ingData, ingExists := elements[ingName]
				if !ingExists {
					continue
				}

				ingTree := map[string]interface{}{
					"name":          ingName,
					"imagePath":     ingData.ImagePath,
					"isBaseElement": ingIsBase,
					"ingredients":   []interface{}{},
				}

				if !ingIsBase {
					ensureIngredientsExpanded(ingTree, elements, baseElements, visited)
				}

				newIngredients = append(newIngredients, ingTree)
			}

			tree["ingredients"] = newIngredients
		}
	} else {
		for _, ing := range ingredients {
			if ingTree, ok := ing.(map[string]interface{}); ok {
				ensureIngredientsExpanded(ingTree, elements, baseElements, visited)
			}
		}
	}
}

func convertPathToTree(path []model.Node, targetElement string, elements map[string]model.Element, baseElements []string) map[string]interface{} {
	if len(path) == 0 {
		return nil
	}

	var targetNode *model.Node
	for i := range path {
		if path[i].Element == targetElement {
			targetNode = &path[i]
			break
		}
	}

	if targetNode == nil {
		return nil
	}

	nodeMap := make(map[string]*model.Node)
	for i := range path {
		nodeMap[path[i].Element] = &path[i]
	}

	processedInBranch := make(map[string]bool)

	var buildTree func(element string, depth int) map[string]interface{}
	buildTree = func(element string, depth int) map[string]interface{} {
		if processedInBranch[element] {
			return map[string]interface{}{
				"name":                element,
				"isCircularReference": true,
				"ingredients":         []interface{}{},
			}
		}

		processedInBranch[element] = true
		defer func() {
			delete(processedInBranch, element)
		}()

		node, found := nodeMap[element]
		if !found {
			elemData, exists := elements[element]
			if !exists {
				return nil
			}

			isBase := false
			for _, base := range baseElements {
				if element == base {
					isBase = true
					break
				}
			}

			treeNode := map[string]interface{}{
				"name":          element,
				"imagePath":     elemData.ImagePath,
				"isBaseElement": isBase,
				"ingredients":   []interface{}{},
			}

			// If it's not a base element, try to expand its ingredients
			if !isBase && depth < 10 && len(elemData.Recipes) > 0 {
				recipe := elemData.Recipes[0]
				for _, ingredient := range recipe.Ingredients {
					// Recursively build subtree for this ingredient
					subtree := buildTree(ingredient, depth+1)
					if subtree != nil {
						treeNode["ingredients"] = append(treeNode["ingredients"].([]interface{}), subtree)
					}
				}
			}

			return treeNode
		}

		isBase := false
		for _, base := range baseElements {
			if element == base {
				isBase = true
				break
			}
		}

		treeNode := map[string]interface{}{
			"name":        element,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
		}

		if isBase {
			treeNode["isBaseElement"] = true
			return treeNode
		}

		ingredients := node.Ingredients
		if ingredients == nil || len(ingredients) == 0 {
			// Try to get from element data
			if elemData, exists := elements[element]; exists && len(elemData.Recipes) > 0 {
				ingredients = elemData.Recipes[0].Ingredients
			}
		}

		if depth < 10 {
			for _, ingredient := range ingredients {
				subtree := buildTree(ingredient, depth+1)
				if subtree != nil {
					treeNode["ingredients"] = append(treeNode["ingredients"].([]interface{}), subtree)
				}
			}
		}

		return treeNode
	}

	return buildTree(targetElement, 0)
}

func (h *Handler) HandleDFSTree(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/dfs-tree/"), "/")
    if len(pathParts) < 1 {
        http.Error(w, "Invalid URL format. Use /api/dfs-tree/{elementName}?count=N", http.StatusBadRequest)
        return
    }
    elementName := strings.Join(pathParts, "/")

    count := 5
    if countParam := r.URL.Query().Get("count"); countParam != "" {
        if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
            count = parsedCount
        }
    }

    if elementName == "Metal" && count < 10 {
        count = 10
        log.Printf("DEBUG: Element Metal detected, increasing count to ensure all variations")
    }

    // Validate element exists
    element, exists := h.elements[elementName]
    if !exists {
        http.Error(w, "Element not found", http.StatusNotFound)
        log.Printf("DEBUG: Element '%s' not found in database", elementName)
        return
    }

    // Handle base elements quickly
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    isBaseElement := false
    for _, base := range baseElements {
        if elementName == base {
            isBaseElement = true
            break
        }
    }

    if isBaseElement {
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
            "algorithm":    "dfs",
        }

        if err := json.NewEncoder(w).Encode(result); err != nil {
            http.Error(w, "Failed to encode response", http.StatusInternalServerError)
            log.Printf("Error encoding response: %v", err)
        }
        return
    }

    log.Printf("DEBUG: Starting DFS tree search for element '%s' (requesting %d trees)",
        elementName, count)
    startTime := time.Now()

    // Create the graph from elements once
    g := graph.NewElementGraph(h.elements)
    
    // Use MultiThreadedElementTreeDFS from your DFS implementation
    trees, visitedCount := alg.MultiThreadedElementTreeDFS(g, elementName, count)

    // If no trees were found, fall back to a simpler approach
    if len(trees) == 0 {
        log.Printf("DEBUG: No trees found with MultiThreadedElementTreeDFS, falling back to simpler DFS approach")
        
        // Use GetElementTreeDFS for a single tree as fallback
        tree, singleVisitCount := alg.GetElementTreeDFS(g, elementName)
        trees = []map[string]interface{}{tree}
        visitedCount = singleVisitCount
        
        log.Printf("DEBUG: Generated fallback tree with DFS (nodes visited: %d)", visitedCount)
    }

    // Ensure we have enough variations for Metal element
    minCount := count
    if elementName == "Metal" && count < 5 {
        minCount = 5
        log.Printf("DEBUG: Ensuring at least 5 trees for Metal element")
    }

    // Limit trees to requested count
    if len(trees) > minCount {
        trees = trees[:minCount]
        log.Printf("DEBUG: Limited trees to requested count: %d", minCount)
    }

    // Process trees to ensure consistent format with previous implementation
    for i := range trees {
        ensureIngredientsExpanded(trees[i], h.elements, baseElements, make(map[string]bool))
    }

    timeElapsed := time.Since(startTime).Milliseconds()

    // Create response
    result := map[string]interface{}{
        "trees":        trees,
        "nodesVisited": visitedCount,
        "timeElapsed":  timeElapsed,
        "algorithm":    "dfs",
    }

    if err := json.NewEncoder(w).Encode(result); err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        log.Printf("Error encoding response: %v", err)
        return
    }

    log.Printf("DEBUG: Successfully sent DFS tree response with %d trees in %d ms",
        len(trees), timeElapsed)
}

func generateDetailedTreeSignature(tree map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(tree["name"].(string))
	sb.WriteString(":")

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return sb.String() + "[]"
	}

	ingredientSignatures := make([]string, 0, len(ingredients))

	for _, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}

		ingredientSig := generateDetailedTreeSignature(ingredient)
		ingredientSignatures = append(ingredientSignatures, ingredientSig)
	}

	sort.Strings(ingredientSignatures)

	sb.WriteString("[")
	sb.WriteString(strings.Join(ingredientSignatures, ","))
	sb.WriteString("]")

	return sb.String()
}


func (h *Handler) HandleBidirectionalSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bidirectional/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bidirectional/{elementName}", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")

	log.Printf("DEBUG: Bidirectional search request for element: %s", elementName)

	count := 3
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
			log.Printf("DEBUG: Requested result count: %d", count)
		}
	}

	useMultithreaded := true
	if mtParam := r.URL.Query().Get("multithreaded"); mtParam != "" {
		useMultithreaded = mtParam == "true"
		log.Printf("DEBUG: Multithreaded mode: %v", useMultithreaded)
	}

	singlePath := false
	if singleParam := r.URL.Query().Get("single"); singleParam == "true" {
		singlePath = true
		log.Printf("DEBUG: Single path mode enabled")
	}

	treeView := false
	if treeParam := r.URL.Query().Get("tree"); treeParam == "true" {
		treeView = true
		log.Printf("DEBUG: Tree visualization requested")
	}

	_, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	isBaseElement := false
	for _, base := range baseElements {
		if elementName == base {
			isBaseElement = true
			break
		}
	}

	if isBaseElement {
		log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)

		if treeView {
			elementData := h.elements[elementName]
			result := map[string]interface{}{
				"trees": []map[string]interface{}{{
					"name":          elementName,
					"imagePath":     elementData.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}},
				"nodesVisited": 1,
				"timeElapsed":  0,
				"algorithm":    "bidirectional",
			}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("ERROR: Failed to encode response: %v", err)
			}
		} else {
			result := map[string]interface{}{
				"paths":        []map[string]interface{}{},
				"nodesVisited": 0,
				"timeElapsed":  0,
				"algorithm":    "bidirectional",
			}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("ERROR: Failed to encode response: %v", err)
			}
		}
		return
	}

	log.Printf("DEBUG: Starting bidirectional search for element '%s'", elementName)
	startTime := time.Now()

	// Request more paths to ensure we get diversity in recipes
	// We're especially looking for multiple recipes
	searchCount := count * 15

	var paths [][]model.Node
	var visitedCount int
	var algoName string

	if useMultithreaded {
		algoName = "multithreaded-bidirectional"
		paths, visitedCount = alg.MultiThreadedBidirectionalBFS(h.elements, elementName, searchCount, singlePath)
	} else {
		algoName = "bidirectional"
		paths, visitedCount = alg.BidirectionalBFS(h.elements, elementName, searchCount, singlePath)
	}

	timeElapsed := time.Since(startTime).Milliseconds()

	// Group by recipe for analytics
	recipeGroups := make(map[string][][]model.Node)
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}

		var targetNode *model.Node
		for i := range path {
			if path[i].Element == elementName {
				targetNode = &path[i]
				break
			}
		}

		if targetNode != nil && targetNode.Ingredients != nil && len(targetNode.Ingredients) >= 2 {
			sortedIngs := make([]string, len(targetNode.Ingredients))
			copy(sortedIngs, targetNode.Ingredients)
			sort.Strings(sortedIngs)
			recipeKey := strings.Join(sortedIngs, "+")
			recipeGroups[recipeKey] = append(recipeGroups[recipeKey], path)
		}
	}

	log.Printf("DEBUG: %s search found %d paths (%d unique recipes) after visiting %d nodes in %d ms",
		algoName, len(paths), len(recipeGroups), visitedCount, timeElapsed)

	if treeView {
		log.Printf("DEBUG: Processing %d paths to create tree visualizations", len(paths))

		trees := make([]map[string]interface{}, 0)
		uniqueSignatures := make(map[string]bool)

		element := h.elements[elementName]
		recipeList := element.Recipes

		log.Printf("DEBUG: Element '%s' has %d recipes in database", elementName, len(recipeList))

		// Group recipes by their keys for better matching
		dbRecipesByKey := make(map[string]model.ElementRecipe)
		for _, recipe := range recipeList {
			if len(recipe.Ingredients) >= 2 {
				sortedIngs := make([]string, len(recipe.Ingredients))
				copy(sortedIngs, recipe.Ingredients)
				sort.Strings(sortedIngs)
				recipeKey := strings.Join(sortedIngs, "+")
				dbRecipesByKey[recipeKey] = recipe
			}
		}

		// First, process paths by recipe to ensure recipe diversity
		for recipeKey, recipePaths := range recipeGroups {
			if len(trees) >= count {
				break
			}

			log.Printf("DEBUG: Processing recipe '%s' with %d paths", recipeKey, len(recipePaths))

			// Sort paths by length for this recipe
			sort.Slice(recipePaths, func(i, j int) bool {
				return len(recipePaths[i]) < len(recipePaths[j])
			})

			// Try each path until we get a valid tree for this recipe
			treeCreated := false
			for _, path := range recipePaths {
				tree := convertPathToTree(path, elementName, h.elements, baseElements)

				if tree != nil {
					ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))
					signature := generateDetailedTreeSignature(tree)

					if !uniqueSignatures[signature] {
						uniqueSignatures[signature] = true
						trees = append(trees, tree)
						log.Printf("DEBUG: Added tree for recipe: %s (tree count: %d)", recipeKey, len(trees))
						treeCreated = true
						break
					}
				}
			}

			if !treeCreated {
				log.Printf("DEBUG: Failed to create tree for recipe: %s, trying manual approach", recipeKey)

				// Create tree manually using the recipe ingredients
				recipe, exists := dbRecipesByKey[recipeKey]
				if !exists {
					// Try to extract from a path
					if len(recipePaths) > 0 {
						for _, node := range recipePaths[0] {
							if node.Element == elementName && node.Ingredients != nil {
								recipe.Ingredients = node.Ingredients
								break
							}
						}
					}
				}

				if len(recipe.Ingredients) >= 2 {
					tree := map[string]interface{}{
						"name":        elementName,
						"imagePath":   element.ImagePath,
						"ingredients": make([]interface{}, 0),
					}

					// Find the best paths for each ingredient
					for _, ingredient := range recipe.Ingredients {
						isIngBase := utils.IsBaseElementName(ingredient, baseElements)
						ingElement, exists := h.elements[ingredient]
						if !exists {
							continue
						}

						ingTree := map[string]interface{}{
							"name":          ingredient,
							"imagePath":     ingElement.ImagePath,
							"isBaseElement": isIngBase,
						}

						if isIngBase {
							ingTree["ingredients"] = []interface{}{}
						} else {
							// Find the best path for this ingredient from our search results
							var bestPath []model.Node
							for _, searchPath := range paths {
								for _, node := range searchPath {
									if node.Element == ingredient {
										subPath := extractSubPath(searchPath, ingredient)
										if len(subPath) > 0 {
											if bestPath == nil || len(subPath) < len(bestPath) {
												bestPath = subPath
											}
										}
										break
									}
								}
							}

							if bestPath != nil {
								ingSubTree := convertPathToTree(bestPath, ingredient, h.elements, baseElements)
								if ingSubTree != nil {
									ingTree = ingSubTree
								} else {
									ingTree["ingredients"] = []interface{}{}
								}
							} else {
								ingTree["ingredients"] = []interface{}{}
							}
						}

						tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
					}

					ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))
					signature := generateDetailedTreeSignature(tree)

					if !uniqueSignatures[signature] {
						uniqueSignatures[signature] = true
						trees = append(trees, tree)
						log.Printf("DEBUG: Added manual tree for recipe: %s (tree count: %d)", recipeKey, len(trees))
					}
				}
			}
		}

		log.Printf("DEBUG: Final tree count: %d", len(trees))

		totalNodeCount := 0
		for _, tree := range trees {
			totalNodeCount += countNodesInTree(tree)
		}

		result := map[string]interface{}{
			"trees":          trees,
			"nodesVisited":   visitedCount,
			"totalTreeNodes": totalNodeCount,
			"timeElapsed":    timeElapsed,
			"algorithm":      algoName,
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Printf("ERROR: Failed to encode response: %v", err)
			return
		}
	} else {
		// Return standard path results
		result := map[string]interface{}{
			"paths":        paths,
			"nodesVisited": visitedCount,
			"timeElapsed":  timeElapsed,
			"algorithm":    algoName,
			"recipeCount":  len(recipeGroups),
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Printf("ERROR: Failed to encode response: %v", err)
			return
		}
	}

	log.Printf("DEBUG: Successfully sent bidirectional search response")
}

// Helper function to extract a subpath leading to a target element
func extractSubPath(path []model.Node, targetElement string) []model.Node {
	for i, node := range path {
		if node.Element == targetElement {
			subPath := make([]model.Node, i+1)
			copy(subPath, path[:i+1])
			return subPath
		}
	}
	return nil
}
