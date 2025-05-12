package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	log.Printf("DEBUG: Starting top-down BFS for target: %s (max results: %d)", target, maxResults)

	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		log.Printf("DEBUG: Target element %s not found in database", target)
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	visitedCount := 0
	var completePaths [][]model.Node   // Paths where all elements can be traced to base elements
	var incompletePaths [][]model.Node // Paths with unmakeable elements

	log.Printf("DEBUG: Target %s has %d recipes to explore", target, len(targetNode.RecipesToMakeThisElement))

	type queueItem struct {
		recipe *graph.Recipe
		path   []*model.Node
	}

	uniquePaths := make(map[string]bool)

	// Try all recipes for the target element
	for _, recipe := range targetNode.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue // Skip recipes with no ingredients
		}

		startPath := []*model.Node{
			{Element: target, ImagePath: targetNode.ImagePath},
		}

		queue := []queueItem{
			{recipe: recipe, path: startPath},
		}

		visited := make(map[string]bool)
		visited[target] = true

		for len(queue) > 0 && (len(completePaths) < maxResults || !singlePath) {
			current := queue[0]
			queue = queue[1:]

			currentRecipe := current.recipe
			currentPath := current.path

			visitedCount++

			allIngredientsAreBaseElements := true
			hasUnmakeableElement := false
			ingredientNodes := make([]*model.Node, 0, len(currentRecipe.Ingredients))

			// Check if all ingredients are either base elements or have recipes
			for _, ingredient := range currentRecipe.Ingredients {
				ingredientNode := g.Nodes[ingredient]
				if ingredientNode == nil {
					// Ingredient not found in database
					hasUnmakeableElement = true
					continue
				}

				ingredientNodeObj := &model.Node{
					Element:   ingredient,
					ImagePath: ingredientNode.ImagePath,
				}
				ingredientNodes = append(ingredientNodes, ingredientNodeObj)

				isBase := false
				for _, base := range baseElements {
					if ingredient == base {
						isBase = true
						break
					}
				}

				// If not a base element and has no recipes, it's unmakeable
				if !isBase && len(ingredientNode.RecipesToMakeThisElement) == 0 {
					hasUnmakeableElement = true
				}

				// If not a base element and has recipes, not all ingredients are base elements
				if !isBase && len(ingredientNode.RecipesToMakeThisElement) > 0 {
					allIngredientsAreBaseElements = false
				}
			}

			// Skip if we're looking for a single path and found unmakeable elements
			if singlePath && hasUnmakeableElement {
				continue
			}

			newPath := make([]*model.Node, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, ingredientNodes...)

			// All ingredients are base elements or we've reached a complete branch
			if allIngredientsAreBaseElements {
				finalPath := make([]model.Node, len(newPath))
				for i, node := range newPath {
					finalPath[i] = *node
				}

				for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
					finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
				}

				pathSignature := GeneratePathSignature(finalPath)
				if !uniquePaths[pathSignature] {
					uniquePaths[pathSignature] = true

					if !hasUnmakeableElement {
						// This is a complete path where all elements can be traced to base elements
						completePaths = append(completePaths, finalPath)
						log.Printf("DEBUG: Found complete path with %d steps", len(finalPath))

						if singlePath {
							// If we only need one complete path, return it immediately
							return [][]model.Node{finalPath}, visitedCount
						}
					} else if !singlePath {
						// Add to incomplete paths if we're collecting multiple paths
						incompletePaths = append(incompletePaths, finalPath)
						log.Printf("DEBUG: Found incomplete path with %d steps (has unmakeable elements)", len(finalPath))
					}
				}

				continue
			}

			// Continue exploring non-base ingredients
			for _, ingredient := range currentRecipe.Ingredients {
				isBase := false
				for _, base := range baseElements {
					if ingredient == base {
						isBase = true
						break
					}
				}

				if isBase || visited[ingredient] {
					continue
				}

				ingredientNode := g.Nodes[ingredient]
				if ingredientNode == nil || len(ingredientNode.RecipesToMakeThisElement) == 0 {
					// Skip unmakeable elements
					continue
				}

				visited[ingredient] = true

				for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
					if len(subRecipe.Ingredients) == 0 {
						continue
					}

					// Create new path for this recipe branch
					ingredientPath := make([]*model.Node, len(newPath))
					copy(ingredientPath, newPath)

					// Add to queue
					queue = append(queue, queueItem{
						recipe: subRecipe,
						path:   ingredientPath,
					})
				}

				delete(visited, ingredient)
			}
		}
	}

	// Prefer complete paths over incomplete ones
	var results [][]model.Node

	if len(completePaths) > 0 {
		// Add this new block right here, before assigning completePaths to results

		if singlePath && len(completePaths) > 0 {
			var bestPath []model.Node
			bestLength := int(^uint(0) >> 1)

			for _, path := range completePaths {
				if IsFullyComposablePath(path, baseElements, g) && len(path) < bestLength {
					bestPath = path
					bestLength = len(path)
					log.Printf("DEBUG: Found fully composable path with %d steps", len(path))
				}
			}

			if bestPath != nil {
				log.Printf("DEBUG: Returning best fully composable path with %d steps", len(bestPath))
				return [][]model.Node{bestPath}, visitedCount
			}

			log.Printf("DEBUG: No fully composable path found, returning first complete path")
			return [][]model.Node{completePaths[0]}, visitedCount
		}

		results = completePaths
		log.Printf("DEBUG: Returning %d complete paths", len(results))
	} else if !singlePath && len(incompletePaths) > 0 {
		results = incompletePaths
		log.Printf("DEBUG: No complete paths found, returning %d incomplete paths", len(results))
	}

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	log.Printf("DEBUG: BFS completed - found %d paths after visiting %d nodes", len(results), visitedCount)

	return results, visitedCount
}

