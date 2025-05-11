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

	forwardFrontier := make([]PathSegment, 0)
	backwardFrontier := make([]PathSegment, 0)
	forwardVisited := make(map[string][]model.Node)
	backwardVisited := make(map[string][]model.Node)
	visitedCount := 0
	var results [][]model.Node

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

	backwardVisited[target] = backwardPath
	visitedCount++

	maxIterations := 50
	for i := 0; i < maxIterations; i++ {
		log.Printf("DEBUG: Bidirectional search iteration %d: Forward frontier: %d, Backward frontier: %d",
			i, len(forwardFrontier), len(backwardFrontier))

		if len(forwardFrontier) == 0 && len(backwardFrontier) == 0 {
			log.Printf("DEBUG: Both frontiers empty, stopping search")
			break
		}

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

			if newConnections > 0 && singlePath {
				log.Printf("DEBUG: Found a path in single path mode, stopping search")
				break
			}
		}

		if len(results) >= maxResults && !singlePath {
			log.Printf("DEBUG: Found %d paths, stopping bidirectional search", len(results))
			break
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

			if newConnections > 0 && singlePath {
				log.Printf("DEBUG: Found a path in single path mode, stopping search")
				break
			}
		}

		if len(results) >= maxResults && !singlePath {
			log.Printf("DEBUG: Found %d paths, stopping bidirectional search", len(results))
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

	if len(validResults) == 0 {
		log.Printf("DEBUG: Standard bidirectional search found no valid paths, trying custom approach")
		if targetNode := g.Nodes[target]; targetNode != nil && len(targetNode.RecipesToMakeThisElement) > 0 {
			for _, recipe := range targetNode.RecipesToMakeThisElement {
				if len(recipe.Ingredients) == 2 {
					customResults, customVisited := customBidirectionalSearch(
						elements,
						g,
						target,
						recipe.Ingredients,
						maxResults,
						singlePath,
					)

					for _, path := range customResults {
						if validateIngredientsInPath(path) {
							validResults = append(validResults, path)
						}
					}

					visitedCount += customVisited

					if len(validResults) > 0 && singlePath {
						break
					}
				}
			}
		}
	}

	sort.Slice(validResults, func(i, j int) bool {
		return len(validResults[i]) < len(validResults[j])
	})

	// Limit results to maxResults
	if len(validResults) > maxResults {
		validResults = validResults[:maxResults]
	}

	log.Printf("DEBUG: Bidirectional search complete, found %d paths after visiting %d nodes",
		len(validResults), visitedCount)

	return validResults, visitedCount
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
		return forwardPath
	}

	result := make([]model.Node, len(forwardPath))
	copy(result, forwardPath)

	if len(backwardPath[meetingIdx].Ingredients) > 0 && result[len(result)-1].Ingredients == nil {
		result[len(result)-1].Ingredients = backwardPath[meetingIdx].Ingredients
	}

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

	var totalVisits int
	var mu sync.Mutex
	pathChan := make(chan []model.Node, maxResults*10)
	maxConcurrency := 4
	if len(recipes) < maxConcurrency {
		maxConcurrency = len(recipes)
	}
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, recipe := range recipes {
		if len(recipe.Ingredients) != 2 {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(r *graph.Recipe) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			imgPathTarget := ""
			if elemData, exists := elements[target]; exists {
				imgPathTarget = elemData.ImagePath
			}

			targetNode := model.Node{
				Element:     target,
				ImagePath:   imgPathTarget,
				Ingredients: r.Ingredients,
			}

			ingredient1 := r.Ingredients[0]
			ingredient2 := r.Ingredients[1]

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
				paths1, visited1 = customBidirectionalSearch(
					elements,
					g,
					ingredient1,
					nil,
					2,
					false,
				)
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
				paths2, visited2 = customBidirectionalSearch(
					elements,
					g,
					ingredient2,
					nil,
					2,
					false,
				)
				mu.Lock()
				totalVisits += visited2
				mu.Unlock()
			}

			if len(paths1) > 0 && len(paths2) > 0 {
				for _, path1 := range paths1 {
					for _, path2 := range paths2 {
						combinedPath := make([]model.Node, len(path1))
						copy(combinedPath, path1)

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
		}(recipe)
	}

	go func() {
		wg.Wait()
		close(pathChan)
	}()

	var allPaths [][]model.Node
	distinctRecipes := make(map[string]bool)

	for path := range pathChan {
		if len(path) > 0 {
			targetNode := path[len(path)-1]
			if targetNode.Element == target && targetNode.Ingredients != nil && len(targetNode.Ingredients) == 2 {
				ingredients := make([]string, len(targetNode.Ingredients))
				copy(ingredients, targetNode.Ingredients)
				sort.Strings(ingredients)
				recipeKey := strings.Join(ingredients, "+")

				if !distinctRecipes[recipeKey] {
					distinctRecipes[recipeKey] = true
					allPaths = append(allPaths, path)
				}
			}
		}
	}

	if len(allPaths) == 0 {
		log.Printf("DEBUG: No paths found in multi-threaded search, trying pure bidirectional approach")
		for _, recipe := range recipes {
			if len(recipe.Ingredients) == 2 {
				customResults, customVisited := customBidirectionalSearch(
					elements,
					g,
					target,
					recipe.Ingredients,
					maxResults,
					singlePath,
				)

				totalVisits += customVisited

				for _, path := range customResults {
					if validateIngredientsInPath(path) {
						ingredients := make([]string, len(recipe.Ingredients))
						copy(ingredients, recipe.Ingredients)
						sort.Strings(ingredients)
						recipeKey := strings.Join(ingredients, "+")

						if !distinctRecipes[recipeKey] {
							distinctRecipes[recipeKey] = true
							allPaths = append(allPaths, path)
						}
					}
				}
			}
		}
	}
	sort.Slice(allPaths, func(i, j int) bool {
		return len(allPaths[i]) < len(allPaths[j])
	})
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
	seenElements[path[0].Element] = true

	for i := 1; i < len(path); i++ {
		currentNode := path[i]

		if i > 0 && (currentNode.Ingredients == nil || len(currentNode.Ingredients) == 0) {
			return false
		}

		if currentNode.Ingredients != nil && len(currentNode.Ingredients) > 0 {
			for _, ingredient := range currentNode.Ingredients {
				if !seenElements[ingredient] {
					return false
				}
			}
		}

		seenElements[currentNode.Element] = true
	}

	return true
}
