package api

import (
	alg "backend/internal/algorithm"
	"backend/internal/graph"
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

	// First, collect only fully composable paths
	fullyComposablePaths := make([][]model.Node, 0)
	for _, path := range paths {
		if alg.IsFullyComposablePath(path, baseElements, graph.NewElementGraph(h.elements)) {
			fullyComposablePaths = append(fullyComposablePaths, path)
		}
	}

	// If we have fully composable paths, only use those
	pathsToProcess := fullyComposablePaths
	if len(pathsToProcess) == 0 {
		// Fall back to all paths if no fully composable paths found
		pathsToProcess = paths
	}

	for _, path := range pathsToProcess {
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

	// Add the code here to filter for makeable trees
	makeableTrees := make([]map[string]interface{}, 0)
	for _, tree := range trees {
		if isTreeFullyMakeable(tree) {
			makeableTrees = append(makeableTrees, tree)
		}
	}

	if len(makeableTrees) == 0 {
		log.Printf("DEBUG: No makeable trees found, trying again with single path mode")

		// Try with single path mode to get a fully composable path
		var singlePath [][]model.Node
		if useMultithreaded {
			singlePath, visitedCount = alg.MultiThreadedBFS(h.elements, elementName, 1, true)
		} else {
			singlePath, visitedCount = alg.BFS(h.elements, elementName, 1, true)
		}

		if len(singlePath) > 0 {
			log.Printf("DEBUG: Got a single path with single path mode, converting to tree")
			tree := convertPathToTree(singlePath[0], elementName, h.elements, baseElements)
			if tree != nil {
				ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))
				if isTreeFullyMakeable(tree) {
					makeableTrees = append(makeableTrees, tree)
					log.Printf("DEBUG: Successfully found a makeable tree with single path mode")
				}
			}
		}

		// If we still have no makeable trees, we need to try alternative recipes
		if len(makeableTrees) == 0 {
			log.Printf("DEBUG: Still no makeable trees, trying alternative recipes")

			// Try creating a tree directly from recipes
			element := h.elements[elementName]
			for _, recipe := range element.Recipes {
				if len(recipe.Ingredients) == 0 {
					continue
				}

				// Check if this recipe only uses base elements or fully traceable elements
				allIngredientsTraceable := true
				for _, ingName := range recipe.Ingredients {
					isBase := false
					for _, base := range baseElements {
						if ingName == base {
							isBase = true
							break
						}
					}

					if !isBase {
						ingElement, exists := h.elements[ingName]
						if !exists || len(ingElement.Recipes) == 0 ||
							!alg.IsElementTraceable(ingName, baseElements, graph.NewElementGraph(h.elements)) {
							allIngredientsTraceable = false
							break
						}
					}
				}

				if allIngredientsTraceable {
					// Create a tree with this recipe
					tree := map[string]interface{}{
						"name":        elementName,
						"imagePath":   element.ImagePath,
						"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
					}

					for _, ingName := range recipe.Ingredients {
						ingElement, exists := h.elements[ingName]
						if !exists {
							continue
						}

						isBase := false
						for _, base := range baseElements {
							if ingName == base {
								isBase = true
								break
							}
						}

						ingTree := map[string]interface{}{
							"name":          ingName,
							"imagePath":     ingElement.ImagePath,
							"isBaseElement": isBase,
							"ingredients":   []interface{}{},
						}

						tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
					}

					ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))
					if isTreeFullyMakeable(tree) {
						makeableTrees = append(makeableTrees, tree)
						log.Printf("DEBUG: Found makeable tree from direct recipe")
						break
					}
				}
			}
		}

		// If we still have no makeable trees, we need to create a special case
		if len(makeableTrees) == 0 {
			log.Printf("DEBUG: Unable to find any makeable trees for %s", elementName)
			// Create a tree with special "unmakeable" notice
			trees = []map[string]interface{}{{
				"name":        elementName,
				"imagePath":   h.elements[elementName].ImagePath,
				"unmakeable":  false, // Don't mark the top element as unmakeable
				"ingredients": []interface{}{},
				"notice":      "This element cannot be fully traced to base elements",
			}}
		} else {
			trees = makeableTrees
		}
	} else {
		trees = makeableTrees
		log.Printf("DEBUG: Using %d makeable trees", len(trees))
	}

	// Continue with the existing code
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

