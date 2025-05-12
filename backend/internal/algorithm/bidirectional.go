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

type PathSegment struct {
	Path     []model.Node
	LastElem string
}

func BidirectionalBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	log.Printf("DEBUG: Starting Bidirectional BFS for target: %s (max results: %d)", target, maxResults)

	g := graph.NewElementGraph(elements)

	if _, exists := g.Nodes[target]; !exists {
		log.Printf("DEBUG: Target element '%s' not found in element graph", target)
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target '%s' is a base element, no search needed", target)
			return [][]model.Node{}, 0
		}
	}

	// Track all recipes for the target element to ensure we find paths for each
	targetRecipes := make(map[string][]string)
	if targetNode := g.Nodes[target]; targetNode != nil {
		for _, recipe := range targetNode.RecipesToMakeThisElement {
			if len(recipe.Ingredients) == 2 {
				// Create recipe key for tracking
				sortedIngs := make([]string, len(recipe.Ingredients))
				copy(sortedIngs, recipe.Ingredients)
				sort.Strings(sortedIngs)
				recipeKey := strings.Join(sortedIngs, "+")
				targetRecipes[recipeKey] = recipe.Ingredients
			}
		}
	}
	log.Printf("DEBUG: Found %d valid recipes for target '%s'", len(targetRecipes), target)

	forwardFrontier := make([]PathSegment, 0)
	backwardFrontier := make([]PathSegment, 0)
	forwardVisited := make(map[string][]model.Node)
	backwardVisited := make(map[string][]model.Node)
	visitedCount := 0
	var results [][]model.Node

	// Recipe-indexed results to track unique recipe paths
	recipeResults := make(map[string][][]model.Node)

	for _, baseElem := range baseElements {
		imgPath := ""
		if elemData, exists := elements[baseElem]; exists {
			imgPath = elemData.ImagePath
		}

		node := model.Node{
			Element:     baseElem,
			ImagePath:   imgPath,
			Ingredients: nil,
		}

		path := []model.Node{node}
		forwardFrontier = append(forwardFrontier, PathSegment{
			Path:     path,
			LastElem: baseElem,
		})

		forwardVisited[baseElem] = path
		visitedCount++
	}

	// Initialize backward frontier for each recipe
	for _, recipe := range targetRecipes {
		imgPath := ""
		if elemData, exists := elements[target]; exists {
			imgPath = elemData.ImagePath
		}

		targetNode := model.Node{
			Element:     target,
			ImagePath:   imgPath,
			Ingredients: recipe,
		}

		backwardPath := []model.Node{targetNode}
		backwardFrontier = append(backwardFrontier, PathSegment{
			Path:     backwardPath,
			LastElem: target,
		})
	}

	if len(backwardFrontier) == 0 {
		// Fallback if no recipes were found
		imgPath := ""
		var targetIngredients []string
		if elemData, exists := elements[target]; exists {
			imgPath = elemData.ImagePath
			if targetNode := g.Nodes[target]; targetNode != nil && len(targetNode.RecipesToMakeThisElement) > 0 {
				targetIngredients = targetNode.RecipesToMakeThisElement[0].Ingredients
			}
		}

		targetNode := model.Node{
			Element:     target,
			ImagePath:   imgPath,
			Ingredients: targetIngredients,
		}

		backwardPath := []model.Node{targetNode}
		backwardFrontier = append(backwardFrontier, PathSegment{
			Path:     backwardPath,
			LastElem: target,
		})
	}

	backwardVisited[target] = backwardFrontier[0].Path
	visitedCount++

	// Configure search to continue longer when recipe diversity is needed
	maxIterations := 50
	maxIterationsWithoutProgress := 10
	iterationsWithoutProgress := 0
	lastResultCount := 0

	for i := 0; i < maxIterations; i++ {
		log.Printf("DEBUG: Bidirectional search iteration %d: Forward frontier: %d, Backward frontier: %d, Recipes found: %d",
			i, len(forwardFrontier), len(backwardFrontier), len(recipeResults))

		if len(forwardFrontier) == 0 && len(backwardFrontier) == 0 {
			log.Printf("DEBUG: Both frontiers empty, stopping search")
			break
		}

		// Track starting recipe count for this iteration
		startingRecipeCount := len(recipeResults)

		if len(forwardFrontier) > 0 {
			newConnections := expandForwardFrontier(
				&forwardFrontier,
				forwardVisited,
				backwardVisited,
				&results,
				elements,
				g,
				&visitedCount,
			)

			// Process newly found paths to group by recipe
			for _, path := range results[lastResultCount:] {
				if len(path) > 0 {
					// Extract recipe from the path
					var targetNode *model.Node
					for i := range path {
						if path[i].Element == target {
							targetNode = &path[i]
							break
						}
					}

					if targetNode != nil && targetNode.Ingredients != nil && len(targetNode.Ingredients) >= 2 {
						// Create recipe key
						sortedIngs := make([]string, len(targetNode.Ingredients))
						copy(sortedIngs, targetNode.Ingredients)
						sort.Strings(sortedIngs)
						recipeKey := strings.Join(sortedIngs, "+")

						recipeResults[recipeKey] = append(recipeResults[recipeKey], path)
					}
				}
			}
			lastResultCount = len(results)

			// In single path mode, stop after finding any path
			if newConnections > 0 && singlePath {
				log.Printf("DEBUG: Found a path in single path mode, stopping search")
				break
			}
		}

		// For multi-recipe searches, check if we have enough recipes
		if !singlePath && len(targetRecipes) > 1 && len(recipeResults) >= len(targetRecipes) &&
			len(recipeResults) >= maxResults/2 {
			log.Printf("DEBUG: Found paths for most recipes (%d/%d), may stop search early",
				len(recipeResults), len(targetRecipes))

			allRecipesHavePaths := true
			for recipeKey := range targetRecipes {
				if len(recipeResults[recipeKey]) == 0 {
					allRecipesHavePaths = false
					break
				}
			}

			// If we have at least one path for each recipe, consider stopping
			if allRecipesHavePaths && len(results) >= maxResults {
				log.Printf("DEBUG: Found at least one path for every recipe and %d total paths, stopping search", len(results))
				break
			}
		}

		if len(backwardFrontier) > 0 {
			newConnections := expandBackwardFrontier(
				&backwardFrontier,
				backwardVisited,
				forwardVisited,
				&results,
				elements,
				g,
				&visitedCount,
				baseElements,
			)

			// Process newly found paths to group by recipe
			for _, path := range results[lastResultCount:] {
				if len(path) > 0 {
					// Extract recipe from the path
					var targetNode *model.Node
					for i := range path {
						if path[i].Element == target {
							targetNode = &path[i]
							break
						}
					}

					if targetNode != nil && targetNode.Ingredients != nil && len(targetNode.Ingredients) >= 2 {
						// Create recipe key
						sortedIngs := make([]string, len(targetNode.Ingredients))
						copy(sortedIngs, targetNode.Ingredients)
						sort.Strings(sortedIngs)
						recipeKey := strings.Join(sortedIngs, "+")

						recipeResults[recipeKey] = append(recipeResults[recipeKey], path)
					}
				}
			}
			lastResultCount = len(results)

			if newConnections > 0 && singlePath {
				log.Printf("DEBUG: Found a path in single path mode, stopping search")
				break
			}
		}

		// Check if we made progress finding new recipes in this iteration
		if len(recipeResults) > startingRecipeCount {
			iterationsWithoutProgress = 0
		} else {
			iterationsWithoutProgress++
		}

		// Stop if we've explored enough without finding new recipes
		if iterationsWithoutProgress > maxIterationsWithoutProgress && len(recipeResults) > 0 {
			log.Printf("DEBUG: No new recipes found in %d iterations, stopping search",
				maxIterationsWithoutProgress)
			break
		}

		// Exit if we've found enough paths overall
		if len(results) >= maxResults*2 && !singlePath {
			log.Printf("DEBUG: Found %d paths, stopping bidirectional search", len(results))
			break
		}
	}

	log.Printf("DEBUG: Found paths for %d different recipes", len(recipeResults))
	for recipeKey, paths := range recipeResults {
		log.Printf("DEBUG: Recipe '%s': %d paths", recipeKey, len(paths))
	}

	var validResults [][]model.Node
	for _, path := range results {
		fixedPath := postProcessPath(path, elements, g)
		if validateIngredientsInPath(fixedPath) {
			validResults = append(validResults, fixedPath)
		}
	}

	// If we didn't find enough paths, try custom bidirectional search for each recipe
	if len(validResults) < maxResults && len(targetRecipes) > 0 {
		log.Printf("DEBUG: Standard bidirectional search found only %d valid paths, trying targeted approach", len(validResults))

		// Try each recipe using custom bidirectional search
		for _, ingredients := range targetRecipes {
			recipeKey := getRecipeKey(ingredients)
			// Skip if we already have paths for this recipe
			if len(recipeResults[recipeKey]) >= 2 {
				continue
			}

			log.Printf("DEBUG: Trying custom bidirectional search for recipe: %v", ingredients)
			customResults, customVisited := customBidirectionalSearch(
				elements,
				g,
				target,
				ingredients,
				maxResults/2,
				false,
			)

			for _, path := range customResults {
				if validateIngredientsInPath(path) {
					validResults = append(validResults, path)
				}
			}

			visitedCount += customVisited

			if len(validResults) >= maxResults && !singlePath {
				break
			}
		}
	}

	// Ensure recipe diversity by selecting some paths from each recipe
	if !singlePath && len(recipeResults) > 1 {
		var diverseResults [][]model.Node

		// First, get at least one path from each recipe
		for _, paths := range recipeResults {
			if len(paths) > 0 {
				// Find the shortest valid path for this recipe
				var bestPath []model.Node
				for _, path := range paths {
					if validateIngredientsInPath(path) {
						if bestPath == nil || len(path) < len(bestPath) {
							bestPath = path
						}
					}
				}

				if bestPath != nil {
					diverseResults = append(diverseResults, bestPath)
				}
			}
		}

		// Then add more paths until we reach maxResults
		remainingSlots := maxResults - len(diverseResults)
		if remainingSlots > 0 {
			// Merge all remaining valid paths and sort by length
			var remainingPaths [][]model.Node
			for _, paths := range recipeResults {
				for _, path := range paths {
					// Skip paths we already included
					alreadyIncluded := false
					for _, included := range diverseResults {
						if pathsEqual(path, included) {
							alreadyIncluded = true
							break
						}
					}

					if !alreadyIncluded && validateIngredientsInPath(path) {
						remainingPaths = append(remainingPaths, path)
					}
				}
			}

			// Sort remaining paths by length
			sort.Slice(remainingPaths, func(i, j int) bool {
				return len(remainingPaths[i]) < len(remainingPaths[j])
			})

			// Add shortest remaining paths
			for i := 0; i < len(remainingPaths) && i < remainingSlots; i++ {
				diverseResults = append(diverseResults, remainingPaths[i])
			}
		}

		if len(diverseResults) > 0 {
			log.Printf("DEBUG: Created diverse result set with %d paths from %d recipes",
				len(diverseResults), len(recipeResults))
			validResults = diverseResults
		}
	} else {
		// Sort by path length for single recipe results
		sort.Slice(validResults, func(i, j int) bool {
			return len(validResults[i]) < len(validResults[j])
		})
	}

	// Limit results to maxResults
	if len(validResults) > maxResults {
		validResults = validResults[:maxResults]
	}

	log.Printf("DEBUG: Bidirectional search complete, found %d paths after visiting %d nodes",
		len(validResults), visitedCount)

	return validResults, visitedCount
}