// Sekarang fungsi MultiThreadedBFS yang perlu diganti
func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)
	targetNode, ok := g.Nodes[target]
	if !ok {
		log.Printf("DEBUG: Target element %s not found in database", target)
		return nil, 0
	}

	// Check if it's a base element
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	validRecipes := make([]*graph.Recipe, 0)
	for _, r := range targetNode.RecipesToMakeThisElement {
		if len(r.Ingredients) > 0 {
			validRecipes = append(validRecipes, r)
		}
	}

	if len(validRecipes) == 0 {
		log.Printf("DEBUG: No valid recipes found for %s", target)
		return nil, 0
	}

	log.Printf("DEBUG: Starting MultiThreaded BFS for target: %s with %d recipes", target, len(validRecipes))

	// Log target's recipes untuk debug
	for i, recipe := range validRecipes {
		log.Printf("DEBUG: Target %s Recipe %d: %v", target, i, recipe.Ingredients)
	}

	resultChan := make(chan []model.Node, maxResults*10)
	completePathChan := make(chan []model.Node, maxResults*5) // Channel for complete paths
	stopChan := make(chan struct{})                           // Channel to signal early termination

	var mu sync.Mutex
	var wg sync.WaitGroup
	visitedCount := 0

	// Process each recipe in a separate goroutine
	for i, recipe := range validRecipes {
		wg.Add(1)
		go func(rcp *graph.Recipe, recipeIdx int) {
			defer wg.Done()
			log.Printf("DEBUG: Goroutine %d starting with recipe: %v", recipeIdx, rcp.Ingredients)

			type queueItem struct {
				path       []model.Node
				recipe     *graph.Recipe
				deadEndIng map[string]bool // Track ingredients that lead to dead ends
			}

			localVisited := 0
			queue := []queueItem{{
				path: []model.Node{{
					Element:     target,
					ImagePath:   targetNode.ImagePath,
					Ingredients: rcp.Ingredients,
				}},
				recipe:     rcp,
				deadEndIng: make(map[string]bool),
			}}

			// Track visited paths to avoid cycles
			visited := make(map[string]bool)

			for len(queue) > 0 {
				// Check if we should stop processing
				select {
				case <-stopChan:
					return
				default:
					// Continue processing
				}

				item := queue[0]
				queue = queue[1:]
				localVisited++

				// Check if all ingredients in this recipe are base elements or traceable
				allBase := true
				hasDeadEnd := false
				newPath := item.path
				pathSignature := GeneratePathSignature(newPath)

				// Build nodes for the ingredients
				nextNodes := make([]model.Node, 0, len(item.recipe.Ingredients))

				// Check ingredients
				for _, ing := range item.recipe.Ingredients {
					ingNode := g.Nodes[ing]
					if ingNode == nil {
						// Ingredient not found in database
						hasDeadEnd = true
						item.deadEndIng[ing] = true
						continue
					}

					isBase := false
					for _, base := range baseElements {
						if ing == base {
							isBase = true
							break
						}
					}

					// Create a node for this ingredient
					nextNodes = append(nextNodes, model.Node{
						Element:   ing,
						ImagePath: ingNode.ImagePath,
						// We'll populate ingredients later if needed
					})

					// If not a base element and cannot be traced further, it's a dead end
					if !isBase && len(ingNode.RecipesToMakeThisElement) == 0 {
						hasDeadEnd = true
						item.deadEndIng[ing] = true
					}

					// If not a base element and can be traced, not all ingredients are base elements
					if !isBase && len(ingNode.RecipesToMakeThisElement) > 0 {
						allBase = false
					}
				}

				// Skip paths that have dead ends when looking for a single complete path
				if singlePath && hasDeadEnd {
					continue
				}

				// Create a new path with the current ingredients
				for i := range nextNodes {
					newPath = append(newPath, nextNodes[i])
				}

				if allBase || (hasDeadEnd && !singlePath) {
					// This is a complete path (all ingredients are base elements)
					// or an acceptable incomplete path when not in single path mode

					// Reverse the path to start from base elements
					reversedPath := make([]model.Node, len(newPath))
					for i, j := 0, len(newPath)-1; i < len(newPath); i, j = i+1, j-1 {
						reversedPath[i] = newPath[j]
					}

					mu.Lock()
					// Skip path signature since we're checking for duplicates in the collecting phase
					log.Printf("DEBUG: Found complete path in goroutine %d: %s", recipeIdx, pathToString(reversedPath))

					if allBase && !hasDeadEnd {
						// Prioritize complete paths
						completePathChan <- reversedPath
						if singlePath {
							// Early termination for single path mode with a complete path
							close(stopChan) // Signal to stop processing
						}
					} else if !singlePath {
						// Send incomplete paths to regular channel if not in single path mode
						resultChan <- reversedPath
					}
					mu.Unlock()
					continue
				}

				// Continue exploration - process ingredients that need further tracing
				for idx, ing := range item.recipe.Ingredients {
					// Skip base elements and ingredients already known to be dead ends
					isBase := false
					for _, base := range baseElements {
						if ing == base {
							isBase = true
							break
						}
					}

					if isBase || item.deadEndIng[ing] {
						continue
					}

					ingNode := g.Nodes[ing]
					if ingNode == nil || len(ingNode.RecipesToMakeThisElement) == 0 {
						continue
					}

					// Try different recipes for this ingredient with varied selection strategy
					ingRecipes := ingNode.RecipesToMakeThisElement

					// No recipes to explore
					if len(ingRecipes) == 0 {
						continue
					}

					log.Printf("DEBUG: Exploring ingredient %s with %d possible recipes", ing, len(ingRecipes))

					// Khusus untuk elemen yang muncul beberapa kali dalam resep
					// Hitung berapa kali elemen ini muncul dalam resep saat ini
					ingCount := 0
					for _, recipeIng := range item.recipe.Ingredients {
						if recipeIng == ing {
							ingCount++
						}
					}

					// Jika elemen muncul beberapa kali dan punya beberapa cara pembuatan
					// kita perlu mencoba lebih banyak variasi resep
					if ingCount > 1 && len(ingNode.RecipesToMakeThisElement) > 1 {
						log.Printf("DEBUG: Special case: %s appears %d times in recipe and has %d ways to make it",
							ing, ingCount, len(ingNode.RecipesToMakeThisElement))

						// Coba semua kombinasi resep untuk setiap kemunculan
						for i := 0; i < len(ingRecipes); i++ {
							recipeIdx := i // Gunakan semua resep langsung
							ingRecipe := ingRecipes[recipeIdx]

							if len(ingRecipe.Ingredients) == 0 {
								continue
							}

							log.Printf("DEBUG: Trying ingredient %s recipe permutation %d: %v",
								ing, recipeIdx, ingRecipe.Ingredients)

							// Create a copy of the path up to the current node
							basePath := make([]model.Node, len(item.path))
							copy(basePath, item.path)

							// Add this specific ingredient node with its recipe
							nextNode := model.Node{
								Element:     ing,
								ImagePath:   ingNode.ImagePath,
								Ingredients: ingRecipe.Ingredients,
							}

							// Create new path for this recipe branch
							newItem := queueItem{
								path:       append(basePath, nextNode),
								recipe:     ingRecipe,
								deadEndIng: make(map[string]bool),
							}

							// Copy over known dead ends
							for k, v := range item.deadEndIng {
								newItem.deadEndIng[k] = v
							}

							// Use a unique signature for each position to explore all options
							positionSig := strconv.Itoa(idx) // Include position in signature
							newPathSig := ing + ":" + pathSignature + ":" + strconv.Itoa(i) + ":" + positionSig

							if !visited[newPathSig] {
								visited[newPathSig] = true
								queue = append(queue, newItem)
								log.Printf("DEBUG: Adding unique permutation for %s (recipe %d of %d) at position %d",
									ing, i+1, len(ingRecipes), idx)
							}
						}
					} else {
						// Kasus normal - gunakan permutation seed
						permutationSeed := (recipeIdx*31 + localVisited*17 + idx*7) % max(1, len(ingRecipes))

						// Explore recipes with a permuted order
						for i := 0; i < len(ingRecipes); i++ {
							recipeIdx := (permutationSeed + i) % len(ingRecipes)
							ingRecipe := ingRecipes[recipeIdx]

							if len(ingRecipe.Ingredients) == 0 {
								continue
							}

							// Create a copy of the path up to the current node
							basePath := make([]model.Node, len(item.path))
							copy(basePath, item.path)

							// Add this specific ingredient node with its recipe
							nextNode := model.Node{
								Element:     ing,
								ImagePath:   ingNode.ImagePath,
								Ingredients: ingRecipe.Ingredients,
							}

							// Create new path for this recipe branch
							newItem := queueItem{
								path:       append(basePath, nextNode),
								recipe:     ingRecipe,
								deadEndIng: make(map[string]bool),
							}

							// Copy over known dead ends
							for k, v := range item.deadEndIng {
								newItem.deadEndIng[k] = v
							}

							// Check for cycles to avoid infinite recursion
							newPathSig := ing + ":" + pathSignature + ":" + strconv.Itoa(recipeIdx)
							if !visited[newPathSig] {
								visited[newPathSig] = true
								queue = append(queue, newItem)
							}
						}
					}
				}
			}

			mu.Lock()
			visitedCount += localVisited
			mu.Unlock()
		}(recipe, i)
	}

	// Wait for all goroutines to finish or until we get early termination signal
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(completePathChan)
		close(resultChan)
	}()

	// Collect results, prioritizing complete paths
	results := [][]model.Node{}
	seenSignatures := map[string]bool{}

	// First, collect all complete paths
	completePaths := [][]model.Node{}