func ensureIngredientsExpanded(tree map[string]interface{}, elements map[string]model.Element, baseElements []string, visited map[string]bool) bool {
	if tree == nil {
		return false
	}

	elementName, ok := tree["name"].(string)
	if !ok || visited[elementName] {
		return false
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
		return true
	}

	// Check if this element is makeable
	elemData, exists := elements[elementName]
	if !exists || len(elemData.Recipes) == 0 {
		// Mark unmakeable elements appropriately
		tree["unmakeable"] = true
		return false
	}

	ingredients, ok := tree["ingredients"].([]interface{})
	allIngredientsValid := true

	if !ok || len(ingredients) == 0 {
		// No ingredients provided, try to get from element data
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
				allIngredientsValid = false
				continue
			}

			ingTree := map[string]interface{}{
				"name":          ingName,
				"imagePath":     ingData.ImagePath,
				"isBaseElement": ingIsBase,
				"ingredients":   []interface{}{},
			}

			if !ingIsBase {
				ingValid := ensureIngredientsExpanded(ingTree, elements, baseElements, visited)
				if !ingValid {
					allIngredientsValid = false
				}
			}

			newIngredients = append(newIngredients, ingTree)
		}

		tree["ingredients"] = newIngredients
	} else {
		// Expand existing ingredients
		for _, ing := range ingredients {
			if ingTree, ok := ing.(map[string]interface{}); ok {
				ingValid := ensureIngredientsExpanded(ingTree, elements, baseElements, visited)
				if !ingValid {
					allIngredientsValid = false
				}
			}
		}
	}

	return allIngredientsValid
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
	positionNodeMap := make(map[string]map[int]*model.Node) // Track nodes by position

	for i := range path {
		nodeMap[path[i].Element] = &path[i]

		// Also index by position if available
		if path[i].Position != 0 {
			elemKey := path[i].Element
			if positionNodeMap[elemKey] == nil {
				positionNodeMap[elemKey] = make(map[int]*model.Node)
			}
			positionNodeMap[elemKey][path[i].Position] = &path[i]
		}
	}

	processedInBranch := make(map[string]bool)
	// Create element validity cache
	validityCache := make(map[string]bool)

	// Check if an element is makeable (has valid recipes)
	isElementMakeable := func(element string) bool {
		// Base elements are always valid
		for _, base := range baseElements {
			if element == base {
				return true
			}
		}

		// Check cache
		if result, ok := validityCache[element]; ok {
			return result
		}

		// Check if this element has valid recipes
		elemData, exists := elements[element]
		if !exists {
			validityCache[element] = false
			return false
		}

		if len(elemData.Recipes) == 0 {
			validityCache[element] = false
			return false
		}

		// Element has recipes, considered valid for now
		validityCache[element] = true
		return true
	}

	var buildTree func(element string, depth int) map[string]interface{}
	buildTree = func(element string, depth int) map[string]interface{} {
		if processedInBranch[element] {
			return map[string]interface{}{
				"name":                element,
				"isCircularReference": true,
				"ingredients":         []interface{}{},
			}
		}

		// Check if we have position-specific nodes for this element
		if posMap, exists := positionNodeMap[element]; exists && len(posMap) > 0 {
			// Log that we're using position-specific recipe variant
			log.Printf("DEBUG: Using position-specific recipe for %s", element)
		}

		// Check if this element is makeable
		if !isElementMakeable(element) && !processedInBranch[element] {
			elemData, exists := elements[element]
			if exists {
				return map[string]interface{}{
					"name":        element,
					"imagePath":   elemData.ImagePath,
					"unmakeable":  true,
					"ingredients": []interface{}{},
				}
			}
			return nil
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
		if len(ingredients) == 0 {
			// Try to get from element data
			if elemData, exists := elements[element]; exists && len(elemData.Recipes) > 0 {
				ingredients = elemData.Recipes[0].Ingredients
			}
		}

		if depth < 10 {
			for _, ingredient := range ingredients {
				subtree := buildTree(ingredient, depth+1)
				if subtree != nil {
					// Add path index information for duplicate elements
					for i := 0; i < len(ingredients); i++ {
						if ingredients[i] == ingredient && i > 0 {
							// Add a path index marker for elements that appear multiple times
							subtree["pathIndex"] = i
							subtree["ingredientIndex"] = i
							break
						}
					}
					treeNode["ingredients"] = append(treeNode["ingredients"].([]interface{}), subtree)
				}
			}
		}

		return treeNode
	}

	return buildTree(targetElement, 0)
}