// Helper function to create a standardized recipe key
func getRecipeKey(ingredients []string) string {
	if len(ingredients) == 0 {
		return ""
	}
	sortedIngs := make([]string, len(ingredients))
	copy(sortedIngs, ingredients)
	sort.Strings(sortedIngs)
	return strings.Join(sortedIngs, "+")
}

func postProcessPath(path []model.Node, elements map[string]model.Element, g *graph.ElementGraph) []model.Node {
	if len(path) <= 1 {
		return path
	}

	result := make([]model.Node, len(path))
	copy(result, path)

	for i := 1; i < len(result); i++ {
		node := &result[i]

		// Skip base elements
		if utils.IsBaseElementName(node.Element, []string{"Water", "Fire", "Earth", "Air"}) {
			continue
		}

		if node.Ingredients == nil || len(node.Ingredients) == 0 {
			graphNode := g.Nodes[node.Element]
			if graphNode != nil && len(graphNode.RecipesToMakeThisElement) > 0 {
				for _, recipe := range graphNode.RecipesToMakeThisElement {
					if len(recipe.Ingredients) != 2 {
						continue
					}

					ingredient1Found := false
					ingredient2Found := false

					for j := 0; j < i; j++ {
						if result[j].Element == recipe.Ingredients[0] {
							ingredient1Found = true
						}
						if result[j].Element == recipe.Ingredients[1] {
							ingredient2Found = true
						}
					}

					if ingredient1Found && ingredient2Found {
						node.Ingredients = recipe.Ingredients
						break
					}
				}
			}
		}

		if node.ImagePath == "" {
			if elemData, exists := elements[node.Element]; exists {
				node.ImagePath = elemData.ImagePath
			}
		}
	}

	return result
}

