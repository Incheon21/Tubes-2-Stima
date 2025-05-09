package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"container/list"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

// parameternya -> graf yg ud dibuat, target yg mau dicari, max hasil yg mau diambil, singlePath buat ambil satu path doang
func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Build the graph once
	g := graph.NewElementGraph(elements)

	//validasi ada ato engga
	targetNode, exists := g.Nodes[target]
	if !exists {
		fmt.Printf("Target element '%s' tidak exists di The Little Alchemist 2\n", target)
		return [][]model.Node{}, 0
	}

	//base element
	baseElements := g.BaseElements
	fmt.Printf("Ditemukan base elemen: %v\n", baseElements)

	//--> handle case kalo targetnya base eleemnt
	for _, base := range baseElements {
		if target == base {
			fmt.Printf("Target '%s' adalah base element\n", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	visited := make(map[string]bool)                //track visited nodes
	pathMap := make(map[string][]model.Node)        //path buat balik ke base elemen
	hasPathToBase := make(map[string]bool)          //cekk ada jalur ga?
	elementIngredients := make(map[string][]string) //buat track ingredients
	results := [][]model.Node{}                     // hasil
	nodesVisited := 0
	targetTier := elements[target].Tier // tier dari target

	currentLevel := []string{}
	for _, elem := range baseElements {
		elemNode := g.Nodes[elem]
		currentLevel = append(currentLevel, elem)
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem, ImagePath: elemNode.ImagePath},
		}
		hasPathToBase[elem] = true
	}

	//rekursif lagi buat lanjut ke level berikutnya
	return bfsLevelProcessing(
		g,
		currentLevel,
		visited,
		pathMap,
		hasPathToBase,
		elementIngredients,
		results,
		&nodesVisited,
		target,
		targetTier,
		maxResults,
		singlePath,
		baseElements,
		elements,
	)
}

// helper functinon bfs (level)
func bfsLevelProcessing(
	g *graph.ElementGraph,
	currentLevel []string,
	visited map[string]bool,
	pathMap map[string][]model.Node,
	hasPathToBase map[string]bool,
	elementIngredients map[string][]string,
	results [][]model.Node,
	nodesVisited *int,
	target string,
	targetTier int,
	maxResults int,
	singlePath bool,
	baseElements []string,
	elements map[string]model.Element,
) ([][]model.Node, int) {
	if len(currentLevel) == 0 || (singlePath && len(results) > 0) {
		return results, *nodesVisited //kalo cuma mau satu path doang ato udah ga ada level lagi (base element)
	}

	var nextLevel []string             //buat prepare next level ini inssialisasi dulu
	*nodesVisited += len(currentLevel) //proses

	for _, current := range currentLevel {
		currentNode := g.Nodes[current]
		if !hasPathToBase[current] {
			continue
		}
		for _, recipe := range currentNode.RecipesToMakeOtherElement {
			resultElement := recipe.Result
			resultNode := g.Nodes[resultElement]
			allIngredientsVisited := true
			allIngredientsHavePathToBase := true

			for _, ingredient := range recipe.Ingredients {
				if !visited[ingredient] {
					allIngredientsVisited = false
					break
				}
				if !hasPathToBase[ingredient] {
					allIngredientsHavePathToBase = false
					break
				}
			}

			allIngredientsHaveValidTier := true
			resultElementTier := elements[resultElement].Tier

			for _, ingredient := range recipe.Ingredients {
				ingredientTier := elements[ingredient].Tier
				if ingredientTier > resultElementTier {
					fmt.Printf("skipping recipe for '%s': ingredient '%s' (tier %d) > result (tier %d)\n",
						resultElement, ingredient, ingredientTier, resultElementTier)
					allIngredientsHaveValidTier = false
					break
				}
			}

			if allIngredientsVisited && allIngredientsHavePathToBase && allIngredientsHaveValidTier {
				if visited[resultElement] {
					continue
				}

				visited[resultElement] = true

				elementIngredients[resultElement] = recipe.Ingredients

				path := buildStructuredPath(pathMap, recipe.Ingredients, resultElement, resultNode.ImagePath, g, baseElements)
				pathMap[resultElement] = path

				hasPathToBase[resultElement] = true

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
						results = append(results, path)

						fmt.Printf("Path to '%s': ", target)
						for i, node := range path {
							if i > 0 {
								fmt.Printf(" -> ")
							}
							fmt.Printf("%s", node.Element)
						}
						fmt.Println()
						if singlePath {
							return results, *nodesVisited
						}
						if len(results) >= maxResults && maxResults > 0 {
							return results, *nodesVisited
						}
					}
				}
				nextLevel = append(nextLevel, resultElement)
			}
		}
	}
	return bfsLevelProcessing(
		g,
		nextLevel,
		visited,
		pathMap,
		hasPathToBase,
		elementIngredients,
		results,
		nodesVisited,
		target,
		targetTier,
		maxResults,
		singlePath,
		baseElements,
		elements,
	)
}

