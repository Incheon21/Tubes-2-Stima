package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"backend/utils"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		fmt.Printf("Target element '%s' tidak exists di The Little Alchemist 2\n", target)
		return [][]model.Node{}, 0
	}

	baseElements := g.BaseElements
	fmt.Printf("Ditemukan base elemen: %v\n", baseElements)

	for _, base := range baseElements {
		if target == base {
			fmt.Printf("Target '%s' adalah base element\n", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	visited := make(map[string]bool)
	results := [][]model.Node{}
	nodesVisited := 0
	targetTier := elements[target].Tier

	currentLevel := []string{}
	levelMap := make(map[string]int)
	pathMap := make(map[string][]model.Node)

	// Start from base elements
	for _, elem := range baseElements {
		elemNode := g.Nodes[elem]
		currentLevel = append(currentLevel, elem)
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem, ImagePath: elemNode.ImagePath},
		}
		levelMap[elem] = 0
	}

	return recursiveBFS(g, currentLevel, 0, visited, levelMap, pathMap, &results, &nodesVisited, target, targetTier, maxResults, singlePath, baseElements, elements)
}

func recursiveBFS(g *graph.ElementGraph, currentLevel []string, level int, visited map[string]bool, levelMap map[string]int, pathMap map[string][]model.Node, results *[][]model.Node, nodesVisited *int, target string, targetTier int, maxResults int, singlePath bool, baseElements []string, elements map[string]model.Element,
) ([][]model.Node, int) {
	if len(currentLevel) == 0 || (singlePath && len(*results) > 0) {
		return *results, *nodesVisited
	}

	nextLevel := []string{}
	*nodesVisited += len(currentLevel)

	fmt.Printf("Processing level %d with %d elements\n", level, len(currentLevel))

	for _, current := range currentLevel {
		currentNode := g.Nodes[current]

		for _, recipe := range currentNode.RecipesToMakeOtherElement {
			resultElement := recipe.Result
			resultNode := g.Nodes[resultElement]

			if visited[resultElement] {
				continue
			}

			allIngredientsVisited := true
			for _, ingredient := range recipe.Ingredients {
				if !visited[ingredient] {
					allIngredientsVisited = false
					break
				}
			}

			allIngredientsHaveValidTier := true
			resultElementTier := elements[resultElement].Tier

			for _, ingredient := range recipe.Ingredients {
				ingredientTier := elements[ingredient].Tier
				if ingredientTier > resultElementTier {
					fmt.Printf("Skipping recipe for '%s': ingredient '%s' (tier %d) > result (tier %d)\n",
						resultElement, ingredient, ingredientTier, resultElementTier)
					allIngredientsHaveValidTier = false
					break
				}
			}

			if allIngredientsVisited && allIngredientsHaveValidTier {
				visited[resultElement] = true
				levelMap[resultElement] = level + 1

				path := buildPath(pathMap, recipe.Ingredients, resultElement, resultNode.ImagePath, g, baseElements)
				pathMap[resultElement] = path

				if resultElement == target {
					fmt.Printf("Ditemukan target '%s'! Path length: %d\n", target, len(path))

					validPath := true
					for _, node := range path {
						if node.Element == target {
							continue
						}

						if ingredient, exists := elements[node.Element]; exists {
							if ingredient.Tier > targetTier {
								validPath = false
								break
							}
						}
					}

					if validPath {
						*results = append(*results, path)

						fmt.Printf("Path to '%s': ", target)
						for i, node := range path {
							if i > 0 {
								fmt.Printf(" -> ")
							}
							fmt.Printf("%s", node.Element)
						}
						fmt.Println()

						if singlePath {
							return *results, *nodesVisited
						}
						if len(*results) >= maxResults && maxResults > 0 {
							return *results, *nodesVisited
						}
					}
				}

				nextLevel = append(nextLevel, resultElement)
			}
		}
	}

	return recursiveBFS(g, nextLevel, level+1, visited, levelMap, pathMap, results, nodesVisited, target, targetTier, maxResults, singlePath, baseElements, elements)
}

