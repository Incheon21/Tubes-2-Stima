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

	traceableRecipes := filterTraceableRecipes(g, targetNode.RecipesToMakeThisElement, baseElements)

	if debug {
		log.Printf("DEBUG: %d recipes remain after filtering untraceable recipes", len(traceableRecipes))
	}

	for _, recipe := range traceableRecipes {
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

	validRecipes := make([]*graph.Recipe, 0, len(targetNode.RecipesToMakeThisElement))
	log.Printf("DEBUG: Target '%s' has %d recipes to check", target, len(targetNode.RecipesToMakeThisElement))

	for _, recipe := range targetNode.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		// Skip recipes with duplicate ingredients
		if hasDuplicateIngredients(recipe.Ingredients) {
			log.Printf("DEBUG: Skipping recipe with duplicate ingredients: %v", recipe.Ingredients)
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

		if selfReference {
			log.Printf("DEBUG: Skipping self-referential recipe: %v", recipe.Ingredients)
			continue
		}

		validRecipes = append(validRecipes, recipe)
	}

	// Log all valid recipes before filtering
	log.Printf("DEBUG: Found %d valid recipes for '%s' before traceability check",
		len(validRecipes), target)

	for i, recipe := range validRecipes {
		log.Printf("DEBUG: Recipe %d: %v", i+1, recipe.Ingredients)
	}

	// Filter recipes to only include those with traceable ingredients
	validRecipes = filterTraceableRecipes(g, validRecipes, baseElements)

	log.Printf("Found %d valid traceable recipes for '%s' (filtered out %d invalid recipes)",
		len(validRecipes), target, len(targetNode.RecipesToMakeThisElement)-len(validRecipes))

	// Log all traceable recipes
	for i, recipe := range validRecipes {
		log.Printf("DEBUG: Traceable recipe %d: %v", i+1, recipe.Ingredients)
	}

	// Increase buffer size to handle more paths
	resultChan := make(chan []model.Node, maxResults*20) // Increased buffer size
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
		{"complete", 45, true}, // Added more strategies
		{"exhaustive", 50, false},
	}

	log.Printf("Starting %d DFS goroutines with varied strategies to find recipes for '%s'",
		len(strategies), target)

	// Create a separate goroutine for each recipe to ensure all recipes are explored
	for recipeIndex, recipe := range validRecipes {
		for strategyIndex, strategy := range strategies {
			wg.Add(1)

			go func(recipe *graph.Recipe, strat struct {
				name            string
				maxDepth        int
				favorSimplicity bool
			}, recipeIdx, strategyIdx int) {
				defer wg.Done()

				localVisited := make(map[string]bool)
				localCount := 0
				localResults := [][]model.Node{}

				// Start with target element
				path := []*model.Node{
					{
						Element:     target,
						ImagePath:   targetNode.ImagePath,
						Ingredients: recipe.Ingredients, // Store recipe ingredients for better tracking
					},
				}

				log.Printf("DEBUG: Goroutine '%s' exploring recipe %d: %v (recipe %d of %d)",
					strat.name, recipeIdx+1, recipe.Ingredients, recipeIdx+1, len(validRecipes))

				exploreWithStrategy(g, recipe, path, localVisited, &localCount, &localResults,
					maxResults, baseElements, strat.maxDepth, strat.favorSimplicity)

				// Submit ALL found paths for this recipe
				if len(localResults) > 0 {
					log.Printf("DEBUG: Goroutine '%s' found %d paths for recipe %d",
						strat.name, len(localResults), recipeIdx+1)

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

					if singlePath {
						log.Printf("DEBUG: Single path found, signaling completion")
					}
				} else {
					log.Printf("DEBUG: Goroutine '%s' found NO paths for recipe %d",
						strat.name, recipeIdx+1)
				}

				mu.Lock()
				totalVisitedCount += localCount
				mu.Unlock()
			}(recipe, strategy, recipeIndex, strategyIndex)

			if singlePath {
				break
			}
		}
	}

	go func() {
		wg.Wait()
		visitCountChan <- totalVisitedCount
		close(resultChan)
		close(visitCountChan)
		log.Printf("All DFS goroutines completed, collected unique paths: %d",
			len(uniquePathSignatures))
	}()

	results := make([][]model.Node, 0, maxResults*2)
	for path := range resultChan {
		results = append(results, path)
	}

	visitedCount := <-visitCountChan

	recipeGroups := make(map[string][][]model.Node)
	for _, path := range results {
		if len(path) == 0 {
			continue
		}

		var targetPathNode *model.Node
		for i := range path {
			if path[i].Element == target {
				targetPathNode = &path[i]
				break
			}
		}

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

	log.Printf("DEBUG: Found %d distinct recipe groups:", len(recipeGroups))
	for recipeKey, paths := range recipeGroups {
		log.Printf("DEBUG: Recipe %s has %d paths", recipeKey, len(paths))
	}

	finalResults := make([][]model.Node, 0, maxResults)

	recipeKeys := make([]string, 0, len(recipeGroups))
	for key := range recipeGroups {
		recipeKeys = append(recipeKeys, key)
	}

	sort.Strings(recipeKeys)

	pathsPerRecipe := 1
	if maxResults > 0 && len(recipeGroups) > 0 {
		pathsPerRecipe = max(1, maxResults/len(recipeGroups))
	}

	for _, recipeKey := range recipeKeys {
		paths := recipeGroups[recipeKey]

		sort.Slice(paths, func(i, j int) bool {
			return len(paths[i]) < len(paths[j])
		})

		for i := 0; i < min(pathsPerRecipe, len(paths)); i++ {
			finalResults = append(finalResults, paths[i])
		}
	}

	if maxResults > 0 && len(finalResults) < maxResults {
		for _, recipeKey := range recipeKeys {
			paths := recipeGroups[recipeKey]
			for i := pathsPerRecipe; i < len(paths) && len(finalResults) < maxResults; i++ {
				finalResults = append(finalResults, paths[i])
			}
		}
	}

	if len(finalResults) == 0 {
		finalResults = results
	}

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

	if hasDuplicateIngredients(ingredients) {
		return
	}

	target := currentPath[0].Element
	for _, ing := range ingredients {
		if ing == target {
			return
		}
	}

	for _, ingredient := range ingredients {
		isBase := false
		for _, base := range baseElements {
			if ingredient == base {
				isBase = true
				break
			}
		}

		if !isBase && !IsElementTraceable(ingredient, baseElements, g) {
			log.Printf("DEBUG: Skipping recipe for '%s' with untraceable ingredient: %s", recipe.Result, ingredient)
			return
		}
	}

	newPath := make([]*model.Node, len(currentPath))
	copy(newPath, currentPath)

	allIngredientsAreBaseElements := true
	ingredientNodes := make([]*model.Node, 0, len(ingredients))

	for idx, ingredient := range ingredients {
		ingredientNode := g.Nodes[ingredient]
		*visitCount++

		ingredientNodeObj := &model.Node{
			Element:     ingredient,
			ImagePath:   ingredientNode.ImagePath,
			Position:    idx + 1, 
			Ingredients: nil, 
		}

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

		ingredientNodes = append(ingredientNodes, ingredientNodeObj)
	}

	if len(newPath) > 0 && newPath[0].Element == recipe.Result {
		newPath[0].Ingredients = recipe.Ingredients
	}

	newPath = append(newPath, ingredientNodes...)

	if allIngredientsAreBaseElements {
		finalPath := make([]model.Node, len(newPath))
		for i, node := range newPath {
			finalPath[i] = *node
			finalPath[i].Position = i
		}

		for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
			finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
		}

		*results = append(*results, finalPath)
		return
	}

	recipeIngredients := make([]string, len(ingredients))
	copy(recipeIngredients, ingredients)

	if favorSimplicity {
		sort.SliceStable(recipeIngredients, func(i, j int) bool {
			iIsBase := isBaseElement(recipeIngredients[i], baseElements)
			jIsBase := isBaseElement(recipeIngredients[j], baseElements)
			return iIsBase && !jIsBase
		})
	} else {
		if *visitCount%3 == 0 {
			for i, j := 0, len(recipeIngredients)-1; i < j; i, j = i+1, j-1 {
				recipeIngredients[i], recipeIngredients[j] = recipeIngredients[j], recipeIngredients[i]
			}
		} else if *visitCount%3 == 1 {
			for i := range recipeIngredients {
				if i > 0 {
					j := (*visitCount + i) % i
					recipeIngredients[i], recipeIngredients[j] = recipeIngredients[j], recipeIngredients[i]
				}
			}
		}
	}

	for ingIndex, ingredient := range recipeIngredients {
		isBase := isBaseElement(ingredient, baseElements)
		if isBase || visited[ingredient] {
			continue
		}

		visited[ingredient] = true

		ingredientNode := g.Nodes[ingredient]
		if len(ingredientNode.RecipesToMakeThisElement) == 0 {
			delete(visited, ingredient)
			continue
		}
		allRecipes := ingredientNode.RecipesToMakeThisElement
		validRecipes := make([]*graph.Recipe, 0, len(allRecipes))

		for _, r := range allRecipes {
			if len(r.Ingredients) == 0 || hasDuplicateIngredients(r.Ingredients) {
				continue
			}

			selfRef := false
			for _, ing := range r.Ingredients {
				if ing == r.Result {
					selfRef = true
					break
				}
			}

			if !selfRef {
				allTraceable := true
				for _, ing := range r.Ingredients {
					if !isBaseElement(ing, baseElements) && !IsElementTraceable(ing, baseElements, g) {
						allTraceable = false
						break
					}
				}

				if allTraceable {
					validRecipes = append(validRecipes, r)
				}
			}
		}

		if favorSimplicity {
			sort.Slice(validRecipes, func(i, j int) bool {
				iBaseCount := countBaseElements(validRecipes[i].Ingredients, baseElements)
				jBaseCount := countBaseElements(validRecipes[j].Ingredients, baseElements)

				if iBaseCount != jBaseCount {
					return iBaseCount > jBaseCount
				}

				return len(validRecipes[i].Ingredients) < len(validRecipes[j].Ingredients)
			})
		} else {
			if *visitCount%2 == 0 {
				for i := range validRecipes {
					if i > 0 {
						j := (*visitCount + i + ingIndex) % i
						validRecipes[i], validRecipes[j] = validRecipes[j], validRecipes[i]
					}
				}
			}
		}

		recipesToTry := len(validRecipes)
		if maxResults > 0 && recipesToTry > maxResults/2 {
			recipesToTry = max(2, maxResults/2)
		}

		for i := 0; i < min(recipesToTry, len(validRecipes)); i++ {
			subRecipe := validRecipes[i]

			ingredientPath := make([]*model.Node, len(newPath))
			copy(ingredientPath, newPath)

			exploreWithStrategy(g, subRecipe, ingredientPath, visited, visitCount,
				results, maxResults, baseElements, maxDepth, favorSimplicity)

			if len(*results) >= maxResults*10 && maxResults > 0 {
				break
			}
		}

		delete(visited, ingredient)
	}
}

func isBaseElement(element string, baseElements []string) bool {
	for _, base := range baseElements {
		if element == base {
			return true
		}
	}
	return false
}

func countBaseElements(ingredients []string, baseElements []string) int {
	count := 0
	for _, ing := range ingredients {
		if isBaseElement(ing, baseElements) {
			count++
		}
	}
	return count
}

func filterTraceableRecipes(g *graph.ElementGraph, recipes []*graph.Recipe, baseElements []string) []*graph.Recipe {
	result := make([]*graph.Recipe, 0, len(recipes))

	for _, recipe := range recipes {
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

			if !isBaseElement && !IsElementTraceable(ingredient, baseElements, g) {
				log.Printf("DEBUG: Recipe ingredient '%s' is not traceable to base elements", ingredient)
				allIngredientsTraceable = false
				break
			}
		}

		if allIngredientsTraceable {
			result = append(result, recipe)
		} else {
			log.Printf("DEBUG: Skipping recipe with untraceable ingredients: %v", recipe.Ingredients)
		}
	}

	return result
}