collectLoop:
	for {
		select {
		case path, ok := <-completePathChan:
			if !ok {
				break collectLoop
			}
			sig := GeneratePathSignature(path)
			if !seenSignatures[sig] {
				seenSignatures[sig] = true
				completePaths = append(completePaths, path)

				// If we have enough results and not in single path mode, break
				if !singlePath && maxResults > 0 && len(completePaths) >= maxResults {
					break collectLoop
				}
			}
		case <-done:
			break collectLoop
		case <-stopChan:
			break collectLoop
		}
	}

	// Log info untuk debugging
	log.Printf("DEBUG: Initial collection found %d paths for %s", len(completePaths), target)

	// Extract more diverse paths by comparing recipe signatures
	uniqueRecipeSignatures := make(map[string]bool)
	diversePaths := make([][]model.Node, 0)

	for _, path := range completePaths {
		recipeSignature := generateRecipeSignature(path)
		if !uniqueRecipeSignatures[recipeSignature] {
			uniqueRecipeSignatures[recipeSignature] = true
			diversePaths = append(diversePaths, path)
			log.Printf("DEBUG: Added diverse path: %s with recipe signature: %s",
				pathToString(path), recipeSignature)
		}
	}

	log.Printf("DEBUG: Found %d unique recipe combinations after filtering", len(diversePaths))

	// If we have diverse paths, use them
	if len(diversePaths) > 0 {
		completePaths = diversePaths
	}

	// For single path mode, find the best fully composable path
	if singlePath && len(completePaths) > 0 {
		// Filter for fully composable paths first
		var composablePaths [][]model.Node
		log.Printf("DEBUG: Checking %d complete paths for full composability", len(completePaths))

		for _, path := range completePaths {
			if IsFullyComposablePath(path, baseElements, g) {
				log.Printf("DEBUG: Found fully composable path with %d steps", len(path))
				composablePaths = append(composablePaths, path)
			} else {
				log.Printf("DEBUG: Rejecting path with unmakeable elements (%d steps)", len(path))
			}
		}

		if len(composablePaths) > 0 {
			// Sort by length for consistency
			sort.Slice(composablePaths, func(i, j int) bool {
				return len(composablePaths[i]) < len(composablePaths[j])
			})

			// Get the middle path for more interesting trees
			middleIndex := len(composablePaths) / 2
			selectedPath := composablePaths[middleIndex]

			log.Printf("DEBUG: Selected middle fully composable path with %d steps (path %d of %d)",
				len(selectedPath), middleIndex+1, len(composablePaths))

			return [][]model.Node{selectedPath}, visitedCount
		}

		// If no fully composable paths, try to find a best effort path
		log.Printf("DEBUG: No fully composable paths found, trying to find a best effort path")
		var bestPath []model.Node
		var bestScore int = -1

		// Score each path based on how many ingredients can be traced back to base elements
		for _, path := range completePaths {
			score := scorePathTraceability(path, baseElements, g)
			if score > bestScore {
				bestScore = score
				bestPath = path
			}
		}

		if bestPath != nil {
			log.Printf("DEBUG: Found best effort path with traceability score %d", bestScore)
			return [][]model.Node{bestPath}, visitedCount
		}

		log.Printf("DEBUG: No good path found, falling back to first complete path")
		return [][]model.Node{completePaths[0]}, visitedCount
	}

	// For multiple results, prioritize fully composable paths
	if !singlePath {
		// Sort paths: fully composable first, then by length
		sort.Slice(completePaths, func(i, j int) bool {
			iComposable := IsFullyComposablePath(completePaths[i], baseElements, g)
			jComposable := IsFullyComposablePath(completePaths[j], baseElements, g)

			if iComposable != jComposable {
				return iComposable // True if i is composable and j is not
			}

			// If both have same composability, shorter path wins
			return len(completePaths[i]) < len(completePaths[j])
		})

		results = completePaths
		if maxResults > 0 && len(results) > maxResults {
			results = results[:maxResults]
		}
	}

	// If we need more results and aren't in single path mode, collect from regular channel
	if !singlePath && (maxResults == 0 || len(results) < maxResults) {
		remainingLimit := 0
		if maxResults > 0 {
			remainingLimit = maxResults - len(results)
		}

	incompleteLoop:
		for {
			select {
			case path, ok := <-resultChan:
				if !ok {
					break incompleteLoop
				}
				sig := GeneratePathSignature(path)
				if !seenSignatures[sig] {
					seenSignatures[sig] = true
					results = append(results, path)
					if remainingLimit > 0 && len(results) >= maxResults {
						break incompleteLoop
					}
				}
			case <-done:
				break incompleteLoop
			default:
				// If there are no more immediate paths but goroutines are still running
				if len(results) > 0 {
					break incompleteLoop
				}
			}
		}
	}

	// Clean up channels if needed
	select {
	case <-done:
		// All goroutines finished normally
	default:
		// Signal all goroutines to stop if they haven't already
		select {
		case <-stopChan:
			// Already closed
		default:
			close(stopChan)
		}
		<-done // Wait for all goroutines to actually finish
	}

	log.Printf("DEBUG: MultiThreaded BFS completed - found %d paths after visiting %d nodes", len(results), visitedCount)
	return results, visitedCount
}