func buildPath(
	pathMap map[string][]model.Node,
	ingredients []string,
	result string,
	imagePath string,
	g *graph.ElementGraph,
	baseElements []string,
) []model.Node {
	var path []model.Node
	processed := make(map[string]bool)

	for _, ingredient := range ingredients {
		ingredientPath := pathMap[ingredient]
		for _, node := range ingredientPath {
			if isBaseElement(node.Element, baseElements) && !processed[node.Element] {
				path = append(path, node)
				processed[node.Element] = true
			}
		}
	}

	iteration := 0
	maxIterations := 100

	for len(processed) < countElements(ingredients, pathMap) && iteration < maxIterations {
		iteration++

		for _, ingredient := range ingredients {
			ingredientPath := pathMap[ingredient]

			for _, node := range ingredientPath {
				if isBaseElement(node.Element, baseElements) || processed[node.Element] {
					continue
				}

				// If element has ingredients, check if they're all processed
				if node.Ingredients != nil && len(node.Ingredients) > 0 {
					allDependenciesProcessed := true
					for _, dep := range node.Ingredients {
						if !processed[dep] {
							allDependenciesProcessed = false
							break
						}
					}

					if allDependenciesProcessed && !processed[node.Element] {
						path = append(path, node)
						processed[node.Element] = true
					}
				}
			}
		}

		if iteration > 5 {
			shouldContinue := false
			for _, ingredient := range ingredients {
				ingredientPath := pathMap[ingredient]
				for _, node := range ingredientPath {
					if !processed[node.Element] {
						path = append(path, node)
						processed[node.Element] = true
						shouldContinue = true
					}
				}
			}
			if !shouldContinue {
				break
			}
		}
	}

	// Finally, add the result element
	resultNode := model.Node{
		Element:     result,
		ImagePath:   imagePath,
		Ingredients: ingredients,
	}
	path = append(path, resultNode)

	return path
}

// total number of unique elements
func countElements(ingredients []string, pathMap map[string][]model.Node) int {
	uniqueElements := make(map[string]bool)

	for _, ingredient := range ingredients {
		ingredientPath := pathMap[ingredient]
		for _, node := range ingredientPath {
			uniqueElements[node.Element] = true
		}
	}

	return len(uniqueElements)
}

