package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"backend/utils"
	"log"
	"sort"
	"strings"
	"sync"
)

func DFS(elements map[string]model.Element, target string, maxResults int, debug bool) ([][]model.Node, int) {
	if debug {
		log.Printf("DEBUG: Starting ReverseDFS for target: %s (max results: %d)", target, maxResults)
	}

	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		if debug {
			log.Printf("DEBUG: Target element %s not found in database", target)
		}
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			if debug {
				log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
			}
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	visited := make(map[string]bool)
	visitedCount := 0
	var results [][]model.Node

	if debug {
		log.Printf("DEBUG: Target %s has %d recipes to explore", target, len(targetNode.RecipesToMakeThisElement))
	}

	for _, recipe := range targetNode.RecipesToMakeThisElement {
		path := []*model.Node{
			{Element: target, ImagePath: targetNode.ImagePath},
		}

		Explore(g, recipe, path, visited, &visitedCount, &results, maxResults, baseElements, debug)

		if len(results) >= maxResults && maxResults > 0 {
			if debug {
				log.Printf("DEBUG: Found %d paths, stopping exploration", len(results))
			}
			break
		}
	}

	if debug {
		log.Printf("DEBUG: ReverseDFS complete - found %d paths after visiting %d nodes", len(results), visitedCount)
	}

	return results, visitedCount
}

func Explore(g *graph.ElementGraph, recipe *graph.Recipe, currentPath []*model.Node, visited map[string]bool, visitedCount *int, results *[][]model.Node, maxResults int, baseElements []string, debug bool) {
	if len(*results) >= maxResults && maxResults > 0 {
		return
	}

	if debug {
		log.Printf("DEBUG: Exploring recipe: %s from ingredients: %v", recipe.Result, recipe.Ingredients)
	}

	ingredients := recipe.Ingredients
	if len(ingredients) == 0 {
		if debug {
			log.Printf("DEBUG: Skipping recipe with no ingredients")
		}
		return
	}

	newPath := make([]*model.Node, len(currentPath))
	copy(newPath, currentPath)

	allIngredientsAreBaseElements := true
	ingredientNodes := make([]*model.Node, 0, len(ingredients))

	for _, ingredient := range ingredients {
		ingredientNode := g.Nodes[ingredient]
		*visitedCount++

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

		if !isBase && len(ingredientNode.RecipesToMakeThisElement) > 0 {
			allIngredientsAreBaseElements = false
		}
	}

	newPath = append(newPath, ingredientNodes...)

	if allIngredientsAreBaseElements {
		finalPath := make([]model.Node, len(newPath))
		for i, node := range newPath {
			finalPath[i] = *node
		}

		for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
			finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
		}

		*results = append(*results, finalPath)

		if debug {
			log.Printf("DEBUG: Found complete path with %d steps", len(finalPath))
		}
		return
	}

	for _, ingredient := range ingredients {
		isBase := false
		for _, base := range baseElements {
			if ingredient == base {
				isBase = true
				break
			}
		}

		if isBase {
			if debug {
				log.Printf("DEBUG: Ingredient %s is a base element, skipping further exploration", ingredient)
			}
			continue
		}

		if visited[ingredient] {
			continue
		}

		visited[ingredient] = true

		ingredientNode := g.Nodes[ingredient]
		if debug {
			log.Printf("DEBUG: Exploring ingredient %s which has %d recipes", ingredient, len(ingredientNode.RecipesToMakeThisElement))
		}
		for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
			ingredientPath := make([]*model.Node, len(newPath))
			copy(ingredientPath, newPath)

			Explore(g, subRecipe, ingredientPath, visited, visitedCount, results, maxResults, baseElements, debug)

			if len(*results) >= maxResults && maxResults > 0 {
				break
			}
		}
		// Backtrack
		delete(visited, ingredient)
	}
}