func buildStructuredPath(pathMap map[string][]model.Node, ingredients []string, result string, imagePath string, g *graph.ElementGraph, baseElements []string) []model.Node {
	var structuredPath []model.Node
	processedElements := make(map[string]bool)

	for _, ingredient := range ingredients {
		ingredientPath := pathMap[ingredient]
		for _, node := range ingredientPath {
			isBase := isBaseElement(node.Element, baseElements)
			if isBase && !processedElements[node.Element] {
				structuredPath = append(structuredPath, node)
				processedElements[node.Element] = true
			}
		}
	}

	for len(processedElements) < countUniqueElements(ingredients, pathMap) {
		for _, ingredient := range ingredients {
			ingredientPath := pathMap[ingredient]
			for _, node := range ingredientPath {
				if isBaseElement(node.Element, baseElements) || processedElements[node.Element] {
					continue
				}

				if node.Ingredients != nil && len(node.Ingredients) > 0 {
					allIngredientsProcessed := true
					for _, ing := range node.Ingredients {
						if !processedElements[ing] {
							allIngredientsProcessed = false
							break
						}
					}

					if allIngredientsProcessed && !processedElements[node.Element] {
						structuredPath = append(structuredPath, node)
						processedElements[node.Element] = true
					}
				}
			}
		}

		previousLength := len(processedElements)
		for _, ingredient := range ingredients {
			ingredientPath := pathMap[ingredient]
			for _, node := range ingredientPath {
				if !processedElements[node.Element] {
					processedElements[node.Element] = true
				}
			}
		}

		if len(processedElements) == previousLength {
			break
		}
	}

	resultNode := model.Node{
		Element:     result,
		ImagePath:   imagePath,
		Ingredients: ingredients,
	}

	structuredPath = append(structuredPath, resultNode)
	return structuredPath
}