func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		fmt.Printf("Target element '%s' tidak ada di Little Alchemist 2\n", target)
		return [][]model.Node{}, 0
	}

	baseElements := g.BaseElements

	for _, base := range baseElements {
		if target == base {
			fmt.Printf("Target '%s' is a base element, returning direct path\n", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	resultChan := make(chan []model.Node, maxResults*10)
	visitedNodesChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex

	visited := make(map[string]bool)
	hasPathToBase := make(map[string]bool)
	allPathsMap := make(map[string][][]model.Node)
	uniquePathSignatures := make(map[string]bool)

	for _, elem := range baseElements {
		elemNode := g.Nodes[elem]
		visited[elem] = true
		path := []model.Node{{Element: elem, ImagePath: elemNode.ImagePath}}
		allPathsMap[elem] = [][]model.Node{path}
		hasPathToBase[elem] = true
	}

	targetFound := false
	visitedNodesCount := 0

	explorationStyles := []struct {
		name         string
		breadthFocus float64
		depthLimit   int
		randomFactor float64
	}{
		{"broad", 0.8, 30, 0.1},
		{"deep", 0.3, 40, 0.2},
		{"random", 0.5, 35, 0.4},
		{"balanced", 0.6, 25, 0.05},
	}

	for i, elem := range baseElements {
		wg.Add(1)
		styleIdx := i % len(explorationStyles)
		style := explorationStyles[styleIdx]

		go parallelExplore(elem, style, g, &mu, visited, hasPathToBase, allPathsMap, uniquePathSignatures, &targetFound, &visitedNodesCount, resultChan, target, baseElements, elements, maxResults, singlePath, &wg)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		visitedNodesChan <- visitedNodesCount
		close(visitedNodesChan)
		fmt.Println("All BFS goroutines finished")
	}()

	results := make([][]model.Node, 0, maxResults)
	for path := range resultChan {
		results = append(results, path)
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}

	finalResults := make([][]model.Node, 0, len(results))
	finalSignatures := make(map[string]bool)
	for _, path := range results {
		signature := generatePathSignature(path)
		if !finalSignatures[signature] {
			finalSignatures[signature] = true
			finalResults = append(finalResults, path)
		}
	}

	fmt.Printf("MultiThreadedBFS found %d unique paths\n", len(finalResults))
	totalVisited := <-visitedNodesChan
	return finalResults, totalVisited
}

func parallelExplore(
	startElement string,
	style struct {
		name         string
		breadthFocus float64
		depthLimit   int
		randomFactor float64
	},
	g *graph.ElementGraph,
	mu *sync.Mutex,
	visited map[string]bool,
	hasPathToBase map[string]bool,
	allPathsMap map[string][][]model.Node,
	uniquePathSignatures map[string]bool,
	targetFound *bool,
	visitedNodesCount *int,
	resultChan chan []model.Node,
	target string,
	baseElements []string,
	elements map[string]model.Element,
	maxResults int,
	singlePath bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	explored := make(map[string]bool)
	explored[startElement] = true

	currentLevel := []string{startElement}
	level := 0

	for len(currentLevel) > 0 && level <= style.depthLimit {
		nextLevel := []string{}

		for _, current := range currentLevel {
			mu.Lock()
			if len(uniquePathSignatures) >= maxResults && maxResults > 0 {
				mu.Unlock()
				return
			}
			mu.Unlock()

			mu.Lock()
			currentNode := g.Nodes[current]
			hasBaseElementPath := hasPathToBase[current]
			mu.Unlock()

			if !hasBaseElementPath {
				continue
			}

			possibleRecipes := currentNode.RecipesToMakeOtherElement
			if style.randomFactor > 0.2 {
				recipesCopy := make([]*graph.Recipe, len(possibleRecipes))
				copy(recipesCopy, possibleRecipes)
				r.Shuffle(len(recipesCopy), func(i, j int) {
					recipesCopy[i], recipesCopy[j] = recipesCopy[j], recipesCopy[i]
				})
				possibleRecipes = recipesCopy
			}

			for _, recipe := range possibleRecipes {
				resultElement := recipe.Result

				mu.Lock()
				resultNode := g.Nodes[resultElement]
				ingredients := recipe.Ingredients

				allIngredientsHavePathToBase := true
				for _, ingredient := range ingredients {
					if !hasPathToBase[ingredient] {
						allIngredientsHavePathToBase = false
						break
					}
				}
				if !allIngredientsHavePathToBase {
					mu.Unlock()
					continue
				}

				allIngredientsVisited := true
				for _, ingredient := range ingredients {
					if !visited[ingredient] {
						allIngredientsVisited = false
						break
					}
				}
				if !allIngredientsVisited {
					mu.Unlock()
					continue
				}

				allIngredientsHaveValidTier := true
				resultElementTier := elements[resultElement].Tier
				for _, ingredient := range ingredients {
					ingredientTier := elements[ingredient].Tier
					if ingredientTier > resultElementTier {
						fmt.Printf("Skipping recipe for '%s': ingredient '%s' (tier %d) > result (tier %d)\n",
							resultElement, ingredient, ingredientTier, resultElementTier)
						allIngredientsHaveValidTier = false
						break
					}
				}
				if !allIngredientsHaveValidTier {
					mu.Unlock()
					continue
				}

				pathCombinations := generatePathCombinations(allPathsMap, ingredients, resultElement, resultNode.ImagePath)

				wasVisited := visited[resultElement]
				visited[resultElement] = true
				if !wasVisited {
					*visitedNodesCount++
				}

				existingPaths := allPathsMap[resultElement]
				newPaths := mergePathSets(existingPaths, pathCombinations)

				maxPathsPerElement := 10
				if len(newPaths) > maxPathsPerElement {
					sort.Slice(newPaths, func(i, j int) bool {
						return len(newPaths[i]) < len(newPaths[j])
					})
					newPaths = newPaths[:maxPathsPerElement]
				}
				allPathsMap[resultElement] = newPaths
				hasPathToBase[resultElement] = true

				isTarget := resultElement == target
				if isTarget {
					*targetFound = true
					for _, path := range pathCombinations {
						signature := generatePathSignature(path)
						if !uniquePathSignatures[signature] {
							uniquePathSignatures[signature] = true
							resultPath := make([]model.Node, len(path))
							copy(resultPath, path)

							if utils.VerifyCompletePath(resultPath, baseElements, target) {
								select {
								case resultChan <- resultPath:
									fmt.Printf("Goroutine %s: Found path to target '%s', path length: %d\n",
										style.name, target, len(resultPath))
								default:
									// Channel full, skip this result
								}

								if singlePath && len(uniquePathSignatures) > 0 {
									mu.Unlock()
									return
								}
							}
						}
					}
				}
				mu.Unlock()

				if !explored[resultElement] && level < style.depthLimit {
					explored[resultElement] = true
					nextLevel = append(nextLevel, resultElement)
				}
			}
		}

		currentLevel = nextLevel
		level++

		if level%5 == 0 {
			fmt.Printf("Goroutine %s: exploring level %d, elements: %d\n",
				style.name, level, len(currentLevel))
		}
	}

	fmt.Printf("Goroutine %s finished after exploring to level %d\n", style.name, level)
}

func generatePathCombinations(allPathsMap map[string][][]model.Node, ingredients []string, result string, imagePath string) [][]model.Node {
	if len(ingredients) == 0 {
		return [][]model.Node{}
	}

	firstIngredient := ingredients[0]
	firstIngredientPaths := allPathsMap[firstIngredient]

	if len(ingredients) == 1 {
		resultPaths := make([][]model.Node, 0, len(firstIngredientPaths))
		for _, path := range firstIngredientPaths {
			newPath := make([]model.Node, len(path))
			copy(newPath, path)

			resultNode := model.Node{
				Element:     result,
				ImagePath:   imagePath,
				Ingredients: ingredients,
			}
			newPath = append(newPath, resultNode)
			resultPaths = append(resultPaths, newPath)
		}
		return resultPaths
	}

	var resultPaths [][]model.Node
	for _, firstPath := range firstIngredientPaths {
		secondIngredient := ingredients[1]
		secondIngredientPaths := allPathsMap[secondIngredient]

		for _, secondPath := range secondIngredientPaths {
			mergedPath := mergePaths(firstPath, secondPath)

			if len(ingredients) > 2 {
				remainingPaths := generatePathCombinations(allPathsMap, ingredients[2:], "", "")
				for _, remainingPath := range remainingPaths {
					fullPath := mergePaths(mergedPath, remainingPath)
					resultNode := model.Node{
						Element:     result,
						ImagePath:   imagePath,
						Ingredients: ingredients,
					}
					fullPath = append(fullPath, resultNode)

					uniquePath := deduplicatePath(fullPath)
					resultPaths = append(resultPaths, uniquePath)
				}
			} else {
				resultNode := model.Node{
					Element:     result,
					ImagePath:   imagePath,
					Ingredients: ingredients,
				}
				mergedPath = append(mergedPath, resultNode)

				uniquePath := deduplicatePath(mergedPath)
				resultPaths = append(resultPaths, uniquePath)
			}
		}
	}

	return resultPaths
}

func mergePaths(path1, path2 []model.Node) []model.Node {
	seen := make(map[string]bool)
	result := make([]model.Node, 0, len(path1)+len(path2))

	for _, node := range path1 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	for _, node := range path2 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	return result
}

func mergePathSets(set1, set2 [][]model.Node) [][]model.Node {
	uniquePaths := make(map[string][]model.Node)

	for _, path := range set1 {
		signature := generatePathSignature(path)
		uniquePaths[signature] = path
	}

	for _, path := range set2 {
		signature := generatePathSignature(path)
		uniquePaths[signature] = path
	}

	result := make([][]model.Node, 0, len(uniquePaths))
	for _, path := range uniquePaths {
		result = append(result, path)
	}

	return result
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
func generatePathSignature(path []model.Node) string {
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
