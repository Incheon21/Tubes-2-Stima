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

func generateTreeCombinations(baseTree map[string]interface{}, ingredientVariations [][]map[string]interface{}, currentIndex int) []map[string]interface{} {
	if currentIndex >= len(ingredientVariations) {
		return []map[string]interface{}{utils.DeepCopyTree(baseTree)}
	}

	currentIngredientVariations := ingredientVariations[currentIndex]

	var results []map[string]interface{}
	for _, ingTree := range currentIngredientVariations {
		ingredientsList := baseTree["ingredients"].([]interface{})
		baseTree["ingredients"] = append(ingredientsList, ingTree)

		subCombinations := generateTreeCombinations(
			baseTree,
			ingredientVariations,
			currentIndex+1,
		)

		results = append(results, subCombinations...)

		baseTree["ingredients"] = ingredientsList
	}

	return results
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

	g := utils.CreateElementGraph(h.elements)

	trees, visited := generateAllRecipeTrees(g, elementName, element.ImagePath, count, baseElements)

	if len(trees) == 0 {
		visitCount := 0
		visitedNodes := make(map[string]bool)
		tree := utils.BuildElementTreeDFS(g, elementName, visitedNodes, &visitCount)
		trees = []map[string]interface{}{tree}
		visited = visitCount
		log.Printf("DEBUG: No recipe trees found, added fallback element tree using DFS (nodes visited: %d)", visitCount)
	}

	minCount := count
	if elementName == "Metal" && count < 5 {
		minCount = 5
		log.Printf("DEBUG: Ensuring at least 5 trees for Metal element")
	}

	if len(trees) > minCount {
		trees = trees[:minCount]
		log.Printf("DEBUG: Limited trees to requested count: %d", minCount)
	}

	timeElapsed := time.Since(startTime).Milliseconds()

	result := map[string]interface{}{
		"trees":        trees,
		"nodesVisited": visited,
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

func generateAllRecipeTrees(g *graph.ElementGraph, elementName, imagePath string, maxCount int, baseElements []string) ([]map[string]interface{}, int) {
	totalVisited := 0
	node := g.Nodes[elementName]

	if node == nil || len(node.RecipesToMakeThisElement) == 0 {
		log.Printf("DEBUG: Element '%s' has no recipes", elementName)
		return []map[string]interface{}{}, 0
	}

	var allTrees []map[string]interface{}
	log.Printf("DEBUG: Element '%s' has %d direct recipes", elementName, len(node.RecipesToMakeThisElement))

	explorationLimit := maxCount * 10

	for recipeIdx, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		log.Printf("DEBUG: Processing recipe %d for %s with ingredients: %v",
			recipeIdx, elementName, recipe.Ingredients)

		baseTree := map[string]interface{}{
			"name":          elementName,
			"imagePath":     imagePath,
			"isBaseElement": false,
			"ingredients":   make([]interface{}, 0, len(recipe.Ingredients)),
		}

		ingredientTreeVariations := make([][]map[string]interface{}, len(recipe.Ingredients))
		localVisited := 0

		for i, ingredient := range recipe.Ingredients {
			// Check if ingredient is a base element
			isBase := false
			for _, base := range baseElements {
				if ingredient == base {
					isBase = true
					break
				}
			}

			ingNode := g.Nodes[ingredient]
			if isBase {
				ingredientTreeVariations[i] = []map[string]interface{}{{
					"name":          ingredient,
					"imagePath":     ingNode.ImagePath,
					"isBaseElement": true,
					"ingredients":   []interface{}{},
				}}
				localVisited++
			} else if ingNode == nil || len(ingNode.RecipesToMakeThisElement) == 0 {
				ingredientTreeVariations[i] = []map[string]interface{}{{
					"name":          ingredient,
					"imagePath":     ingNode.ImagePath,
					"isBaseElement": false,
					"ingredients":   []interface{}{},
				}}
				localVisited++
			} else {
				ingredientMaxCount := 10

				log.Printf("DEBUG: Ingredient %s has %d recipes",
					ingredient, len(ingNode.RecipesToMakeThisElement))

				subVisited := 0
				subTrees, subVisited := generateAllRecipeTrees(g, ingredient, ingNode.ImagePath, ingredientMaxCount, baseElements)

				if len(subTrees) == 0 {
					ingredientTreeVariations[i] = []map[string]interface{}{{
						"name":          ingredient,
						"imagePath":     ingNode.ImagePath,
						"isBaseElement": false,
						"ingredients":   []interface{}{},
					}}
					localVisited++
				} else {
					ingredientTreeVariations[i] = subTrees
					log.Printf("DEBUG: Found %d recipe variations for ingredient %s",
						len(subTrees), ingredient)
					localVisited += subVisited
				}
			}
		}

		treeCombinations := generateTreeCombinations(baseTree, ingredientTreeVariations, 0)

		log.Printf("DEBUG: Generated %d tree combinations for recipe %d",
			len(treeCombinations), recipeIdx)

		allTrees = append(allTrees, treeCombinations...)

		totalVisited += localVisited

		if len(allTrees) > explorationLimit {
			log.Printf("DEBUG: Generated %d trees, stopping early", len(allTrees))
			break
		}
	}

	uniqueTrees := improvedDeduplicateTrees(allTrees)

	log.Printf("DEBUG: Generated %d unique trees from %d total combinations",
		len(uniqueTrees), len(allTrees))

	return uniqueTrees, totalVisited
}

func improvedDeduplicateTrees(trees []map[string]interface{}) []map[string]interface{} {
	if len(trees) <= 1 {
		return trees
	}

	uniqueSignatures := make(map[string]bool)
	var uniqueTrees []map[string]interface{}

	for _, tree := range trees {
		signature := generateDetailedTreeSignature(tree)

		if !uniqueSignatures[signature] {
			uniqueSignatures[signature] = true
			uniqueTrees = append(uniqueTrees, tree)
		}
	}

	log.Printf("DEBUG: After improved deduplication: %d unique trees from %d input trees",
		len(uniqueTrees), len(trees))

	return uniqueTrees
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

	// Extract element name from URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bidirectional/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bidirectional/{elementName}", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")

	log.Printf("DEBUG: Bidirectional search request for element: %s", elementName)

	// Parse query parameters
	count := 3 // Default number of results
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

	// Check if tree visualization is requested
	treeView := false
	if treeParam := r.URL.Query().Get("tree"); treeParam == "true" {
		treeView = true
		log.Printf("DEBUG: Tree visualization requested")
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

		// Adapt response based on whether tree view is requested
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

	// Run the search with a higher count to ensure diversity
	log.Printf("DEBUG: Starting bidirectional search for element '%s'", elementName)
	startTime := time.Now()

	// Request MANY more paths to ensure we get diverse recipes
	searchCount := count * 10

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

	log.Printf("DEBUG: %s search found %d paths after visiting %d nodes in %d ms",
		algoName, len(paths), visitedCount, timeElapsed)

	if treeView {
		log.Printf("DEBUG: Processing %d paths to create tree visualizations", len(paths))

		trees := make([]map[string]interface{}, 0)
		uniqueSignatures := make(map[string]bool)

		// Get element recipes directly from the database
		element := h.elements[elementName]
		recipeList := element.Recipes

		// Log the available recipes
		log.Printf("DEBUG: Element '%s' has %d recipes in database", elementName, len(recipeList))

		// First approach: Generate trees directly from recipes in the database
		if len(recipeList) > 0 && len(trees) < count {
			log.Printf("DEBUG: Generating trees from database recipes first")

			// Get graph for recipe generation
			g := utils.CreateElementGraph(h.elements)

			// Create one tree per recipe
			for _, recipe := range recipeList {
				if len(trees) >= count {
					break
				}

				// Skip recipes without enough ingredients
				if len(recipe.Ingredients) < 2 {
					continue
				}

				// Build tree for this specific recipe
				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": make([]interface{}, 0),
				}

				// Add ingredient trees
				for _, ingredient := range recipe.Ingredients {
					// Check if ingredient is a base element
					isIngBase := false
					for _, base := range baseElements {
						if ingredient == base {
							isIngBase = true
							break
						}
					}

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
						// Generate a small tree for each non-base ingredient
						visitCount := 0
						visitedNodes := make(map[string]bool)
						subTree := utils.BuildElementTreeDFS(g, ingredient, visitedNodes, &visitCount)
						ingTree = subTree
					}

					tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
				}

				// Ensure all ingredients are expanded properly
				ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

				// Add tree if unique
				signature := generateDetailedTreeSignature(tree)
				if !uniqueSignatures[signature] {
					uniqueSignatures[signature] = true
					trees = append(trees, tree)
					log.Printf("DEBUG: Added tree from database recipe (total: %d)", len(trees))
				}
			}
		}

		// Second approach: Process search paths
		if len(trees) < count {
			log.Printf("DEBUG: Still need more trees, processing search paths")

			// Group paths by recipe
			recipeGroups := make(map[string][][]model.Node)

			for _, path := range paths {
				if len(path) == 0 {
					continue
				}

				// Find the target node in the path
				var targetNode *model.Node
				for i := range path {
					if path[i].Element == elementName {
						targetNode = &path[i]
						break
					}
				}

				// Skip invalid paths
				if targetNode == nil || targetNode.Ingredients == nil || len(targetNode.Ingredients) < 2 {
					continue
				}

				// Create recipe key
				sortedIngredients := make([]string, len(targetNode.Ingredients))
				copy(sortedIngredients, targetNode.Ingredients)
				sort.Strings(sortedIngredients)
				recipeKey := strings.Join(sortedIngredients, "+")

				// Add to recipe group
				recipeGroups[recipeKey] = append(recipeGroups[recipeKey], path)
			}

			log.Printf("DEBUG: Found %d unique recipes from paths", len(recipeGroups))

			// Process recipes in sorted order for consistency
			recipeKeys := make([]string, 0, len(recipeGroups))
			for key := range recipeGroups {
				recipeKeys = append(recipeKeys, key)
			}
			sort.Strings(recipeKeys)

			// Create one tree per recipe
			for _, recipeKey := range recipeKeys {
				if len(trees) >= count {
					break
				}

				recipePaths := recipeGroups[recipeKey]

				// Sort paths by length
				sort.Slice(recipePaths, func(i, j int) bool {
					return len(recipePaths[i]) < len(recipePaths[j])
				})

				// Try each path until we get a valid tree
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
							break // Move to next recipe
						}
					}
				}

				// Log if we couldn't create a tree for this recipe
				if !treeCreated {
					log.Printf("DEBUG: Failed to create tree for recipe: %s", recipeKey)
				}
			}
		}

		// Third approach: Fallback to graph-based generation if needed
		if len(trees) < count {
			log.Printf("DEBUG: Still need %d more trees, using fallback generation", count-len(trees))

			g := utils.CreateElementGraph(h.elements)
			additionalTrees, _ := generateAllRecipeTrees(g, elementName, element.ImagePath, count, baseElements)

			// Add these trees with duplicate checking
			for _, tree := range additionalTrees {
				if len(trees) >= count {
					break
				}

				signature := generateDetailedTreeSignature(tree)

				if !uniqueSignatures[signature] {
					uniqueSignatures[signature] = true
					trees = append(trees, tree)
					log.Printf("DEBUG: Added fallback tree (total: %d)", len(trees))
				}
			}
		}

		log.Printf("DEBUG: Final tree count: %d", len(trees))

		// Calculate total node count
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
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Printf("ERROR: Failed to encode response: %v", err)
			return
		}
	}

	log.Printf("DEBUG: Successfully sent bidirectional search response")
}
