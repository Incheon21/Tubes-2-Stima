package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"backend/utils"
	"log"
	"sort"
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

	// Channel to collect results from different goroutines
	resultChan := make(chan []model.Node, maxResults*2)
	visitCountChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex

	uniquePathSignatures := make(map[string]bool)
	totalVisitedCount := 0

	strategies := []struct {
		name            string
		maxDepth        int
		favorSimplicity bool
	}{
		{"deep", 30, false},
		{"simple", 20, true},
		{"balanced", 25, true},
		{"varied", 22, false},
	}

	routineCount := 0
	for _, strategy := range strategies {
		wg.Add(1)
		routineCount++

		go func(strat struct {
			name            string
			maxDepth        int
			favorSimplicity bool
		}) {
			defer wg.Done()

			localVisited := make(map[string]bool)
			localCount := 0
			localResults := [][]model.Node{}

			for _, recipe := range targetNode.RecipesToMakeThisElement {
				if len(recipe.Ingredients) == 0 {
					continue
				}

				path := []*model.Node{
					{Element: target, ImagePath: targetNode.ImagePath},
				}

				exploreWithStrategy(g, recipe, path, localVisited, &localCount, &localResults, maxResults, baseElements, strat.maxDepth, strat.favorSimplicity)

				if len(localResults) > 0 {
					sort.Slice(localResults, func(i, j int) bool {
						return len(localResults[i]) < len(localResults[j])
					})

					bestPath := localResults[0]

					mu.Lock()
					pathSignature := GeneratePathSignature(bestPath)
					if !uniquePathSignatures[pathSignature] {
						uniquePathSignatures[pathSignature] = true
						resultChan <- bestPath
						totalVisitedCount += localCount
					}
					mu.Unlock()

					if singlePath {
						return
					}
				}
			}

			log.Printf("Goroutine '%s' finished with %d paths", strat.name, len(localResults))
		}(strategy)
	}

	go func() {
		wg.Wait()
		visitCountChan <- totalVisitedCount
		close(resultChan)
		close(visitCountChan)
	}()

	results := make([][]model.Node, 0, maxResults)
	for path := range resultChan {
		results = append(results, path)
		if len(results) >= maxResults {
			break
		}
	}

	visitedCount := <-visitCountChan

	if len(results) == 0 {
		log.Printf("No paths found in parallel exploration, falling back to standard DFS")
		return DFS(elements, target, maxResults, false)
	}

	log.Printf("MultiThreadedDFS found %d unique paths after visiting %d nodes", len(results), visitedCount)
	return results, visitedCount
}

func exploreWithStrategy(g *graph.ElementGraph, recipe *graph.Recipe, currentPath []*model.Node, visited map[string]bool, visitCount *int, results *[][]model.Node, maxResults int, baseElements []string, maxDepth int, favorSimplicity bool) {
	if len(currentPath) > maxDepth {
		return
	}

	ingredients := recipe.Ingredients
	if len(ingredients) == 0 {
		return
	}

	newPath := make([]*model.Node, len(currentPath))
	copy(newPath, currentPath)

	allIngredientsAreBaseElements := true
	ingredientNodes := make([]*model.Node, 0, len(ingredients))

	baseElementCount := 0
	for _, ingredient := range ingredients {
		ingredientNode := g.Nodes[ingredient]
		*visitCount++

		ingredientNodeObj := &model.Node{
			Element:   ingredient,
			ImagePath: ingredientNode.ImagePath,
		}
		ingredientNodes = append(ingredientNodes, ingredientNodeObj)

		isBase := false
		for _, base := range baseElements {
			if ingredient == base {
				isBase = true
				baseElementCount++
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
		return
	}

	recipeIngredients := make([]string, len(ingredients))
	copy(recipeIngredients, ingredients)

	if favorSimplicity {
		sort.SliceStable(recipeIngredients, func(i, j int) bool {
			iIsBase := false
			jIsBase := false

			for _, base := range baseElements {
				if recipeIngredients[i] == base {
					iIsBase = true
				}
				if recipeIngredients[j] == base {
					jIsBase = true
				}
			}

			return iIsBase && !jIsBase
		})
	}

	for _, ingredient := range recipeIngredients {
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

		visited[ingredient] = true

		ingredientNode := g.Nodes[ingredient]

		subRecipes := make([]*graph.Recipe, len(ingredientNode.RecipesToMakeThisElement))
		copy(subRecipes, ingredientNode.RecipesToMakeThisElement)

		if favorSimplicity {
			sort.Slice(subRecipes, func(i, j int) bool {
				iBaseCount := 0
				jBaseCount := 0

				for _, ing := range subRecipes[i].Ingredients {
					for _, base := range baseElements {
						if ing == base {
							iBaseCount++
							break
						}
					}
				}

				for _, ing := range subRecipes[j].Ingredients {
					for _, base := range baseElements {
						if ing == base {
							jBaseCount++
							break
						}
					}
				}

				if iBaseCount > 0 && jBaseCount > 0 {
					return iBaseCount > jBaseCount
				}

				return len(subRecipes[i].Ingredients) < len(subRecipes[j].Ingredients)
			})
		}

		for _, subRecipe := range subRecipes {
			ingredientPath := make([]*model.Node, len(newPath))
			copy(ingredientPath, newPath)

			exploreWithStrategy(g, subRecipe, ingredientPath, visited, visitCount, results, maxResults, baseElements, maxDepth, favorSimplicity)

			if len(*results) >= maxResults && maxResults > 0 {
				break
			}
		}

		// Backtrack
		delete(visited, ingredient)
	}
}