func pathToString(path []model.Node) string {
	elements := make([]string, len(path))
	for i, node := range path {
		elemStr := node.Element
		if len(node.Ingredients) > 0 {
			ingredStr := strings.Join(node.Ingredients, "+")
			elemStr = fmt.Sprintf("%s(%s)", elemStr, ingredStr)
		}
		elements[i] = elemStr
	}
	return strings.Join(elements, " -> ")
}

// Tambahkan fungsi baru untuk debug yang lebih detail
func detailedPathToString(path []model.Node) string {
	var sb strings.Builder
	sb.WriteString("Path details:\n")

	for i, node := range path {
		sb.WriteString(fmt.Sprintf("  [%d] %s", i, node.Element))

		if len(node.Ingredients) > 0 {
			sb.WriteString(" made from: ")
			sb.WriteString(strings.Join(node.Ingredients, " + "))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// Tambahkan fungsi untuk menghasilkan recipe signature
func generateRecipeSignature(path []model.Node) string {
	var sig strings.Builder

	for _, node := range path {
		// Sort ingredients untuk normalisasi urutan
		if len(node.Ingredients) > 0 {
			sortedIngs := make([]string, len(node.Ingredients))
			copy(sortedIngs, node.Ingredients)
			sort.Strings(sortedIngs)

			sig.WriteString(node.Element)
			sig.WriteString("(")
			sig.WriteString(strings.Join(sortedIngs, "+"))
			sig.WriteString("),")
		}
	}

	return sig.String()
}

// Helper function to find maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsFullyComposablePath(path []model.Node, baseElements []string, g *graph.ElementGraph) bool {
	// Create a map for quick base element lookup
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	// Track elements we're currently processing to detect circular dependencies
	processingStack := make(map[string]bool)

	// Cache results to avoid repeated checks
	validityCache := make(map[string]bool)

	// Define a recursive helper function to check if an element can be traced to base elements
	var isElementTraceable func(element string) bool

	isElementTraceable = func(element string) bool {
		// Base elements are always traceable
		if baseMap[element] {
			return true
		}

		// Check cache
		if result, exists := validityCache[element]; exists {
			return result
		}

		// Detect circular references
		if processingStack[element] {
			log.Printf("DEBUG: Circular reference detected for element %s", element)
			validityCache[element] = false
			return false
		}

		// Get node from graph
		elementNode := g.Nodes[element]
		if elementNode == nil {
			log.Printf("DEBUG: Element %s not found in graph", element)
			validityCache[element] = false
			return false
		}

		// If no recipes to make this element, it's not traceable
		if len(elementNode.RecipesToMakeThisElement) == 0 {
			log.Printf("DEBUG: Element %s has no recipes, not traceable", element)
			validityCache[element] = false
			return false
		}

		// Mark as being processed
		processingStack[element] = true
		defer delete(processingStack, element)

		// Check if at least one recipe has all traceable ingredients
		recipeValid := false
		for _, recipe := range elementNode.RecipesToMakeThisElement {
			if len(recipe.Ingredients) == 0 {
				continue
			}

			allIngredientsTraceable := true
			for _, ing := range recipe.Ingredients {
				if !isElementTraceable(ing) {
					allIngredientsTraceable = false
					break
				}
			}

			if allIngredientsTraceable {
				recipeValid = true
				break
			}
		}

		if !recipeValid {
			log.Printf("DEBUG: Element %s has no valid recipes (all lead to unmakeable elements)", element)
			validityCache[element] = false
			return false
		}

		validityCache[element] = true
		return true
	}

	// For each node in the path, verify it can be traced to base elements
	for _, node := range path {
		// Skip base elements
		if baseMap[node.Element] {
			continue
		}

		// If not a base element, it must be traceable
		if !isElementTraceable(node.Element) {
			log.Printf("DEBUG: Path element %s cannot be traced to base elements", node.Element)
			return false
		}
	}

	return true
}

func appendPath(path []model.Node, element, imgPath string, ingredients []string) []model.Node {
	newp := make([]model.Node, len(path), len(path)+1)
	copy(newp, path)
	newp = append(newp, model.Node{
		Element:     element,
		ImagePath:   imgPath,
		Ingredients: ingredients,
	})
	return newp
}

func deduplicatePath(path []model.Node) []model.Node {
	seen := make(map[string]bool)
	result := make([]model.Node, 0, len(path))

	for _, node := range path {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	return result
}
func GeneratePathSignature(path []model.Node) string {
	uniquePath := deduplicatePath(path)

	var signature strings.Builder

	for i, node := range uniquePath {
		signature.WriteString(node.Element)

		if len(node.Ingredients) > 0 {
			signature.WriteString("(")

			sortedIngredients := make([]string, len(node.Ingredients))
			copy(sortedIngredients, node.Ingredients)
			sort.Strings(sortedIngredients)

			for j, ing := range sortedIngredients {
				signature.WriteString(ing)
				if j < len(sortedIngredients)-1 {
					signature.WriteString(",")
				}
			}
			signature.WriteString(")")
		}

		if i < len(uniquePath)-1 {
			signature.WriteString("-")
		}
	}

	return signature.String()
}

// Add this function to expose the element traceability check
func IsElementTraceable(element string, baseElements []string, g *graph.ElementGraph) bool {
	// Create a map for quick base element lookup
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	// Base elements are always traceable
	if baseMap[element] {
		return true
	}

	// Use a cache to avoid repeated checks
	validityCache := make(map[string]bool)
	processingStack := make(map[string]bool)

	var isTraceable func(string) bool
	isTraceable = func(elem string) bool {
		// Base elements are always traceable
		if baseMap[elem] {
			return true
		}

		// Check cache
		if result, exists := validityCache[elem]; exists {
			return result
		}

		// Detect circular references
		if processingStack[elem] {
			validityCache[elem] = false
			return false
		}

		// Get node from graph
		elementNode := g.Nodes[elem]
		if elementNode == nil {
			validityCache[elem] = false
			return false
		}

		// If no recipes to make this element, it's not traceable
		if len(elementNode.RecipesToMakeThisElement) == 0 {
			validityCache[elem] = false
			return false
		}

		// Mark as being processed
		processingStack[elem] = true
		defer delete(processingStack, elem)

		// Check if at least one recipe has all traceable ingredients
		recipeValid := false
		for _, recipe := range elementNode.RecipesToMakeThisElement {
			if len(recipe.Ingredients) == 0 {
				continue
			}

			allIngredientsTraceable := true
			for _, ing := range recipe.Ingredients {
				if !isTraceable(ing) {
					allIngredientsTraceable = false
					break
				}
			}

			if allIngredientsTraceable {
				recipeValid = true
				break
			}
		}

		validityCache[elem] = recipeValid
		return recipeValid
	}

	return isTraceable(element)
}

// Add this helper function to score paths based on traceability
func scorePathTraceability(path []model.Node, baseElements []string, g *graph.ElementGraph) int {
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	// Count how many elements are base or traceable
	traceableCount := 0
	unmakeableCount := 0

	for _, node := range path {
		if baseMap[node.Element] {
			traceableCount++
		} else if IsElementTraceable(node.Element, baseElements, g) {
			traceableCount++
		} else {
			unmakeableCount++
		}
	}

	// Prefer paths with no unmakeable elements
	if unmakeableCount == 0 {
		return 1000 + traceableCount
	}

	// Otherwise score based on ratio of traceable to unmakeable
	return traceableCount - (unmakeableCount * 10)
}