func MultiThreadedElementTreeDFS(g *graph.ElementGraph, elementName string, count int) ([]map[string]interface{}, int) {
	totalVisitedCount := 0
	resultTrees := make([]map[string]interface{}, 0, count)
	uniqueSignatures := make(map[string]bool)

	resultChan := make(chan map[string]interface{}, count*3)
	visitCountChan := make(chan int, count*3)

	node := g.Nodes[elementName]
	if node == nil || len(node.RecipesToMakeThisElement) == 0 {
		visitCount := 0
		visited := make(map[string]bool)
		tree := utils.BuildElementTreeDFS(g, elementName, visited, &visitCount)
		return []map[string]interface{}{tree}, visitCount
	}

	activeGoroutines := 0
	for _, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		utils.GenerateRecipeVariations(g, elementName, node.ImagePath, recipe, &activeGoroutines,
			resultChan, visitCountChan, 0, count)
	}

	log.Printf("DEBUG: Started %d goroutines to explore recipe variations", activeGoroutines)

	if activeGoroutines == 0 {
		visitCount := 0
		visited := make(map[string]bool)
		tree := utils.BuildElementTreeDFS(g, elementName, visited, &visitCount)
		return []map[string]interface{}{tree}, visitCount
	}

	for i := 0; i < activeGoroutines; i++ {
		tree := <-resultChan
		visitCount := <-visitCountChan

		signature := utils.GenerateTreeSignature(tree)
		if !uniqueSignatures[signature] {
			uniqueSignatures[signature] = true
			resultTrees = append(resultTrees, tree)
			totalVisitedCount += visitCount

			if len(resultTrees) >= count {
				log.Printf("DEBUG: Reached target count of %d unique trees, will stop adding more", count)
				continue
			}
		}
	}

	if len(resultTrees) < count {
		log.Printf("DEBUG: Only found %d unique trees from goroutines, generating %d more trees",
			len(resultTrees), count-len(resultTrees))

		for i := len(resultTrees); i < count; i++ {
			visitCount := 0
			visited := make(map[string]bool)
			tree := utils.BuildElementTreeDFS(g, elementName, visited, &visitCount)

			signature := utils.GenerateTreeSignature(tree)
			if !uniqueSignatures[signature] {
				uniqueSignatures[signature] = true
				resultTrees = append(resultTrees, tree)
				totalVisitedCount += visitCount
			}

			if len(resultTrees) >= count || len(uniqueSignatures) >= count*2 {
				break
			}
		}
	}

	log.Printf("DEBUG: Final result contains %d unique trees", len(resultTrees))
	return resultTrees, totalVisitedCount
}

func GetElementTreeDFS(g *graph.ElementGraph, elementName string) (map[string]interface{}, int) {
	visited := make(map[string]bool)
	visitedCount := 0

	result := utils.BuildElementTreeDFS(g, elementName, visited, &visitedCount)
	return result, visitedCount
}

func MultiThreadedDFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		log.Printf("Target element '%s' not found in database", target)
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("Target '%s' is a base element, returning direct path", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	// Filter out invalid recipes first
	validRecipes := make([]*graph.Recipe, 0, len(targetNode.RecipesToMakeThisElement))
	for _, recipe := range targetNode.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		// Skip recipes with duplicate ingredients
		if hasDuplicateIngredients(recipe.Ingredients) {
			continue
		}

		// Skip self-reference recipes
		selfReference := false
		for _, ing := range recipe.Ingredients {
			if ing == target {
				selfReference = true
				break
			}
		}

		if !selfReference {
			validRecipes = append(validRecipes, recipe)
		}
	}

	log.Printf("Found %d valid recipes for '%s' (filtered out %d invalid recipes)",
		len(validRecipes), target, len(targetNode.RecipesToMakeThisElement)-len(validRecipes))

	// Increase buffer size to handle more paths
	resultChan := make(chan []model.Node, maxResults*10)
	visitCountChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex

	uniquePathSignatures := make(map[string]bool)
	totalVisitedCount := 0

	// Add more diverse strategies with different parameters
	strategies := []struct {
		name            string
		maxDepth        int
		favorSimplicity bool
	}{
		{"deep", 40, false},    // Deeper search
		{"simple", 20, true},   // Focus on simple recipes
		{"balanced", 25, true}, // Balanced approach
		{"varied", 30, false},  // More variety
		{"thorough", 35, true}, // More thorough search
		{"wide", 28, false},    // Wide search
		{"diverse", 32, true},  // Add more strategies for diversity
		{"explorer", 38, false},
	}

	log.Printf("Starting %d DFS goroutines with varied strategies to find recipes for '%s'",
		len(strategies), target)

	for _, strategy := range strategies {
		wg.Add(1)

		go func(strat struct {
			name            string
			maxDepth        int
			favorSimplicity bool
		}) {
			defer wg.Done()

			localVisited := make(map[string]bool)
			localCount := 0
			localResults := [][]model.Node{}

			// Process each recipe for the target element
			for _, recipe := range validRecipes {
				// Start with target element
				path := []*model.Node{
					{
						Element:     target,
						ImagePath:   targetNode.ImagePath,
						Ingredients: recipe.Ingredients, // Store recipe ingredients for better tracking
					},
				}

				exploreWithStrategy(g, recipe, path, localVisited, &localCount, &localResults,
					maxResults, baseElements, strat.maxDepth, strat.favorSimplicity)

				// Submit ALL found paths for this recipe
				if len(localResults) > 0 {
					// Submit paths from this recipe exploration
					for _, foundPath := range localResults {
						mu.Lock()
						pathSignature := GeneratePathSignature(foundPath)
						if !uniquePathSignatures[pathSignature] {
							uniquePathSignatures[pathSignature] = true
							resultChan <- foundPath
						}
						mu.Unlock()
					}

					// Reset local results for next recipe
					localResults = [][]model.Node{}

					// If single path mode, exit after finding first valid path
					if singlePath {
						return
					}
				}
			}

			mu.Lock()
			totalVisitedCount += localCount
			mu.Unlock()

			log.Printf("Goroutine '%s' finished with %d paths (visited %d nodes)",
				strat.name, len(localResults), localCount)
		}(strategy)
	}

	// Wait for all goroutines to complete and close channels
	go func() {
		wg.Wait()
		visitCountChan <- totalVisitedCount
		close(resultChan)
		close(visitCountChan)
		log.Printf("All DFS goroutines completed, collected unique paths: %d",
			len(uniquePathSignatures))
	}()

	// Collect all results without early breaking
	results := make([][]model.Node, 0, maxResults*2)
	for path := range resultChan {
		results = append(results, path)
	}

	// Get total visited count
	visitedCount := <-visitCountChan

	// Group paths by recipe signature for better diversity
	recipeGroups := make(map[string][][]model.Node)
	for _, path := range results {
		if len(path) == 0 {
			continue
		}

		// Find target node in path to get its ingredients
		var targetPathNode *model.Node
		for i := range path {
			if path[i].Element == target {
				targetPathNode = &path[i]
				break
			}
		}

		// Group by ingredients used to make the target
		if targetPathNode != nil && targetPathNode.Ingredients != nil && len(targetPathNode.Ingredients) > 0 {
			sortedIngs := make([]string, len(targetPathNode.Ingredients))
			copy(sortedIngs, targetPathNode.Ingredients)
			sort.Strings(sortedIngs)
			recipeKey := strings.Join(sortedIngs, "+")

			if _, exists := recipeGroups[recipeKey]; !exists {
				recipeGroups[recipeKey] = [][]model.Node{}
			}
			recipeGroups[recipeKey] = append(recipeGroups[recipeKey], path)
		}
	}

	// Select diverse paths from each recipe group
	finalResults := make([][]model.Node, 0, maxResults)

	// Process recipe groups to ensure recipe diversity
	recipeKeys := make([]string, 0, len(recipeGroups))
	for key := range recipeGroups {
		recipeKeys = append(recipeKeys, key)
	}

	// Sort recipe keys for consistent ordering
	sort.Strings(recipeKeys)

	// Take paths from each recipe group to ensure diversity
	pathsPerRecipe := 1
	if maxResults > 0 && len(recipeGroups) > 0 {
		pathsPerRecipe = max(1, maxResults/len(recipeGroups))
	}

	for _, recipeKey := range recipeKeys {
		paths := recipeGroups[recipeKey]

		// Sort by path length (shorter first)
		sort.Slice(paths, func(i, j int) bool {
			return len(paths[i]) < len(paths[j])
		})

		// Take up to pathsPerRecipe paths from this recipe
		for i := 0; i < min(pathsPerRecipe, len(paths)); i++ {
			finalResults = append(finalResults, paths[i])
		}
	}

	// If we still need more paths, add remaining ones
	if maxResults > 0 && len(finalResults) < maxResults {
		for _, recipeKey := range recipeKeys {
			paths := recipeGroups[recipeKey]
			// Start from pathsPerRecipe to get additional paths
			for i := pathsPerRecipe; i < len(paths) && len(finalResults) < maxResults; i++ {
				finalResults = append(finalResults, paths[i])
			}
		}
	}

	// If final results is empty, use all results
	if len(finalResults) == 0 {
		finalResults = results
	}

	// Apply final limit if needed
	if maxResults > 0 && len(finalResults) > maxResults {
		finalResults = finalResults[:maxResults]
	}

	if len(finalResults) == 0 {
		log.Printf("No paths found in parallel exploration, falling back to standard DFS")
		return DFS(elements, target, maxResults, false)
	}

	log.Printf("MultiThreadedDFS found %d unique paths across %d recipe groups after visiting %d nodes",
		len(finalResults), len(recipeGroups), visitedCount)
	return finalResults, visitedCount
}