func isTreeFullyMakeable(tree map[string]interface{}) bool {
	// Check if this node is marked as unmakeable
	if unmakeable, ok := tree["unmakeable"].(bool); ok && unmakeable {
		elementName, _ := tree["name"].(string)
		log.Printf("DEBUG: Tree node %s is unmakeable, rejecting tree", elementName)
		return false
	}

	// Check if this is a non-base element with empty ingredients
	elementName, _ := tree["name"].(string)
	ingredients, hasIngredients := tree["ingredients"].([]interface{})
	isBase, hasBase := tree["isBaseElement"].(bool)

	// If it's not a base element and has no ingredients or empty ingredients list, it's unmakeable
	if (!hasBase || !isBase) && (!hasIngredients || len(ingredients) == 0) {
		log.Printf("DEBUG: Non-base element %s has no ingredients, marking as unmakeable", elementName)
		tree["unmakeable"] = true
		return false
	}

	// Check all ingredients
	if hasIngredients {
		for _, ing := range ingredients {
			ingredient, ok := ing.(map[string]interface{})
			if !ok {
				continue
			}

			if !isTreeFullyMakeable(ingredient) {
				// If any ingredient is unmakeable, the whole tree is unmakeable
				log.Printf("DEBUG: Tree node %s has unmakeable ingredient, rejecting tree", elementName)
				return false
			}
		}
	}

	return true
}