func expandForwardFrontier(frontier *[]PathSegment, visited map[string][]model.Node, otherVisited map[string][]model.Node, results *[][]model.Node, elements map[string]model.Element, g *graph.ElementGraph, visitedCount *int) int {
	if len(*frontier) == 0 {
		return 0
	}

	var nextFrontier []PathSegment
	connectionsFound := 0
	currentLevel := *frontier
	*frontier = nil

	// Sort frontier by path length to prioritize shorter paths
	sort.Slice(currentLevel, func(i, j int) bool {
		return len(currentLevel[i].Path) < len(currentLevel[j].Path)
	})

	// Take the top N paths to avoid explosion
	maxPaths := 100
	if len(currentLevel) > maxPaths {
		currentLevel = currentLevel[:maxPaths]
	}

	for _, segment := range currentLevel {
		currentElem := segment.LastElem
		currentPath := segment.Path

		node := g.Nodes[currentElem]
		if node == nil {
			continue
		}

		for _, recipe := range node.RecipesMakingOtherElements {
			if len(recipe.Ingredients) != 2 {
				continue
			}
			var otherIngredient string
			if recipe.Ingredients[0] == currentElem {
				otherIngredient = recipe.Ingredients[1]
			} else {
				otherIngredient = recipe.Ingredients[0]
			}
			otherPath, otherIngredientFound := visited[otherIngredient]
			if !otherIngredientFound {
				continue
			}
			resultElem := recipe.Result
			if _, alreadyVisited := visited[resultElem]; alreadyVisited {
				continue
			}
			resultImgPath := ""
			var ingredients []string
			if elemData, exists := elements[resultElem]; exists {
				resultImgPath = elemData.ImagePath
				ingredients = recipe.Ingredients
			}

			resultNode := model.Node{
				Element:     resultElem,
				ImagePath:   resultImgPath,
				Ingredients: ingredients,
			}

			newPath := make([]model.Node, len(currentPath))
			copy(newPath, currentPath)

			ensureIngredientInPath(&newPath, otherPath, otherIngredient)

			newPath = append(newPath, resultNode)

			visited[resultElem] = newPath
			*visitedCount++

			nextFrontier = append(nextFrontier, PathSegment{
				Path:     newPath,
				LastElem: resultElem,
			})

			if backwardPath, found := otherVisited[resultElem]; found {
				completePath := mergePaths(newPath, backwardPath)

				if validateIngredientsInPath(completePath) {
					if !containsPath(*results, completePath) {
						*results = append(*results, completePath)
						connectionsFound++
					}
				}
			}
		}
	}
	*frontier = nextFrontier
	return connectionsFound
}