func countUniqueElements(ingredients []string, pathMap map[string][]model.Node) int {
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
	var mu sync.Mutex //mutex buat sinkronisasi antar goroutine

	//share map buat node yg divisit
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
	visitedNodesAfterTarget := 0

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	explorationStyles := []struct {
		name         string
		breadthFocus float64 // 0-1: mengutamakan breadth (lebar) eksplorasi
		depthLimit   int     // batas kedalaman eksplorasi
		randomFactor float64 // 0-1: tingkat keacakan dalam pemilihan node
	}{
		{"broad", 0.8, 30, 0.1},
		{"deep", 0.3, 40, 0.2},
		{"random", 0.5, 35, 0.4},
		{"balanced", 0.6, 25, 0.05},
		{"prioritize_base", 0.9, 20, 0.3},
	}

	for i, elem := range baseElements {
		wg.Add(1)

		//gouroutine
		styleIdx := i % len(explorationStyles)
		style := explorationStyles[styleIdx]

		go func(startElement string, style struct {
			name         string
			breadthFocus float64
			depthLimit   int
			randomFactor float64
		}) {
			defer wg.Done()

			localQueue := list.New()
			localQueue.PushBack(startElement)

			depth := 0
			currentLevelSize := 1
			nextLevelSize := 0

			localEnqueued := make(map[string]bool)
			localEnqueued[startElement] = true

			for localQueue.Len() > 0 && depth <= style.depthLimit {
				//di freeze dl, buat cek unique ato ga
				mu.Lock()
				if len(uniquePathSignatures) >= maxResults && maxResults > 0 {
					mu.Unlock()
					return
				}
				var current string
				var currentElement *list.Element

				if r.Float64() < style.randomFactor && localQueue.Len() > 3 {
					//pilih elemen secara acak dari queue (bukan FIFO)
					idx := r.Intn(localQueue.Len())
					currentElement = localQueue.Front()
					for i := 0; i < idx; i++ {
						currentElement = currentElement.Next()
					}
					current = currentElement.Value.(string)
					localQueue.Remove(currentElement)
				} else {
					// Cara normal BFS (FIFO)
					current = localQueue.Front().Value.(string)
					localQueue.Remove(localQueue.Front())
				}

				currentNode := g.Nodes[current]
				hasBaseElementPath := hasPathToBase[current]
				isTargetFound := targetFound

				mu.Unlock()

				if !hasBaseElementPath {
					continue
				}

				possibleRecipes := currentNode.RecipesToMakeOtherElement

				if style.randomFactor > 0.2 {
					//buat copy buat diacak -> biar cegah rae condition dan dll lah os wkkw
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
					// bwt dptin ingridient
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
					if !wasVisited && isTargetFound {
						visitedNodesAfterTarget++
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
						targetFound = true
						for _, path := range pathCombinations {
							signature := generatePathSignature(path)
							if !uniquePathSignatures[signature] {
								uniquePathSignatures[signature] = true
								resultPath := make([]model.Node, len(path))
								copy(resultPath, path)
								if verifyCompletePath(resultPath, baseElements, target) {
									select {
									case resultChan <- resultPath:
										fmt.Printf("Goroutine %s: Found path to target '%s', path length: %d\n",
											style.name, target, len(resultPath))
									default:
										//channel full, skip this result
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
					if !localEnqueued[resultElement] && depth < style.depthLimit {
						localQueue.PushBack(resultElement)
						localEnqueued[resultElement] = true
						nextLevelSize++
					}
				}
				currentLevelSize--
				if currentLevelSize == 0 {
					depth++
					currentLevelSize = nextLevelSize
					nextLevelSize = 0
					if depth%5 == 0 {
						fmt.Printf("Goroutine %s: exploring depth %d, queue size: %d\n",
							style.name, depth, localQueue.Len())
					}
				}
			}
			fmt.Printf("Goroutine %s finished after exploring to depth %d\n", style.name, depth)
		}(elem, style)
	}
	go func() {
		wg.Wait()
		close(resultChan)
		visitedNodesChan <- visitedNodesAfterTarget
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
					resultPaths = append(resultPaths, fullPath)
				}
			} else {
				resultNode := model.Node{
					Element:     result,
					ImagePath:   imagePath,
					Ingredients: ingredients,
				}
				mergedPath = append(mergedPath, resultNode)
				resultPaths = append(resultPaths, mergedPath)
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

func generatePathSignature(path []model.Node) string {
	var signature strings.Builder

	for i, node := range path {
		signature.WriteString(node.Element)

		if len(node.Ingredients) > 0 {
			signature.WriteString("(")
			for j, ing := range node.Ingredients {
				signature.WriteString(ing)
				if j < len(node.Ingredients)-1 {
					signature.WriteString(",")
				}
			}
			signature.WriteString(")")
		}

		if i < len(path)-1 {
			signature.WriteString("-")
		}
	}

	return signature.String()
}
func verifyCompletePath(path []model.Node, baseElements []string, target string) bool {
	if len(path) == 0 {
		return false
	}
	if path[len(path)-1].Element != target {
		return false
	}
	firstElem := path[0].Element
	isBaseElement := false
	for _, base := range baseElements {
		if firstElem == base {
			isBaseElement = true
			break
		}
	}
	return isBaseElement
}

func GetElementTreeBFS(g *graph.ElementGraph, elementName string) (map[string]interface{}, int) {
	visited := make(map[string]bool)
	visitedCount := 0

	result := buildElementTreeBFS(g, elementName, visited, &visitedCount)
	return result, visitedCount
}

func buildElementTreeBFS(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
	if visited[elementName] {
		// If we've seen this element before, just return its info without recursion
		node := g.Nodes[elementName]
		return map[string]interface{}{
			"name":                elementName,
			"imagePath":           node.ImagePath,
			"isCircularReference": true,
		}
	}

	visited[elementName] = true
	*visitedCount++

	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Check if it's a base element
	for _, base := range baseElements {
		if elementName == base {
			return map[string]interface{}{
				"name":          elementName,
				"imagePath":     node.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}
		}
	}

	// Get the first recipe to make this element (BFS takes the first recipe found)
	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found, might be a base element not in our list
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	// Choose first recipe (BFS approach)
	recipe := node.RecipesToMakeThisElement[0]
	ingredients := make([]interface{}, 0, len(recipe.Ingredients))

	// Process ingredients in order
	for _, ingredientName := range recipe.Ingredients {
		ingredientTree := buildElementTreeBFS(g, ingredientName, visited, visitedCount)
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        elementName,
		"imagePath":   node.ImagePath,
		"ingredients": ingredients,
	}
}