func (h *Handler) HandleDFSTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/dfs-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/dfs-tree/{elementName}?count=N", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")

	// Get count parameter or set to unlimited (-1) if "all" is specified
	count := -1 // Default to unlimited
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if countParam == "all" {
			count = -1 // Explicitly set to unlimited
		} else if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
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

	countStr := "unlimited"
	if count != -1 {
		countStr = strconv.Itoa(count)
	}
	log.Printf("DEBUG: Starting DFS tree search for element '%s' (requesting %s trees)",
		elementName, countStr)
	startTime := time.Now()

	// Use a higher search path count for diversity
	// If unlimited count requested, use a very high number but not unlimited to avoid exhausting resources
	searchPathCount := 1000 // Set very high to ensure all recipes are found
	if count > 0 {
		searchPathCount = count * 20 // Increased multiplier for better coverage
	}

	paths, visitedCount := alg.MultiThreadedDFS(h.elements, elementName, searchPathCount, false)

	log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visitedCount)

	// Convert paths to trees
	trees := make([]map[string]interface{}, 0, len(paths))
	uniqueSignatures := make(map[string]bool)
	recipeSignatures := make(map[string]bool)

	// First pass: Convert paths to trees and track unique recipes
	for _, path := range paths {
		// Convert path to tree
		tree := convertPathToTree(path, elementName, h.elements, baseElements)

		// Ensure all ingredients are fully expanded
		if tree != nil {
			ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

			// Get recipe signature (just top-level ingredients)
			recipeSig := getTopLevelRecipeSignature(tree)

			// Deduplicate by detailed signature
			signature := generateDetailedTreeSignature(tree)
			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				recipeSignatures[recipeSig] = true
				trees = append(trees, tree)
			}
		}
	}

	log.Printf("DEBUG: Generated %d unique trees from %d paths", len(trees), len(paths))

	// Generate additional trees from direct DB recipes
	// If unlimited, get all recipes; otherwise check if we have enough
	shouldGenerateMore := count == -1 ||
		(len(trees) < count || len(recipeSignatures) < min(count, len(element.Recipes)))

	if shouldGenerateMore {
		log.Printf("DEBUG: Generating more trees from element recipes")

		// Create trees directly from element recipes
		for recipeIdx, recipe := range element.Recipes {
			if count != -1 && (len(trees) >= count*2 || len(recipeSignatures) >= count*2) {
				break
			}

			// Skip recipes with no ingredients
			if len(recipe.Ingredients) == 0 {
				continue
			}

			// Check if all ingredients in this recipe are traceable before creating a tree
			allIngredientsTraceable := true
			for _, ingredient := range recipe.Ingredients {
				// Skip checking base elements
				isBaseElement := false
				for _, base := range baseElements {
					if ingredient == base {
						isBaseElement = true
						break
					}
				}

				// If not base element, check if it's traceable
				if !isBaseElement && !alg.IsElementTraceable(ingredient, baseElements, graph.NewElementGraph(h.elements)) {
					log.Printf("DEBUG: Recipe ingredient '%s' is not traceable to base elements in direct tree generation", ingredient)
					allIngredientsTraceable = false
					break
				}
			}

			if !allIngredientsTraceable {
				log.Printf("DEBUG: Skipping untraceable recipe in direct tree generation: %v", recipe.Ingredients)
				continue
			}

			// Create base tree for this recipe
			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   element.ImagePath,
				"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
			}

			// Build ingredients for this recipe
			for _, ingredient := range recipe.Ingredients {
				isBase := false
				for _, base := range baseElements {
					if ingredient == base {
						isBase = true
						break
					}
				}

				ingredientData, exists := h.elements[ingredient]
				if !exists {
					continue
				}

				// Create ingredient subtree
				ingTree := map[string]interface{}{
					"name":          ingredient,
					"imagePath":     ingredientData.ImagePath,
					"isBaseElement": isBase,
					"ingredients":   []interface{}{},
				}

				// For non-base ingredients, try to find an existing path or expand recipes
				if !isBase {
					// Try existing paths first
					found := false
					for _, path := range paths {
						subPath := extractSubPath(path, ingredient)
						if subPath != nil {
							subTree := convertPathToTree(subPath, ingredient, h.elements, baseElements)
							if subTree != nil {
								ingTree = subTree
								found = true
								break
							}
						}
					}

					// If no path found, just expand the ingredient by selecting a recipe
					if !found && len(ingredientData.Recipes) > 0 {
						// Pick a different recipe for variety
						recipeIndex := (recipeIdx + len(tree["ingredients"].([]interface{}))) % len(ingredientData.Recipes)
						ingRecipe := ingredientData.Recipes[recipeIndex]

						if len(ingRecipe.Ingredients) > 0 {
							// Create subtrees for this recipe's ingredients
							for _, subIngName := range ingRecipe.Ingredients {
								isSubIngBase := false
								for _, base := range baseElements {
									if subIngName == base {
										isSubIngBase = true
										break
									}
								}

								subIngData, exists := h.elements[subIngName]
								if !exists {
									continue
								}

								subIngTree := map[string]interface{}{
									"name":          subIngName,
									"imagePath":     subIngData.ImagePath,
									"isBaseElement": isSubIngBase,
									"ingredients":   []interface{}{},
								}

								// Add as ingredient
								if ingTree["ingredients"] == nil {
									ingTree["ingredients"] = []interface{}{}
								}
								ingTree["ingredients"] = append(ingTree["ingredients"].([]interface{}), subIngTree)
							}
						}
					}
				}

				// Add to tree ingredients
				tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
			}

			// Ensure all ingredients are fully expanded
			ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

			// Add tree if unique
			recipeSig := getTopLevelRecipeSignature(tree)
			signature := generateDetailedTreeSignature(tree)

			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				recipeSignatures[recipeSig] = true
				trees = append(trees, tree)
				log.Printf("DEBUG: Added direct recipe tree for recipe %d", recipeIdx)
			}
		}

		// If we still need more trees and not unlimited, use randomness for variety
		shouldAddRandomVariations := count == -1 || len(trees) < count
		if shouldAddRandomVariations {
			log.Printf("DEBUG: Adding more tree variations with randomness")

			// Determine max variations to try
			maxVariations := 50
			if count > 0 {
				maxVariations = count * 2
			}

			// Use ensureIngredientsRandomlyExpanded for more variety
			for i := 0; i < maxVariations; i++ {
				// Stop if we have enough trees when count is limited
				if count > 0 && len(trees) >= count {
					break
				}

				// Start with any recipe
				if len(element.Recipes) == 0 {
					continue
				}

				recipeIdx := i % len(element.Recipes)
				recipe := element.Recipes[recipeIdx]

				if len(recipe.Ingredients) == 0 {
					continue
				}

				// Check if all ingredients in this recipe are traceable before creating a tree
				allIngredientsTraceable := true
				for _, ingredient := range recipe.Ingredients {
					// Skip checking base elements
					isBaseElement := false
					for _, base := range baseElements {
						if ingredient == base {
							isBaseElement = true
							break
						}
					}

					// If not base element, check if it's traceable
					if !isBaseElement && !alg.IsElementTraceable(ingredient, baseElements, graph.NewElementGraph(h.elements)) {
						log.Printf("DEBUG: Recipe ingredient '%s' is not traceable in variation generation", ingredient)
						allIngredientsTraceable = false
						break
					}
				}

				if !allIngredientsTraceable {
					log.Printf("DEBUG: Skipping untraceable recipe in variation generation: %v", recipe.Ingredients)
					continue
				}

				// Create base tree
				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
				}

				// Add ingredients
				for _, ingredient := range recipe.Ingredients {
					isBase := false
					for _, base := range baseElements {
						if ingredient == base {
							isBase = true
							break
						}
					}

					ingData, exists := h.elements[ingredient]
					if !exists {
						continue
					}

					ingTree := map[string]interface{}{
						"name":          ingredient,
						"imagePath":     ingData.ImagePath,
						"isBaseElement": isBase,
						"ingredients":   []interface{}{},
					}

					tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
				}

				// Use random expansion
				visited := make(map[string]bool)
				ensureIngredientsRandomlyExpanded(tree, h.elements, baseElements, visited, i)

				// Add if unique
				signature := generateDetailedTreeSignature(tree)
				recipeSig := getTopLevelRecipeSignature(tree)

				if !uniqueSignatures[signature] {
					uniqueSignatures[signature] = true
					recipeSignatures[recipeSig] = true
					trees = append(trees, tree)
				}
			}
		}
	}

	// If we still have no trees, use a fallback approach
	if len(trees) == 0 {
		log.Printf("DEBUG: No trees generated at all, using fallback DFS tree builder")
		g := utils.CreateElementGraph(h.elements)
		visitCount := 0
		visitedNodes := make(map[string]bool)
		tree := utils.BuildElementTreeDFS(g, elementName, visitedNodes, &visitCount)
		trees = []map[string]interface{}{tree}
		visitedCount += visitCount
	}

	// Sort trees by complexity (base elements count)
	sort.Slice(trees, func(i, j int) bool {
		return getTreeComplexityScore(trees[i]) < getTreeComplexityScore(trees[j])
	})

	// If count is specified and we have more trees than requested, apply recipe diversity selection
	if count > 0 && len(trees) > count {
		selectedTrees := make([]map[string]interface{}, 0, count)
		selectedRecipes := make(map[string]bool)

		// First, select trees with different top-level recipe signatures
		for _, tree := range trees {
			if len(selectedTrees) >= count {
				break
			}

			recipeSig := getTopLevelRecipeSignature(tree)
			if !selectedRecipes[recipeSig] {
				selectedRecipes[recipeSig] = true
				selectedTrees = append(selectedTrees, tree)
			}
		}

		// If we still need more trees, add the most diverse remaining trees
		if len(selectedTrees) < count {
			// Create a map of existing tree signatures
			existingSigs := make(map[string]bool)
			for _, tree := range selectedTrees {
				sig := generateDetailedTreeSignature(tree)
				existingSigs[sig] = true
			}

			// Add more trees prioritizing diversity
			for _, tree := range trees {
				if len(selectedTrees) >= count {
					break
				}

				sig := generateDetailedTreeSignature(tree)
				if !existingSigs[sig] {
					existingSigs[sig] = true
					selectedTrees = append(selectedTrees, tree)
				}
			}
		}

		trees = selectedTrees
	}

	traceableTrees := make([]map[string]interface{}, 0)
	for _, tree := range trees {
		if isTreeFullyTraceable(tree, baseElements, h.elements) {
			traceableTrees = append(traceableTrees, tree)
		} else {
			log.Printf("DEBUG: Filtering out untraceable tree in final check")
		}
	}
	trees = traceableTrees

	// Calculate total node count
	totalNodeCount := 0
	for _, tree := range trees {
		totalNodeCount += countNodesInTree(tree)
	}

	timeElapsed := time.Since(startTime).Milliseconds()

	result := map[string]interface{}{
		"trees":          trees,
		"nodesVisited":   visitedCount,
		"totalTreeNodes": totalNodeCount,
		"timeElapsed":    timeElapsed,
		"algorithm":      "dfs",
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent DFS tree response with %d trees in %d ms",
		len(trees), timeElapsed)
}