func ensureIngredientInPath(path *[]model.Node, ingredientPath []model.Node, ingredient string) {
	for _, node := range *path {
		if node.Element == ingredient {
			return
		}
	}

	var ingredientNode model.Node
	for _, node := range ingredientPath {
		if node.Element == ingredient {
			ingredientNode = node
			break
		}
	}

	newPath := []model.Node{ingredientNode}
	newPath = append(newPath, *path...)
	*path = newPath
}

func expandBackwardFrontier(frontier *[]PathSegment, visited map[string][]model.Node, otherVisited map[string][]model.Node, results *[][]model.Node, elements map[string]model.Element, g *graph.ElementGraph, visitedCount *int, baseElements []string) int {
	if len(*frontier) == 0 {
		return 0
	}

	var nextFrontier []PathSegment
	connectionsFound := 0
	currentLevel := *frontier
	*frontier = nil

	for _, segment := range currentLevel {
		currentElem := segment.LastElem
		currentPath := segment.Path

		node := g.Nodes[currentElem]
		if node == nil {
			continue
		}

		for _, recipe := range node.RecipesToMakeThisElement {
			if len(recipe.Ingredients) != 2 {
				continue
			}

			for _, ingredient := range recipe.Ingredients {
				if _, alreadyVisited := visited[ingredient]; alreadyVisited {
					continue
				}

				imgPath := ""
				if elemData, exists := elements[ingredient]; exists {
					imgPath = elemData.ImagePath
				}

				ingredientNode := model.Node{
					Element:     ingredient,
					ImagePath:   imgPath,
					Ingredients: nil,
				}

				newPath := []model.Node{ingredientNode}
				newPath = append(newPath, currentPath...)

				visited[ingredient] = newPath
				*visitedCount++

				nextFrontier = append(nextFrontier, PathSegment{
					Path:     newPath,
					LastElem: ingredient,
				})

				isBase := utils.IsBaseElementName(ingredient, baseElements)

				if isBase || otherVisited[ingredient] != nil {
					forwardPath := otherVisited[ingredient]

					if forwardPath == nil && isBase {
						forwardPath = []model.Node{{
							Element:     ingredient,
							ImagePath:   imgPath,
							Ingredients: nil,
						}}
					}

					if forwardPath != nil {
						completePath := mergePaths(forwardPath, newPath)

						// Avoid duplicates
						if !containsPath(*results, completePath) {
							*results = append(*results, completePath)
							connectionsFound++
						}
					}
				}
			}
		}
	}

	*frontier = nextFrontier
	return connectionsFound
}

func mergePaths(forwardPath, backwardPath []model.Node) []model.Node {
	meetingElem := forwardPath[len(forwardPath)-1].Element

	meetingIdx := -1
	for i, node := range backwardPath {
		if node.Element == meetingElem {
			meetingIdx = i
			break
		}
	}

	if meetingIdx == -1 {
		// If meeting point not found in backward path, return forward path
		return forwardPath
	}

	// Create result with forward path elements
	result := make([]model.Node, len(forwardPath))
	copy(result, forwardPath)

	// Preserve ingredients from the meeting point in backward path if available
	if len(backwardPath[meetingIdx].Ingredients) > 0 &&
		(result[len(result)-1].Ingredients == nil || len(result[len(result)-1].Ingredients) == 0) {
		result[len(result)-1].Ingredients = backwardPath[meetingIdx].Ingredients
	}

	// Append remaining elements from backward path after the meeting point
	if len(backwardPath) > meetingIdx+1 {
		result = append(result, backwardPath[meetingIdx+1:]...)
	}

	return result
}

func containsPath(paths [][]model.Node, newPath []model.Node) bool {
	for _, existingPath := range paths {
		if pathsEqual(existingPath, newPath) {
			return true
		}
	}
	return false
}

func pathsEqual(path1, path2 []model.Node) bool {
	if len(path1) != len(path2) {
		return false
	}

	for i := range path1 {
		if path1[i].Element != path2[i].Element {
			return false
		}
	}

	return true
}

func MultiThreadedBidirectionalBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	log.Printf("DEBUG: Starting Multi-threaded Bidirectional BFS for target: %s", target)

	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		log.Printf("DEBUG: Target element '%s' not found", target)
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target is base element, returning empty result")
			return [][]model.Node{}, 0
		}
	}

	recipes := targetNode.RecipesToMakeThisElement
	if len(recipes) == 0 {
		log.Printf("DEBUG: No recipes found to make target '%s'", target)
		return [][]model.Node{}, 0
	}

	log.Printf("DEBUG: Found %d recipes to make target '%s'", len(recipes), target)

	// Filter to valid recipes with exactly 2 ingredients
	var validRecipes []*graph.Recipe
	for _, recipe := range recipes {
		if len(recipe.Ingredients) == 2 {
			// Check if both ingredients are traceable to base elements
			allIngredientsTraceable := true
			for _, ing := range recipe.Ingredients {
				// Skip checking base elements
				isBaseElement := false
				for _, base := range baseElements {
					if ing == base {
						isBaseElement = true
						break
					}
				}

				// If not base element, check traceability
				if !isBaseElement && !IsElementTraceable(ing, baseElements, g) {
					log.Printf("DEBUG: Recipe ingredient '%s' for target '%s' is not traceable to base elements", ing, target)
					allIngredientsTraceable = false
					break
				}
			}

			if allIngredientsTraceable {
				validRecipes = append(validRecipes, recipe)
			} else {
				log.Printf("DEBUG: Skipping recipe for '%s' with untraceable ingredients: %v", target, recipe.Ingredients)
			}
		}
	}

	if len(validRecipes) == 0 {
		log.Printf("DEBUG: No valid traceable recipes found for '%s'", target)
		return [][]model.Node{}, 0
	}

	log.Printf("DEBUG: Processing %d valid traceable recipes", len(validRecipes))

	// Rest of the function remains the same
	var totalVisits int
	var mu sync.Mutex
	pathChan := make(chan []model.Node, maxResults*len(validRecipes)*2)
	maxConcurrency := 8
	if len(validRecipes) < maxConcurrency {
		maxConcurrency = len(validRecipes)
	}

	// Use worker pool pattern for better resource management
	recipeChan := make(chan *graph.Recipe, len(validRecipes))

	// Feed recipes to channel
	for _, recipe := range validRecipes {
		recipeChan <- recipe
	}
	close(recipeChan)

	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for recipe := range recipeChan {
				imgPathTarget := ""
				if elemData, exists := elements[target]; exists {
					imgPathTarget = elemData.ImagePath
				}

				targetNode := model.Node{
					Element:     target,
					ImagePath:   imgPathTarget,
					Ingredients: recipe.Ingredients,
				}

				ingredient1 := recipe.Ingredients[0]
				ingredient2 := recipe.Ingredients[1]

				ingredient1IsBase := utils.IsBaseElementName(ingredient1, baseElements)
				ingredient2IsBase := utils.IsBaseElementName(ingredient2, baseElements)

				var paths1, paths2 [][]model.Node
				var visited1, visited2 int

				if ingredient1IsBase {
					imgPath1 := ""
					if elemData, exists := elements[ingredient1]; exists {
						imgPath1 = elemData.ImagePath
					}

					paths1 = [][]model.Node{{
						{Element: ingredient1, ImagePath: imgPath1, Ingredients: nil},
					}}
				} else {
					// Try harder to find paths for non-base ingredients
					pathsPerIngredient := 3
					paths1, visited1 = customBidirectionalSearch(
						elements,
						g,
						ingredient1,
						nil,
						pathsPerIngredient,
						false,
					)

					// If that fails, try fallback approach
					if len(paths1) == 0 {
						log.Printf("DEBUG: Trying harder to find paths for ingredient: %s", ingredient1)
						paths1, visited1 = BidirectionalBFS(elements, ingredient1, pathsPerIngredient, false)
					}

					mu.Lock()
					totalVisits += visited1
					mu.Unlock()
				}

				if ingredient2IsBase {
					imgPath2 := ""
					if elemData, exists := elements[ingredient2]; exists {
						imgPath2 = elemData.ImagePath
					}

					paths2 = [][]model.Node{{
						{Element: ingredient2, ImagePath: imgPath2, Ingredients: nil},
					}}
				} else {
					// Try harder to find paths for non-base ingredients
					pathsPerIngredient := 3
					paths2, visited2 = customBidirectionalSearch(
						elements,
						g,
						ingredient2,
						nil,
						pathsPerIngredient,
						false,
					)

					// If that fails, try fallback approach
					if len(paths2) == 0 {
						log.Printf("DEBUG: Trying harder to find paths for ingredient: %s", ingredient2)
						paths2, visited2 = BidirectionalBFS(elements, ingredient2, pathsPerIngredient, false)
					}

					mu.Lock()
					totalVisits += visited2
					mu.Unlock()
				}

				log.Printf("DEBUG: For recipe [%s + %s], found %d and %d paths for ingredients",
					ingredient1, ingredient2, len(paths1), len(paths2))

				if len(paths1) > 0 && len(paths2) > 0 {
					// Limit path combinations to avoid explosion
					maxPathsPerIngredient := 5
					if len(paths1) > maxPathsPerIngredient {
						paths1 = paths1[:maxPathsPerIngredient]
					}
					if len(paths2) > maxPathsPerIngredient {
						paths2 = paths2[:maxPathsPerIngredient]
					}

					for _, path1 := range paths1 {
						for _, path2 := range paths2 {
							combinedPath := make([]model.Node, len(path1))
							copy(combinedPath, path1)

							// Add nodes from path2 that aren't already in the combined path
							for _, node := range path2 {
								exists := false
								for _, existingNode := range combinedPath {
									if existingNode.Element == node.Element {
										exists = true
										break
									}
								}
								if !exists {
									combinedPath = append(combinedPath, node)
								}
							}

							finalPath := append(combinedPath, targetNode)

							if validateIngredientsInPath(finalPath) {
								pathChan <- finalPath
							}
						}
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(pathChan)
	}()

	var allPaths [][]model.Node
	distinctRecipes := make(map[string][]model.Node)

	for path := range pathChan {
		if len(path) > 0 {
			targetNode := path[len(path)-1]
			if targetNode.Element == target && targetNode.Ingredients != nil && len(targetNode.Ingredients) == 2 {
				// Create recipe key
				ingredients := make([]string, len(targetNode.Ingredients))
				copy(ingredients, targetNode.Ingredients)
				sort.Strings(ingredients)
				recipeKey := strings.Join(ingredients, "+")

				// For each recipe, keep the shortest path found
				existingPath, hasPath := distinctRecipes[recipeKey]
				if !hasPath || len(path) < len(existingPath) {
					distinctRecipes[recipeKey] = path
				}
			}
		}
	}

	// Collect all paths, prioritizing recipe diversity
	for _, path := range distinctRecipes {
		allPaths = append(allPaths, path)
	}

	// If we didn't get enough diverse recipes, try standard approach for any missing ones
	if len(allPaths) < len(validRecipes) {
		log.Printf("DEBUG: Multi-threaded approach found paths for only %d/%d recipes, trying standard approach",
			len(allPaths), len(validRecipes))

		// Find which recipes we're missing
		foundRecipes := make(map[string]bool)
		for _, path := range allPaths {
			if targetNode := path[len(path)-1]; targetNode.Ingredients != nil {
				foundRecipes[getRecipeKey(targetNode.Ingredients)] = true
			}
		}

		// Try standard bidirectional search for each missing recipe
		for _, recipe := range validRecipes {
			recipeKey := getRecipeKey(recipe.Ingredients)
			if !foundRecipes[recipeKey] {
				log.Printf("DEBUG: Trying standard approach for missing recipe: %v", recipe.Ingredients)
				customResults, customVisited := customBidirectionalSearch(
					elements,
					g,
					target,
					recipe.Ingredients,
					2,
					false,
				)

				totalVisits += customVisited

				if len(customResults) > 0 {
					// Add the shortest valid path
					sort.Slice(customResults, func(i, j int) bool {
						return len(customResults[i]) < len(customResults[j])
					})

					for _, path := range customResults {
						if validateIngredientsInPath(path) {
							allPaths = append(allPaths, path)
							foundRecipes[recipeKey] = true
							break
						}
					}
				}
			}
		}
	}

	// Sort paths by length
	sort.Slice(allPaths, func(i, j int) bool {
		return len(allPaths[i]) < len(allPaths[j])
	})

	// Limit results to maxResults
	if len(allPaths) > maxResults {
		allPaths = allPaths[:maxResults]
	}

	log.Printf("DEBUG: Multi-threaded bidirectional BFS complete. Found %d paths with %d distinct recipes, visited %d nodes",
		len(allPaths), len(distinctRecipes), totalVisits)

	return allPaths, totalVisits
}

func customBidirectionalSearch(elements map[string]model.Element, g *graph.ElementGraph, target string, ingredients []string, maxResults int, singlePath bool) ([][]model.Node, int) {
	var results [][]model.Node
	visitedCount := 0
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	forwardVisited := make(map[string][]model.Node)
	forwardFrontier := make([]PathSegment, 0)

	for _, baseElem := range baseElements {
		imgPath := ""
		if elemData, exists := elements[baseElem]; exists {
			imgPath = elemData.ImagePath
		}

		node := model.Node{
			Element:     baseElem,
			ImagePath:   imgPath,
			Ingredients: nil,
		}

		path := []model.Node{node}
		forwardFrontier = append(forwardFrontier, PathSegment{
			Path:     path,
			LastElem: baseElem,
		})

		forwardVisited[baseElem] = path
		visitedCount++
	}

	backwardVisited := make(map[string][]model.Node)
	backwardFrontier := make([]PathSegment, 0)

	imgPath := ""
	if elemData, exists := elements[target]; exists {
		imgPath = elemData.ImagePath
	}

	targetNode := model.Node{
		Element:     target,
		ImagePath:   imgPath,
		Ingredients: ingredients,
	}

	backwardPath := []model.Node{targetNode}
	backwardFrontier = append(backwardFrontier, PathSegment{
		Path:     backwardPath,
		LastElem: target,
	})

	backwardVisited[target] = backwardPath
	visitedCount++

	maxIterations := 30
	for i := 0; i < maxIterations; i++ {
		if len(forwardFrontier) == 0 && len(backwardFrontier) == 0 {
			break
		}

		if len(forwardFrontier) > 0 {
			connectionsFound := expandForwardFrontierTargeted(
				&forwardFrontier,
				forwardVisited,
				backwardVisited,
				&results,
				elements,
				g,
				&visitedCount,
				ingredients,
			)

			if connectionsFound > 0 && singlePath {
				break
			}
		}
		checkForIngredientConnections(
			forwardVisited,
			target,
			ingredients,
			elements,
			&results,
		)
		if len(results) > 0 && singlePath {
			break
		}
		if len(backwardFrontier) > 0 {
			connectionsFound := expandBackwardFrontierTargeted(
				&backwardFrontier,
				backwardVisited,
				forwardVisited,
				&results,
				elements,
				g,
				&visitedCount,
				baseElements,
			)
			if connectionsFound > 0 && singlePath {
				break
			}
		}
		if len(results) >= maxResults && !singlePath {
			break
		}
	}
	var validResults [][]model.Node
	for _, path := range results {
		fixedPath := postProcessPath(path, elements, g)
		if validateIngredientsInPath(fixedPath) {
			validResults = append(validResults, fixedPath)
		}
	}
	sort.Slice(validResults, func(i, j int) bool {
		return len(validResults[i]) < len(validResults[j])
	})
	if len(validResults) > maxResults {
		validResults = validResults[:maxResults]
	}
	return validResults, visitedCount
}

func expandForwardFrontierTargeted(frontier *[]PathSegment, visited map[string][]model.Node, otherVisited map[string][]model.Node, results *[][]model.Node, elements map[string]model.Element, g *graph.ElementGraph, visitedCount *int, targetIngredients []string) int {
	if len(*frontier) == 0 {
		return 0
	}

	var nextFrontier []PathSegment
	connectionsFound := 0
	currentLevel := *frontier
	*frontier = nil

	for _, segment := range currentLevel {
		currentElem := segment.LastElem
		currentPath := segment.Path

		node := g.Nodes[currentElem]
		if node == nil {
			continue
		}

		for _, recipe := range node.RecipesMakingOtherElements {
			if len(recipe.Ingredients) != 2 {
				continue
			}

			var otherIngredient string
			if recipe.Ingredients[0] == currentElem {
				otherIngredient = recipe.Ingredients[1]
			} else {
				otherIngredient = recipe.Ingredients[0]
			}

			_, seen := visited[otherIngredient]
			if !seen {
				continue
			}

			resultElem := recipe.Result

			if _, alreadyVisited := visited[resultElem]; alreadyVisited {
				continue
			}

			resultImgPath := ""
			if elemData, exists := elements[resultElem]; exists {
				resultImgPath = elemData.ImagePath
			}

			resultNode := model.Node{
				Element:     resultElem,
				ImagePath:   resultImgPath,
				Ingredients: recipe.Ingredients,
			}

			newPath := make([]model.Node, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, resultNode)

			visited[resultElem] = newPath
			*visitedCount++

			isTargetIngredient := false
			for _, target := range targetIngredients {
				if resultElem == target {
					isTargetIngredient = true
					break
				}
			}

			if isTargetIngredient {
				nextFrontier = append([]PathSegment{{
					Path:     newPath,
					LastElem: resultElem,
				}}, nextFrontier...)
			} else {
				nextFrontier = append(nextFrontier, PathSegment{
					Path:     newPath,
					LastElem: resultElem,
				})
			}

			if backwardPath, found := otherVisited[resultElem]; found {
				completePath := mergePaths(newPath, backwardPath)

				// Avoid duplicates
				if !containsPath(*results, completePath) {
					*results = append(*results, completePath)
					connectionsFound++
				}
			}
		}
	}

	*frontier = nextFrontier
	return connectionsFound
}

func expandBackwardFrontierTargeted(frontier *[]PathSegment, visited map[string][]model.Node, otherVisited map[string][]model.Node, results *[][]model.Node, elements map[string]model.Element, g *graph.ElementGraph, visitedCount *int, baseElements []string) int {
	if len(*frontier) == 0 {
		return 0
	}

	var nextFrontier []PathSegment
	connectionsFound := 0
	currentLevel := *frontier
	*frontier = nil

	for _, segment := range currentLevel {
		currentElem := segment.LastElem
		currentPath := segment.Path

		node := g.Nodes[currentElem]
		if node == nil {
			continue
		}

		for _, recipe := range node.RecipesToMakeThisElement {
			if len(recipe.Ingredients) != 2 {
				continue
			}

			for _, ingredient := range recipe.Ingredients {
				if _, alreadyVisited := visited[ingredient]; alreadyVisited {
					continue
				}

				isBase := utils.IsBaseElementName(ingredient, baseElements)

				imgPath := ""
				if elemData, exists := elements[ingredient]; exists {
					imgPath = elemData.ImagePath
				}

				ingredientNode := model.Node{
					Element:     ingredient,
					ImagePath:   imgPath,
					Ingredients: nil,
				}

				newPath := []model.Node{ingredientNode}
				newPath = append(newPath, currentPath...)

				visited[ingredient] = newPath
				*visitedCount++

				if isBase {
					nextFrontier = append([]PathSegment{{
						Path:     newPath,
						LastElem: ingredient,
					}}, nextFrontier...)
				} else {
					nextFrontier = append(nextFrontier, PathSegment{
						Path:     newPath,
						LastElem: ingredient,
					})
				}

				if forwardPath, found := otherVisited[ingredient]; found {
					completePath := mergePaths(forwardPath, newPath)

					if !containsPath(*results, completePath) {
						*results = append(*results, completePath)
						connectionsFound++
					}
				} else if isBase {
					basePath := []model.Node{{
						Element:     ingredient,
						ImagePath:   imgPath,
						Ingredients: nil,
					}}

					completePath := mergePaths(basePath, newPath)

					if !containsPath(*results, completePath) {
						*results = append(*results, completePath)
						connectionsFound++
					}
				}
			}
		}
	}

	*frontier = nextFrontier
	return connectionsFound
}

func checkForIngredientConnections(forwardVisited map[string][]model.Node, target string, targetIngredients []string, elements map[string]model.Element, results *[][]model.Node) bool {
	foundConnections := false

	ingredientPaths := make([][]model.Node, 0, len(targetIngredients))
	ingredientsFound := 0

	for _, ingredient := range targetIngredients {
		if path, found := forwardVisited[ingredient]; found {
			ingredientPaths = append(ingredientPaths, path)
			ingredientsFound++
		}
	}

	if ingredientsFound == len(targetIngredients) && len(targetIngredients) > 0 {
		imgPath := ""
		if elemData, exists := elements[target]; exists {
			imgPath = elemData.ImagePath
		}

		targetNode := model.Node{
			Element:     target,
			ImagePath:   imgPath,
			Ingredients: targetIngredients,
		}

		for _, path := range ingredientPaths {
			newPath := make([]model.Node, len(path))
			copy(newPath, path)
			newPath = append(newPath, targetNode)

			if !containsPath(*results, newPath) {
				*results = append(*results, newPath)
				foundConnections = true
			}
		}
	}

	return foundConnections
}

func HybridSearch(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	log.Printf("DEBUG: Starting hybrid search for target: %s", target)

	paths, visited := MultiThreadedBidirectionalBFS(elements, target, maxResults, singlePath)

	if len(paths) > 0 {
		return paths, visited
	}

	log.Printf("DEBUG: Multi-threaded bidirectional search failed, falling back to standard bidirectional BFS")
	return BidirectionalBFS(elements, target, maxResults, singlePath)
}

func ConcurrentElementSearch(elements map[string]model.Element, targets []string, maxResultsPerTarget int, singlePath bool) map[string][][]model.Node {
	log.Printf("DEBUG: Starting concurrent search for %d targets", len(targets))

	results := make(map[string][][]model.Node)
	resultsMutex := sync.Mutex{}

	maxConcurrency := 4
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(tgt string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			paths, _ := HybridSearch(elements, tgt, maxResultsPerTarget, singlePath)

			resultsMutex.Lock()
			results[tgt] = paths
			resultsMutex.Unlock()
		}(target)
	}

	wg.Wait()

	return results
}