// Check if a recipe contains duplicate ingredients
func hasDuplicateIngredients(ingredients []string) bool {
	seen := make(map[string]bool)
	for _, ingredient := range ingredients {
		if seen[ingredient] {
			return true
		}
		seen[ingredient] = true
	}
	return false
}

func exploreWithStrategy(g *graph.ElementGraph, recipe *graph.Recipe, currentPath []*model.Node, visited map[string]bool, visitCount *int, results *[][]model.Node, maxResults int, baseElements []string, maxDepth int, favorSimplicity bool) {
	if len(currentPath) > maxDepth {
		return
	}

	ingredients := recipe.Ingredients
	if len(ingredients) == 0 {
		return
	}

	// Skip recipes with duplicate ingredients
	if hasDuplicateIngredients(ingredients) {
		return
	}

	// Skip self-reference recipes
	target := currentPath[0].Element
	for _, ing := range ingredients {
		if ing == target {
			return
		}
	}

	// Create a new path with the current element and its ingredients
	newPath := make([]*model.Node, len(currentPath))
	copy(newPath, currentPath)

	// Check if all ingredients are base elements
	allIngredientsAreBaseElements := true
	ingredientNodes := make([]*model.Node, 0, len(ingredients))

	for _, ingredient := range ingredients {
		ingredientNode := g.Nodes[ingredient]
		*visitCount++

		// Create ingredient node
		ingredientNodeObj := &model.Node{
			Element:   ingredient,
			ImagePath: ingredientNode.ImagePath,
		}

		// Check if this is a base element
		isBase := false
		for _, base := range baseElements {
			if ingredient == base {
				isBase = true
				break
			}
		}

		// If not a base element and has recipes, need to explore further
		if !isBase && len(ingredientNode.RecipesToMakeThisElement) > 0 {
			allIngredientsAreBaseElements = false
		}

		ingredientNodes = append(ingredientNodes, ingredientNodeObj)
	}

	// Store recipe information in the target node
	if len(newPath) > 0 && newPath[0].Element == recipe.Result {
		newPath[0].Ingredients = recipe.Ingredients
	}

	// Add ingredient nodes to the path
	newPath = append(newPath, ingredientNodes...)

	// If all ingredients are base elements, we found a complete path
	if allIngredientsAreBaseElements {
		finalPath := make([]model.Node, len(newPath))
		for i, node := range newPath {
			finalPath[i] = *node
		}

		// Reverse the path for correct display (base elements first)
		for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
			finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
		}

		*results = append(*results, finalPath)
		return
	}

	// Sort ingredients to control exploration order
	recipeIngredients := make([]string, len(ingredients))
	copy(recipeIngredients, ingredients)

	// Try different exploration orders based on strategy
	if favorSimplicity {
		// Prioritize base elements first for simple strategies
		sort.SliceStable(recipeIngredients, func(i, j int) bool {
			iIsBase := isBaseElement(recipeIngredients[i], baseElements)
			jIsBase := isBaseElement(recipeIngredients[j], baseElements)
			return iIsBase && !jIsBase
		})
	} else {
		// For exploratory strategies, try different orders
		if *visitCount%3 == 0 { // Sometimes reverse order
			for i, j := 0, len(recipeIngredients)-1; i < j; i, j = i+1, j-1 {
				recipeIngredients[i], recipeIngredients[j] = recipeIngredients[j], recipeIngredients[i]
			}
		} else if *visitCount%3 == 1 { // Sometimes randomize (using visitCount as pseudo-random)
			for i := range recipeIngredients {
				if i > 0 {
					j := (*visitCount + i) % i
					recipeIngredients[i], recipeIngredients[j] = recipeIngredients[j], recipeIngredients[i]
				}
			}
		}
	}

	// For each non-base ingredient, explore all its recipes
	for _, ingredient := range recipeIngredients {
		// Skip base elements
		isBase := isBaseElement(ingredient, baseElements)
		if isBase || visited[ingredient] {
			continue
		}

		// Mark visited to prevent cycles
		visited[ingredient] = true

		ingredientNode := g.Nodes[ingredient]
		if len(ingredientNode.RecipesToMakeThisElement) == 0 {
			delete(visited, ingredient)
			continue
		}

		// Get all recipes for this ingredient
		allRecipes := ingredientNode.RecipesToMakeThisElement
		validRecipes := make([]*graph.Recipe, 0, len(allRecipes))

		// Filter out invalid recipes
		for _, r := range allRecipes {
			if len(r.Ingredients) == 0 || hasDuplicateIngredients(r.Ingredients) {
				continue
			}

			// Check for self-referential recipes
			selfRef := false
			for _, ing := range r.Ingredients {
				if ing == r.Result {
					selfRef = true
					break
				}
			}

			if !selfRef {
				validRecipes = append(validRecipes, r)
			}
		}

		// Sort recipes based on strategy
		if favorSimplicity {
			// Sort by number of base elements (more base elements first)
			sort.Slice(validRecipes, func(i, j int) bool {
				iBaseCount := countBaseElements(validRecipes[i].Ingredients, baseElements)
				jBaseCount := countBaseElements(validRecipes[j].Ingredients, baseElements)

				if iBaseCount != jBaseCount {
					return iBaseCount > jBaseCount
				}

				// Shorter recipes preferred
				return len(validRecipes[i].Ingredients) < len(validRecipes[j].Ingredients)
			})
		}

		// Try all recipes for this ingredient - important for finding all combinations!
		for _, subRecipe := range validRecipes {
			// Create a copy of the path for this branch
			ingredientPath := make([]*model.Node, len(newPath))
			copy(ingredientPath, newPath)

			// Explore this recipe
			exploreWithStrategy(g, subRecipe, ingredientPath, visited, visitCount,
				results, maxResults, baseElements, maxDepth, favorSimplicity)
		}

		// Backtrack
		delete(visited, ingredient)
	}
}

// Helper function to check if an element is a base element
func isBaseElement(element string, baseElements []string) bool {
	for _, base := range baseElements {
		if element == base {
			return true
		}
	}
	return false
}

// Helper function to count base elements in a recipe
func countBaseElements(ingredients []string, baseElements []string) int {
	count := 0
	for _, ing := range ingredients {
		if isBaseElement(ing, baseElements) {
			count++
		}
	}
	return count
}