// Helper function for min of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Add this function after the isTreeFullyMakeable function

func isTreeFullyTraceable(tree map[string]interface{}, baseElements []string, elements map[string]model.Element) bool {
	// Check if this node is marked as unmakeable
	if unmakeable, ok := tree["unmakeable"].(bool); ok && unmakeable {
		return false
	}

	// Get element name
	elementName, ok := tree["name"].(string)
	if !ok {
		return false
	}

	// Base elements are always traceable
	for _, base := range baseElements {
		if elementName == base {
			return true
		}
	}

	// Check if the element is explicitly marked untraceable
	if elementName == "Tree" {
		return false // Tree is explicitly not traceable to base elements
	}

	// Check traceability
	g := graph.NewElementGraph(elements)
	if !alg.IsElementTraceable(elementName, baseElements, g) {
		return false
	}

	// Check all ingredients
	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok {
		// Non-base element without ingredients is not traceable
		return false
	}

	for _, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}

		if !isTreeFullyTraceable(ingredient, baseElements, elements) {
			return false
		}
	}

	return true
}

func getTopLevelRecipeSignature(tree map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString(tree["name"].(string))
	sb.WriteString(":[")

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return sb.String() + "]"
	}

	ingNames := make([]string, 0, len(ingredients))
	for _, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}
		ingName, ok := ingredient["name"].(string)
		if ok {
			ingNames = append(ingNames, ingName)
		}
	}

	sort.Strings(ingNames)
	sb.WriteString(strings.Join(ingNames, ","))
	sb.WriteString("]")

	return sb.String()
}