func FindShortestPath(elements map[string]model.Element, target string) ([]model.Node, int) {
	paths, visited := BidirectionalBFS(elements, target, 1, true)

	if len(paths) > 0 {
		return paths[0], visited
	}

	return nil, visited
}

func AnalyzeElementComplexity(elements map[string]model.Element, elementName string) int {
	path, _ := FindShortestPath(elements, elementName)

	if path == nil {
		return -1
	}

	return len(path) - 1
}

func FindPrerequisiteElements(elements map[string]model.Element, target string) map[string]bool {
	paths, _ := BidirectionalBFS(elements, target, 5, false)
	prerequisites := make(map[string]bool)

	for _, path := range paths {
		for _, node := range path {
			prerequisites[node.Element] = true
		}
	}
	delete(prerequisites, target)

	return prerequisites
}

func validateIngredientsInPath(path []model.Node) bool {
	if len(path) <= 1 {
		return true
	}

	seenElements := make(map[string]bool)
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Add all base elements to the seen set initially
	for _, baseElem := range baseElements {
		seenElements[baseElem] = true
	}

	// Also mark all elements in path as seen
	for _, node := range path {
		seenElements[node.Element] = true
	}

	// Then verify that each non-base element has its ingredients earlier in the path
	for i := 1; i < len(path); i++ {
		currentNode := path[i]

		// Skip validation for base elements
		if utils.IsBaseElementName(currentNode.Element, baseElements) {
			continue
		}

		// For non-base elements, they must have ingredients defined
		if currentNode.Ingredients == nil || len(currentNode.Ingredients) == 0 {
			// Try to find ingredients among previous elements
			found := false
			for j := 0; j < i && !found; j++ {
				// If any previous element can be used as ingredient
				prevElem := path[j].Element
				// This is a simplified check - you might need more complex recipe validation
				if prevElem != currentNode.Element {
					found = true
				}
			}
			if !found {
				return false
			}
		} else {
			// Check all ingredients are available and traceable
			for _, ingredient := range currentNode.Ingredients {
				ingredientAvailable := false

				// Skip base elements - they're always available
				if utils.IsBaseElementName(ingredient, baseElements) {
					continue
				}

				// Check if this ingredient appears earlier in path
				for j := 0; j < i; j++ {
					if path[j].Element == ingredient {
						ingredientAvailable = true
						break
					}
				}

				if !ingredientAvailable && !seenElements[ingredient] {
					return false
				}
			}
		}
	}

	return true
}

// Add this function to fix the problem in HandleBidirectionalSearch
func IsRecipeTraceable(ingredients []string, baseElements []string, g *graph.ElementGraph) bool {
	// Check if both ingredients are traceable to base elements
	for _, ing := range ingredients {
		// Skip checking base elements
		isBaseElement := false
		for _, base := range baseElements {
			if ing == base {
				isBaseElement = true
				break
			}
		}

		// If not base element, check traceability
		if !isBaseElement && !IsElementTraceable(ing, baseElements, g) {
			return false
		}
	}
	return true
}
