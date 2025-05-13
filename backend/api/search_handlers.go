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

		if len(makeableTrees) == 0 {
			log.Printf("DEBUG: Still no makeable trees, trying alternative recipes")

			element := h.elements[elementName]
			for _, recipe := range element.Recipes {
				if len(recipe.Ingredients) == 0 {
					continue
				}

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

		if len(makeableTrees) == 0 {
			log.Printf("DEBUG: Unable to find any makeable trees for %s", elementName)
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

	count := 1

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
		tree["unmakeable"] = true
		return false
	}

	ingredients, ok := tree["ingredients"].([]interface{})
	allIngredientsValid := true

	if !ok || len(ingredients) == 0 {
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

		if path[i].Position != 0 {
			elemKey := path[i].Element
			if positionNodeMap[elemKey] == nil {
				positionNodeMap[elemKey] = make(map[int]*model.Node)
			}
			positionNodeMap[elemKey][path[i].Position] = &path[i]
		}
	}

	processedInBranch := make(map[string]bool)
	validityCache := make(map[string]bool)

	isElementMakeable := func(element string) bool {
		for _, base := range baseElements {
			if element == base {
				return true
			}
		}

		if result, ok := validityCache[element]; ok {
			return result
		}

		elemData, exists := elements[element]
		if !exists {
			validityCache[element] = false
			return false
		}

		if len(elemData.Recipes) == 0 {
			validityCache[element] = false
			return false
		}

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

		if posMap, exists := positionNodeMap[element]; exists && len(posMap) > 0 {
			log.Printf("DEBUG: Using position-specific recipe for %s", element)
		}

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

			if !isBase && depth < 10 && len(elemData.Recipes) > 0 {
				recipe := elemData.Recipes[0]
				for _, ingredient := range recipe.Ingredients {
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
			if elemData, exists := elements[element]; exists && len(elemData.Recipes) > 0 {
				ingredients = elemData.Recipes[0].Ingredients
			}
		}

		if depth < 10 {
			for _, ingredient := range ingredients {
				subtree := buildTree(ingredient, depth+1)
				if subtree != nil {
					for i := 0; i < len(ingredients); i++ {
						if ingredients[i] == ingredient && i > 0 {
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
	if unmakeable, ok := tree["unmakeable"].(bool); ok && unmakeable {
		elementName, _ := tree["name"].(string)
		log.Printf("DEBUG: Tree node %s is unmakeable, rejecting tree", elementName)
		return false
	}

	elementName, _ := tree["name"].(string)
	ingredients, hasIngredients := tree["ingredients"].([]interface{})
	isBase, hasBase := tree["isBaseElement"].(bool)

	if (!hasBase || !isBase) && (!hasIngredients || len(ingredients) == 0) {
		log.Printf("DEBUG: Non-base element %s has no ingredients, marking as unmakeable", elementName)
		tree["unmakeable"] = true
		return false
	}

	if hasIngredients {
		for _, ing := range ingredients {
			ingredient, ok := ing.(map[string]interface{})
			if !ok {
				continue
			}

			if !isTreeFullyMakeable(ingredient) {
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

	count := -1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if countParam == "all" {
			count = -1 
		} else if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
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

	searchPathCount := 1000 
	if count > 0 {
		searchPathCount = count * 20 
	}

	paths, visitedCount := alg.MultiThreadedDFS(h.elements, elementName, searchPathCount, false)

	log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visitedCount)

	trees := make([]map[string]interface{}, 0, len(paths))
	uniqueSignatures := make(map[string]bool)
	recipeSignatures := make(map[string]bool)

	for _, path := range paths {
		tree := convertPathToTree(path, elementName, h.elements, baseElements)

		if tree != nil {
			ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

			recipeSig := getTopLevelRecipeSignature(tree)

			signature := generateDetailedTreeSignature(tree)
			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				recipeSignatures[recipeSig] = true
				trees = append(trees, tree)
			}
		}
	}

	log.Printf("DEBUG: Generated %d unique trees from %d paths", len(trees), len(paths))

	shouldGenerateMore := count == -1 ||
		(len(trees) < count || len(recipeSignatures) < min(count, len(element.Recipes)))

	if shouldGenerateMore {
		log.Printf("DEBUG: Generating more trees from element recipes")

		for recipeIdx, recipe := range element.Recipes {
			if count != -1 && (len(trees) >= count*2 || len(recipeSignatures) >= count*2) {
				break
			}

			if len(recipe.Ingredients) == 0 {
				continue
			}

			allIngredientsTraceable := true
			for _, ingredient := range recipe.Ingredients {
				isBaseElement := false
				for _, base := range baseElements {
					if ingredient == base {
						isBaseElement = true
						break
					}
				}

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

			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   element.ImagePath,
				"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
			}

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

				ingTree := map[string]interface{}{
					"name":          ingredient,
					"imagePath":     ingredientData.ImagePath,
					"isBaseElement": isBase,
					"ingredients":   []interface{}{},
				}

				if !isBase {
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

					if !found && len(ingredientData.Recipes) > 0 {
						recipeIndex := (recipeIdx + len(tree["ingredients"].([]interface{}))) % len(ingredientData.Recipes)
						ingRecipe := ingredientData.Recipes[recipeIndex]

						if len(ingRecipe.Ingredients) > 0 {
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

				tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingTree)
			}

			ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))
			recipeSig := getTopLevelRecipeSignature(tree)
			signature := generateDetailedTreeSignature(tree)

			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				recipeSignatures[recipeSig] = true
				trees = append(trees, tree)
				log.Printf("DEBUG: Added direct recipe tree for recipe %d", recipeIdx)
			}
		}

		shouldAddRandomVariations := count == -1 || len(trees) < count
		if shouldAddRandomVariations {
			log.Printf("DEBUG: Adding more tree variations with randomness")

			maxVariations := 50
			if count > 0 {
				maxVariations = count * 2
			}

			for i := 0; i < maxVariations; i++ {
				if count > 0 && len(trees) >= count {
					break
				}

				if len(element.Recipes) == 0 {
					continue
				}

				recipeIdx := i % len(element.Recipes)
				recipe := element.Recipes[recipeIdx]

				if len(recipe.Ingredients) == 0 {
					continue
				}

				allIngredientsTraceable := true
				for _, ingredient := range recipe.Ingredients {
					isBaseElement := false
					for _, base := range baseElements {
						if ingredient == base {
							isBaseElement = true
							break
						}
					}

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

				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
				}

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

				visited := make(map[string]bool)
				ensureIngredientsRandomlyExpanded(tree, h.elements, baseElements, visited, i)

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

	if len(trees) == 0 {
		log.Printf("DEBUG: No trees generated at all, using fallback DFS tree builder")
		g := utils.CreateElementGraph(h.elements)
		visitCount := 0
		visitedNodes := make(map[string]bool)
		tree := utils.BuildElementTreeDFS(g, elementName, visitedNodes, &visitCount)
		trees = []map[string]interface{}{tree}
		visitedCount += visitCount
	}

	sort.Slice(trees, func(i, j int) bool {
		return getTreeComplexityScore(trees[i]) < getTreeComplexityScore(trees[j])
	})

	if count > 0 && len(trees) > count {
		selectedTrees := make([]map[string]interface{}, 0, count)
		selectedRecipes := make(map[string]bool)

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

		if len(selectedTrees) < count {
			existingSigs := make(map[string]bool)
			for _, tree := range selectedTrees {
				sig := generateDetailedTreeSignature(tree)
				existingSigs[sig] = true
			}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


func isTreeFullyTraceable(tree map[string]interface{}, baseElements []string, elements map[string]model.Element) bool {
	if unmakeable, ok := tree["unmakeable"].(bool); ok && unmakeable {
		return false
	}

	elementName, ok := tree["name"].(string)
	if !ok {
		return false
	}

	for _, base := range baseElements {
		if elementName == base {
			return true
		}
	}

	if elementName == "Tree" {
		return false
	}

	g := graph.NewElementGraph(elements)
	if !alg.IsElementTraceable(elementName, baseElements, g) {
		return false
	}

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok {
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

func getTreeComplexityScore(tree map[string]interface{}) int {
	if tree == nil {
		return 0
	}

	if isBase, ok := tree["isBaseElement"].(bool); ok && isBase {
		return 0
	}

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return 1 
	}

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

	return (nonBaseCount * 10) - baseCount + ingredientComplexity
}


func generateDetailedTreeSignature(tree map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString(tree["name"].(string))
	if pos, ok := tree["position"].(int); ok && pos > 0 {
		sb.WriteString(fmt.Sprintf("#%d", pos))
	}

	sb.WriteString(":")

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return sb.String() + "[]"
	}

	elementName, _ := tree["name"].(string)
	preserveOrder := (elementName == "Planet" || elementName == "Continent") ||
		(len(ingredients) >= 2 && ingredients[0].(map[string]interface{})["name"] == ingredients[1].(map[string]interface{})["name"])

	ingredientSignatures := make([]string, 0, len(ingredients))

	for i, ing := range ingredients {
		ingredient, ok := ing.(map[string]interface{})
		if !ok {
			continue
		}

		ingredientSig := generateDetailedTreeSignature(ingredient)

		if i > 0 && ingredient["name"] == ingredients[i-1].(map[string]interface{})["name"] {
			ingredientSig = fmt.Sprintf("%d:%s", i, ingredientSig)
		} else if preserveOrder {
			ingredientSig = fmt.Sprintf("%d:%s", i, ingredientSig)
		}

		ingredientSignatures = append(ingredientSignatures, ingredientSig)
	}

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

	recipeGroups := make(map[string][][]model.Node)

	elementRecipes := element.Recipes
	for _, recipe := range elementRecipes {
		if len(recipe.Ingredients) >= 2 {
			sortedIngs := make([]string, len(recipe.Ingredients))
			copy(sortedIngs, recipe.Ingredients)
			sort.Strings(sortedIngs)
			recipeKey := strings.Join(sortedIngs, "+")
			if _, exists := recipeGroups[recipeKey]; !exists {
				recipeGroups[recipeKey] = [][]model.Node{}
			}
		}
	}

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

		recipeKeys := make([]string, 0, len(recipeGroups))
		for key := range recipeGroups {
			recipeKeys = append(recipeKeys, key)
		}

		sort.Slice(recipeKeys, func(i, j int) bool {
			return len(strings.Split(recipeKeys[i], "+")) < len(strings.Split(recipeKeys[j], "+"))
		})

		for _, recipeKey := range recipeKeys {
			recipePaths := recipeGroups[recipeKey]

			log.Printf("DEBUG: Processing recipe '%s' with %d paths", recipeKey, len(recipePaths))

			treeCreated := false

			if len(recipePaths) > 0 {
				sort.Slice(recipePaths, func(i, j int) bool {
					return len(recipePaths[i]) < len(recipePaths[j])
				})

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

			if !treeCreated {
				log.Printf("DEBUG: Creating manual tree for recipe: %s", recipeKey)

				recipe, exists := dbRecipesByKey[recipeKey]
				if !exists && len(recipePaths) > 0 {
					for _, node := range recipePaths[0] {
						if node.Element == elementName && node.Ingredients != nil && len(node.Ingredients) > 0 {
							recipe.Ingredients = node.Ingredients
							break
						}
					}
				}

				if len(recipe.Ingredients) > 0 && alg.IsRecipeTraceable(recipe.Ingredients, baseElements, graph.NewElementGraph(h.elements)) {
					tree := map[string]interface{}{
						"name":        elementName,
						"imagePath":   element.ImagePath,
						"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
					}

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

						if !isIngBase {
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

					ensureIngredientsExpanded(tree, h.elements, baseElements, make(map[string]bool))

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

			if len(trees) >= count {
				log.Printf("DEBUG: Reached requested tree count (%d), stopping tree generation", count)
				break
			}
		}

		if len(trees) < count {
			log.Printf("DEBUG: Only generated %d/%d trees, trying to create more variations", len(trees), count)

			for _, recipe := range element.Recipes {
				if len(trees) >= count {
					break
				}

				if len(recipe.Ingredients) == 0 {
					continue
				}

				if !alg.IsRecipeTraceable(recipe.Ingredients, baseElements, graph.NewElementGraph(h.elements)) {
					log.Printf("DEBUG: Skipping untraceable recipe for variation: %v", recipe.Ingredients)
					continue
				}

				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
				}

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

				visited := make(map[string]bool)
				ensureIngredientsRandomlyExpanded(tree, h.elements, baseElements, visited, len(trees))

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

	if !alg.IsElementTraceable(elementName, baseElements, graph.NewElementGraph(elements)) {
		tree["unmakeable"] = true
		return
	}

	ingredients, ok := tree["ingredients"].([]interface{})

	if (!ok || len(ingredients) == 0) && !isBase {
		elemData, exists := elements[elementName]
		if exists && len(elemData.Recipes) > 0 {
			var validRecipe model.ElementRecipe
			foundValidRecipe := false

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