// Helper function to score tree complexity for sorting
// Lower score = simpler tree (more base elements, fewer steps)
func getTreeComplexityScore(tree map[string]interface{}) int {
	if tree == nil {
		return 0
	}

	// Base elements have low complexity
	if isBase, ok := tree["isBaseElement"].(bool); ok && isBase {
		return 0
	}

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return 1 // Just a node with no ingredients
	}

	// Count base elements in direct ingredients
	baseCount := 0
	nonBaseCount := 0
	ingredientComplexity := 0

	for _, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}

		if isBase, ok := ingredient["isBaseElement"].(bool); ok && isBase {
			baseCount++
		} else {
			nonBaseCount++
			ingredientComplexity += getTreeComplexityScore(ingredient)
		}
	}

	// Trees with more base elements directly are simpler
	// Trees with more non-base elements are more complex
	return (nonBaseCount * 10) - baseCount + ingredientComplexity
}

// Update the tree signature generation to capture more structure

func generateDetailedTreeSignature(tree map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(tree["name"].(string))

	// Include position if available
	if pos, ok := tree["position"].(int); ok && pos > 0 {
		sb.WriteString(fmt.Sprintf("#%d", pos))
	}

	sb.WriteString(":")

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return sb.String() + "[]"
	}

	// For combinations like Planet(Continent+Continent), preserve the original order
	// and include more details about each ingredient's path
	elementName, _ := tree["name"].(string)
	preserveOrder := (elementName == "Planet" || elementName == "Continent") ||
		(len(ingredients) >= 2 && ingredients[0].(map[string]interface{})["name"] == ingredients[1].(map[string]interface{})["name"])

	// Generate detailed signatures including positions and path details
	ingredientSignatures := make([]string, 0, len(ingredients))

	for i, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}

		// Include position and index in signature
		ingredientSig := generateDetailedTreeSignature(ingredient)

		// Always include position index for duplicated elements to preserve uniqueness
		if i > 0 && ingredient["name"] == ingredients[i-1].(map[string]interface{})["name"] {
			ingredientSig = fmt.Sprintf("%d:%s", i, ingredientSig)
		} else if preserveOrder {
			// For elements where order matters, include the position always
			ingredientSig = fmt.Sprintf("%d:%s", i, ingredientSig)
		}

		ingredientSignatures = append(ingredientSignatures, ingredientSig)
	}

	// Only sort if they're not position-sensitive and don't contain duplicated elements
	if !preserveOrder && len(ingredientSignatures) > 1 {
		sort.Strings(ingredientSignatures)
	}

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

	element, exists := h.elements[elementName]
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

	// Increase search count to ensure we get enough diversity
	searchCount := count * 20

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

	// First, try to extract all recipes directly from the element data
	// This ensures we consider all possible recipes from the database
	elementRecipes := element.Recipes
	for _, recipe := range elementRecipes {
		if len(recipe.Ingredients) >= 2 {
			sortedIngs := make([]string, len(recipe.Ingredients))
			copy(sortedIngs, recipe.Ingredients)
			sort.Strings(sortedIngs)
			recipeKey := strings.Join(sortedIngs, "+")

			// Initialize with empty path list if this recipe doesn't have paths yet
			if _, exists := recipeGroups[recipeKey]; !exists {
				recipeGroups[recipeKey] = [][]model.Node{}
			}
		}
	}

	// Then add paths to their respective recipe groups
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
		log.Printf("DEBUG: Processing %d recipes to create tree visualizations", len(recipeGroups))

		trees := make([]map[string]interface{}, 0)
		uniqueSignatures := make(map[string]bool)

		// Group recipes by their keys for better matching
		dbRecipesByKey := make(map[string]model.ElementRecipe)
		for _, recipe := range element.Recipes {
			if len(recipe.Ingredients) >= 2 {
				sortedIngs := make([]string, len(recipe.Ingredients))
				copy(sortedIngs, recipe.Ingredients)
				sort.Strings(sortedIngs)
				recipeKey := strings.Join(sortedIngs, "+")
				dbRecipesByKey[recipeKey] = recipe
			}
		}

		// Process each recipe group to create trees
		// Sort recipe keys to ensure consistent processing order
		recipeKeys := make([]string, 0, len(recipeGroups))
		for key := range recipeGroups {
			recipeKeys = append(recipeKeys, key)
		}

		// Sort by recipe complexity (number of ingredients)
		sort.Slice(recipeKeys, func(i, j int) bool {
			return len(strings.Split(recipeKeys[i], "+")) < len(strings.Split(recipeKeys[j], "+"))
		})

		for _, recipeKey := range recipeKeys {
			recipePaths := recipeGroups[recipeKey]

			log.Printf("DEBUG: Processing recipe '%s' with %d paths", recipeKey, len(recipePaths))

			// Try to create at least one tree for each recipe
			treeCreated := false

			// First attempt: use paths if available
			if len(recipePaths) > 0 {
				// Sort paths by length (shorter paths first for simpler trees)
				sort.Slice(recipePaths, func(i, j int) bool {
					return len(recipePaths[i]) < len(recipePaths[j])
				})

				// Try each path until we get a valid tree for this recipe
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
			}

			// Second attempt: manually create tree from recipe ingredients
			if !treeCreated {
				log.Printf("DEBUG: Creating manual tree for recipe: %s", recipeKey)

				// Get recipe ingredients from database or from path
				recipe, exists := dbRecipesByKey[recipeKey]
				if !exists && len(recipePaths) > 0 {
					// Extract ingredients from the first path
					for _, node := range recipePaths[0] {
						if node.Element == elementName && node.Ingredients != nil && len(node.Ingredients) > 0 {
							recipe.Ingredients = node.Ingredients
							break
						}
					}
				}

				// Check if all ingredients in this recipe are traceable before creating the tree
				if len(recipe.Ingredients) > 0 && alg.IsRecipeTraceable(recipe.Ingredients, baseElements, graph.NewElementGraph(h.elements)) {
					tree := map[string]interface{}{
						"name":        elementName,
						"imagePath":   element.ImagePath,
						"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
					}

					// Add ingredient subtrees
					for _, ingredient := range recipe.Ingredients {
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
							"ingredients":   []interface{}{},
						}

						// For non-base ingredients, try to find a path or expand them
						if !isIngBase {
							// Try to find a path for this ingredient
							var ingPath []model.Node
							for _, path := range paths {
								subPath := extractSubPath(path, ingredient)
								if subPath != nil {
									ingPath = subPath
									break
								}
							}

							if ingPath != nil {
								ingSubTree := convertPathToTree(ingPath, ingredient, h.elements, baseElements)
								if ingSubTree != nil {
									ingTree = ingSubTree
								}
							}
						}

						tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
					}

					// Make sure all ingredients are fully expanded
					ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

					// Add to trees if unique
					signature := generateDetailedTreeSignature(tree)
					if !uniqueSignatures[signature] {
						uniqueSignatures[signature] = true
						trees = append(trees, tree)
						log.Printf("DEBUG: Added manual tree for recipe: %s (tree count: %d)", recipeKey, len(trees))
						treeCreated = true
					}
				} else {
					log.Printf("DEBUG: Skipping untraceable recipe: %s", recipeKey)
				}
			}

			// If we have enough trees, stop adding more
			if len(trees) >= count {
				log.Printf("DEBUG: Reached requested tree count (%d), stopping tree generation", count)
				break
			}
		}

		if len(trees) < count {
			log.Printf("DEBUG: Only generated %d/%d trees, trying to create more variations", len(trees), count)

			// Try using direct recipes from the element data
			for _, recipe := range element.Recipes {
				if len(trees) >= count {
					break
				}

				// Skip empty recipes
				if len(recipe.Ingredients) == 0 {
					continue
				}

				// Check if this recipe is traceable before creating a tree for it
				if !alg.IsRecipeTraceable(recipe.Ingredients, baseElements, graph.NewElementGraph(h.elements)) {
					log.Printf("DEBUG: Skipping untraceable recipe for variation: %v", recipe.Ingredients)
					continue
				}

				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
				}

				// Create ingredient subtrees
				for _, ingredient := range recipe.Ingredients {
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
						"ingredients":   []interface{}{},
					}

					tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
				}

				// Ensure ingredients are expanded, using a random approach for variety
				visited := make(map[string]bool)
				ensureIngredientsRandomlyExpanded(tree, h.elements, baseElements, visited, len(trees))

				// Check if this tree is unique
				signature := generateDetailedTreeSignature(tree)
				if !uniqueSignatures[signature] {
					uniqueSignatures[signature] = true
					trees = append(trees, tree)
					log.Printf("DEBUG: Added variation tree (count: %d)", len(trees))
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

// Helper function to expand ingredients with some randomness for variety
func ensureIngredientsRandomlyExpanded(tree map[string]interface{}, elements map[string]model.Element, baseElements []string, visited map[string]bool, seed int) {
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

	// Check if this element is traceable before expanding
	if !alg.IsElementTraceable(elementName, baseElements, graph.NewElementGraph(elements)) {
		tree["unmakeable"] = true
		return
	}

	ingredients, ok := tree["ingredients"].([]interface{})

	if (!ok || len(ingredients) == 0) && !isBase {
		elemData, exists := elements[elementName]
		if exists && len(elemData.Recipes) > 0 {
			// Find a traceable recipe
			var validRecipe model.ElementRecipe
			foundValidRecipe := false

			// Try up to 3 random recipes
			for attempt := 0; attempt < 3 && !foundValidRecipe; attempt++ {
				recipeIdx := (seed + attempt) % len(elemData.Recipes)
				recipe := elemData.Recipes[recipeIdx]

				if alg.IsRecipeTraceable(recipe.Ingredients, baseElements, graph.NewElementGraph(elements)) {
					validRecipe = recipe
					foundValidRecipe = true
				}
			}

			if !foundValidRecipe {
				tree["unmakeable"] = true
				return
			}

			newIngredients := make([]interface{}, 0, len(validRecipe.Ingredients))

			for _, ingName := range validRecipe.Ingredients {
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
					// Only expand if the ingredient is traceable
					if alg.IsElementTraceable(ingName, baseElements, graph.NewElementGraph(elements)) {
						ensureIngredientsRandomlyExpanded(ingTree, elements, baseElements, visited, seed+1)
					} else {
						ingTree["unmakeable"] = true
					}
				}

				newIngredients = append(newIngredients, ingTree)
			}

			tree["ingredients"] = newIngredients
		}
	} else {
		// Expand existing ingredients
		for i, ing := range ingredients {
			if ingTree, ok := ing.(map[string]interface{}); ok {
				ingName, hasName := ingTree["name"].(string)
				if hasName && !alg.IsElementTraceable(ingName, baseElements, graph.NewElementGraph(elements)) {
					ingTree["unmakeable"] = true
					continue
				}
				ensureIngredientsRandomlyExpanded(ingTree, elements, baseElements, visited, seed+i)
			}
		}
	}
}

// Helper function to extract a sufbpath leading to a target element
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
