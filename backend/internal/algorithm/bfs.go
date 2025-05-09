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

// BFS finds recipe paths from base elements to target using Breadth-First Search
// Parameters:
// - elements: map of all available elements
// - target: the element we want to find recipes for
// - maxResults: maximum number of different recipes to find
// - singlePath: whether to return only the shortest path or multiple paths
// Returns:
// - [][]model.Node: list of recipe paths
// - int: number of nodes visited after target found
// BFS finds recipe paths from base elements to target using Breadth-First Search
// Now implemented with a recursive approach for better clarity and multiple path support
func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Build the graph once
	g := graph.NewElementGraph(elements)

	// Check if target exists in the graph
	targetNode, exists := g.Nodes[target]
	if !exists {
		fmt.Printf("Target element '%s' does not exist in the database\n", target)
		return [][]model.Node{}, 0
	}

	// Get base elements
	baseElements := g.BaseElements
	fmt.Printf("Base elements found: %v\n", baseElements)

	// Handle case where target is a base element
	for _, base := range baseElements {
		if target == base {
			fmt.Printf("Target '%s' is a base element, returning direct path\n", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	// Initialize data structures!
	visited := make(map[string]bool)
	pathMap := make(map[string][]model.Node)
	hasPathToBase := make(map[string]bool)
	elementIngredients := make(map[string][]string)
	results := [][]model.Node{}
	nodesVisited := 0
	targetTier := elements[target].Tier

	// Initialize first level with base elements
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

	// Start recursive BFS with the first level
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

// Helper function to process each BFS level recursively
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
	// Base case: no more elements in this level or we already have enough results
	if len(currentLevel) == 0 || (singlePath && len(results) > 0) {
		return results, *nodesVisited
	}

	// Process current level and prepare next level
	var nextLevel []string
	*nodesVisited += len(currentLevel)

	// Process all elements in current level
	for _, current := range currentLevel {
		currentNode := g.Nodes[current]

		// Skip if this element doesn't have path to base
		if !hasPathToBase[current] {
			continue
		}

		// Check recipes where current element is used as an ingredient
		for _, recipe := range currentNode.RecipesToMakeOtherElement {
			resultElement := recipe.Result
			resultNode := g.Nodes[resultElement]

			// Validate all ingredients are visited and have paths to base
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

			// Check tier validity - higher tier elements can't create lower tier elements
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

			// Process element only if all validations pass
			if allIngredientsVisited && allIngredientsHavePathToBase && allIngredientsHaveValidTier {
				// Skip if already visited
				if visited[resultElement] {
					continue
				}

				visited[resultElement] = true

				// Store ingredient information
				elementIngredients[resultElement] = recipe.Ingredients

				// Create structured path for this element
				path := buildStructuredPath(pathMap, recipe.Ingredients, resultElement, resultNode.ImagePath, g, baseElements)
				pathMap[resultElement] = path

				// Mark that this element has path to base
				hasPathToBase[resultElement] = true

				// Check if we found the target
				if resultElement == target {
					fmt.Printf("Found target '%s'! Path length: %d\n", target, len(path))

					// Validate the complete path for tier constraints
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
						// Add to results
						results = append(results, path)

						// Print path for debugging
						fmt.Printf("Path to '%s': ", target)
						for i, node := range path {
							if i > 0 {
								fmt.Printf(" -> ")
							}
							fmt.Printf("%s", node.Element)
						}
						fmt.Println()

						// Return immediately if we only want shortest path
						if singlePath {
							return results, *nodesVisited
						}

						// Stop if we have enough results
						if len(results) >= maxResults && maxResults > 0 {
							return results, *nodesVisited
						}
					}
				}

				// Add to next level for further exploration
				nextLevel = append(nextLevel, resultElement)
			}
		}
	}

	// Process next level recursively
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

// The rest of the helper functions remain unchanged
// buildStructuredPath, countUniqueElements, etc.

// Helper function to build a structured path ensuring elements are properly organized
func buildStructuredPath(pathMap map[string][]model.Node, ingredients []string, result string, imagePath string, g *graph.ElementGraph, baseElements []string) []model.Node {
	// Start with base elements, then build up to the result
	var structuredPath []model.Node
	processedElements := make(map[string]bool)

	// First, add all base elements
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

	// Next, add intermediate elements in order of dependency
	for len(processedElements) < countUniqueElements(ingredients, pathMap) {
		for _, ingredient := range ingredients {
			ingredientPath := pathMap[ingredient]
			for _, node := range ingredientPath {
				// Skip base elements (already added) and already processed elements
				if isBaseElement(node.Element, baseElements) || processedElements[node.Element] {
					continue
				}

				// Check if all ingredients for this node are already processed
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

		// Break if we couldn't add any new elements (avoid infinite loop)
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

	// Add the result element last, ensuring its ingredients are correctly set
	resultNode := model.Node{
		Element:     result,
		ImagePath:   imagePath,
		Ingredients: ingredients,
	}

	structuredPath = append(structuredPath, resultNode)
	return structuredPath
}

// Helper to count unique elements in the ingredient paths
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

// ImprovedMultiThreadedBFS performs BFS using multiple goroutines for better performance
// This version is optimized to find multiple unique paths, including paths that differ by just one node
// ImprovedMultiThreadedBFS performs BFS using multiple goroutines for better performance and diversity
// This version is optimized to find multiple unique paths, including paths that differ by just one node
func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Build the graph once
	g := graph.NewElementGraph(elements)

	// Check if target exists
	targetNode, exists := g.Nodes[target]
	if !exists {
		fmt.Printf("Target element '%s' does not exist in the database\n", target)
		return [][]model.Node{}, 0
	}

	baseElements := g.BaseElements

	// Handle case where target is a base element
	for _, base := range baseElements {
		if target == base {
			fmt.Printf("Target '%s' is a base element, returning direct path\n", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	// Channels for communication between goroutines
	resultChan := make(chan []model.Node, maxResults*10) // Increased buffer for more paths
	visitedNodesChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex // For synchronizing access to shared data

	// Shared maps for visited elements
	visited := make(map[string]bool)
	// Map to track if an element has a complete path to base elements
	hasPathToBase := make(map[string]bool)
	// Use a map to store all discovered paths to each element
	// This is key to finding multiple recipes - we'll store multiple paths for each element
	allPathsMap := make(map[string][][]model.Node)

	// Track unique path signatures to avoid duplicates
	uniquePathSignatures := make(map[string]bool)

	// Initialize base elements in shared maps
	for _, elem := range baseElements {
		elemNode := g.Nodes[elem]
		visited[elem] = true
		path := []model.Node{{Element: elem, ImagePath: elemNode.ImagePath}}
		allPathsMap[elem] = [][]model.Node{path}
		// Base elements have a path to themselves by definition
		hasPathToBase[elem] = true
	}

	targetFound := false
	visitedNodesAfterTarget := 0

	// Create a random source for each goroutine to introduce variety
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// PERBAIKAN: Parameter untuk keberagaman eksplorasi
	explorationStyles := []struct {
		name         string
		breadthFocus float64 // 0-1: mengutamakan breadth (lebar) eksplorasi
		depthLimit   int     // batas kedalaman eksplorasi
		randomFactor float64 // 0-1: tingkat keacakan dalam pemilihan node
	}{
		{"broad", 0.8, 30, 0.1},           // Eksplorasi luas, sedikit keacakan
		{"deep", 0.3, 40, 0.2},            // Eksplorasi dalam, lebih banyak keacakan
		{"random", 0.5, 35, 0.4},          // Seimbang dengan keacakan tinggi
		{"balanced", 0.6, 25, 0.05},       // Seimbang, keacakan rendah
		{"prioritize_base", 0.9, 20, 0.3}, // Prioritas elemen dasar
	}

	// Start a goroutine for each base element with different exploration strategies
	for i, elem := range baseElements {
		wg.Add(1)

		// Pilih strategi eksplorasi yang berbeda untuk setiap goroutine
		styleIdx := i % len(explorationStyles)
		style := explorationStyles[styleIdx]

		go func(startElement string, style struct {
			name         string
			breadthFocus float64
			depthLimit   int
			randomFactor float64
		}) {
			defer wg.Done()

			fmt.Printf("Starting goroutine for '%s' with exploration style: %s\n",
				startElement, style.name)

			// Buat queue lokal untuk BFS dengan prioritas
			localQueue := list.New()
			localQueue.PushBack(startElement)

			// Batasan untuk mencegah eksplorasi terlalu dalam
			depth := 0
			currentLevelSize := 1
			nextLevelSize := 0

			// Set untuk mencatat elemen yang ditambahkan ke queue (anti-duplikat)
			localEnqueued := make(map[string]bool)
			localEnqueued[startElement] = true

			for localQueue.Len() > 0 && depth <= style.depthLimit {
				// Check if we have found enough results
				mu.Lock()
				if len(uniquePathSignatures) >= maxResults && maxResults > 0 {
					mu.Unlock()
					return
				}

				// Pilih elemen berikutnya - kadang secara acak untuk meningkatkan keberagaman
				var current string
				var currentElement *list.Element

				if r.Float64() < style.randomFactor && localQueue.Len() > 3 {
					// Pilih node secara acak dari queue
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

				// Skip if the current element doesn't have a path to base elements
				if !hasBaseElementPath {
					continue
				}

				// Explore recipes where current element is used as ingredient
				possibleRecipes := currentNode.RecipesToMakeOtherElement

				// PERBAIKAN: Secara acak urutkan resep untuk meningkatkan keberagaman
				if style.randomFactor > 0.2 {
					// Buat copy dari resep untuk diacak
					recipesCopy := make([]*graph.Recipe, len(possibleRecipes))
					copy(recipesCopy, possibleRecipes)

					// Acak urutan eksplorasi resep
					r.Shuffle(len(recipesCopy), func(i, j int) {
						recipesCopy[i], recipesCopy[j] = recipesCopy[j], recipesCopy[i]
					})

					possibleRecipes = recipesCopy
				}

				for _, recipe := range possibleRecipes {
					resultElement := recipe.Result

					mu.Lock()
					resultNode := g.Nodes[resultElement]

					// Get all ingredients for this recipe
					ingredients := recipe.Ingredients

					// Check if all ingredients have paths to base elements
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

					// Check if all ingredients have been visited
					allIngredientsVisited := true
					for _, ingredient := range ingredients {
						if !visited[ingredient] {
							allIngredientsVisited = false
							break
						}
					}

					// Skip if not all ingredients are visited
					if !allIngredientsVisited {
						mu.Unlock()
						continue
					}

					// PERBAIKAN: Validasi tier
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

					// This is the key change - create multiple path combinations
					// For each ingredient, we may have multiple paths
					// We need to generate all valid combinations
					pathCombinations := generatePathCombinations(allPathsMap, ingredients, resultElement, resultNode.ImagePath)

					// Update visited flag if this is a new element
					wasVisited := visited[resultElement]
					visited[resultElement] = true

					if !wasVisited && isTargetFound {
						visitedNodesAfterTarget++
					}

					// Store all valid path combinations
					existingPaths := allPathsMap[resultElement]
					newPaths := mergePathSets(existingPaths, pathCombinations)

					// PERBAIKAN: Batasi jumlah path yang disimpan per elemen untuk mencegah penggunaan memori berlebihan
					maxPathsPerElement := 10
					if len(newPaths) > maxPathsPerElement {
						// Prioritas short paths
						sort.Slice(newPaths, func(i, j int) bool {
							return len(newPaths[i]) < len(newPaths[j])
						})
						newPaths = newPaths[:maxPathsPerElement]
					}

					allPathsMap[resultElement] = newPaths

					// Mark that this element has a path to base elements
					hasPathToBase[resultElement] = true

					isTarget := resultElement == target
					if isTarget {
						targetFound = true

						// For each unique path to the target, send it to the result channel
						for _, path := range pathCombinations {
							// Generate a signature for this path to check uniqueness
							signature := generatePathSignature(path)

							if !uniquePathSignatures[signature] {
								uniquePathSignatures[signature] = true

								// Make a copy of the path to avoid race conditions
								resultPath := make([]model.Node, len(path))
								copy(resultPath, path)

								// Verify this is a complete path
								if verifyCompletePath(resultPath, baseElements, target) {
									// Send the result through the channel
									select {
									case resultChan <- resultPath:
										fmt.Printf("Goroutine %s: Found path to target '%s', path length: %d\n",
											style.name, target, len(resultPath))
									default:
										// Channel full, skip this result
									}

									// Stop exploring if we only want the shortest path
									if singlePath && len(uniquePathSignatures) > 0 {
										mu.Unlock()
										return
									}
								}
							}
						}
					}

					mu.Unlock()

					// Continue BFS exploration
					if !localEnqueued[resultElement] && depth < style.depthLimit {
						localQueue.PushBack(resultElement)
						localEnqueued[resultElement] = true
						nextLevelSize++
					}
				}

				// Track BFS level for debugging and depth limiting
				currentLevelSize--
				if currentLevelSize == 0 {
					depth++
					currentLevelSize = nextLevelSize
					nextLevelSize = 0

					// Debug output
					if depth%5 == 0 {
						fmt.Printf("Goroutine %s: Exploring depth %d, queue size: %d\n",
							style.name, depth, localQueue.Len())
					}
				}
			}

			fmt.Printf("Goroutine %s finished after exploring to depth %d\n", style.name, depth)
		}(elem, style)
	}

	// Wait for all goroutines to finish and close channels
	go func() {
		wg.Wait()
		close(resultChan)
		visitedNodesChan <- visitedNodesAfterTarget
		close(visitedNodesChan)
		fmt.Println("All BFS goroutines finished")
	}()

	// Collect results from the channel
	results := make([][]model.Node, 0, maxResults)
	for path := range resultChan {
		results = append(results, path)
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}

	// Deduplicate results for final output
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

// generatePathCombinations creates all possible path combinations from ingredient paths
func generatePathCombinations(allPathsMap map[string][][]model.Node, ingredients []string, result string, imagePath string) [][]model.Node {
	if len(ingredients) == 0 {
		return [][]model.Node{}
	}

	// For the first ingredient, get all its paths
	firstIngredient := ingredients[0]
	firstIngredientPaths := allPathsMap[firstIngredient]

	// If there's only one ingredient, create paths with just that ingredient
	if len(ingredients) == 1 {
		resultPaths := make([][]model.Node, 0, len(firstIngredientPaths))

		for _, path := range firstIngredientPaths {
			newPath := make([]model.Node, len(path))
			copy(newPath, path)

			// Add the result element
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

	// For multiple ingredients, we need to combine their paths
	var resultPaths [][]model.Node

	// For each path of the first ingredient
	for _, firstPath := range firstIngredientPaths {
		// Get the second ingredient paths
		secondIngredient := ingredients[1]
		secondIngredientPaths := allPathsMap[secondIngredient]

		// For each path of the second ingredient
		for _, secondPath := range secondIngredientPaths {
			// Merge the paths
			mergedPath := mergePaths(firstPath, secondPath)

			// If there are more ingredients, recursively merge them
			if len(ingredients) > 2 {
				remainingPaths := generatePathCombinations(allPathsMap, ingredients[2:], "", "")

				// For each remaining path combination
				for _, remainingPath := range remainingPaths {
					fullPath := mergePaths(mergedPath, remainingPath)

					// Add result node
					resultNode := model.Node{
						Element:     result,
						ImagePath:   imagePath,
						Ingredients: ingredients,
					}
					fullPath = append(fullPath, resultNode)
					resultPaths = append(resultPaths, fullPath)
				}
			} else {
				// Just add the result node to the merged path
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

// mergePaths combines two paths, avoiding duplicates while preserving order
func mergePaths(path1, path2 []model.Node) []model.Node {
	// Create a set to track elements we've already added
	seen := make(map[string]bool)

	// Start with all nodes from path1
	result := make([]model.Node, 0, len(path1)+len(path2))

	// Add all nodes from path1
	for _, node := range path1 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	// Add nodes from path2 that aren't already in result
	for _, node := range path2 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	return result
}

// mergePathSets combines two sets of paths, removing duplicates
func mergePathSets(set1, set2 [][]model.Node) [][]model.Node {
	// Create a map to track unique path signatures
	uniquePaths := make(map[string][]model.Node)

	// Add all paths from set1
	for _, path := range set1 {
		signature := generatePathSignature(path)
		uniquePaths[signature] = path
	}

	// Add all paths from set2
	for _, path := range set2 {
		signature := generatePathSignature(path)
		uniquePaths[signature] = path
	}

	// Convert map back to slice
	result := make([][]model.Node, 0, len(uniquePaths))
	for _, path := range uniquePaths {
		result = append(result, path)
	}

	return result
}

// generatePathSignature creates a unique string signature for a path
func generatePathSignature(path []model.Node) string {
	// Create a representation of the path that captures its uniqueness
	var signature strings.Builder

	for i, node := range path {
		// Add element name
		signature.WriteString(node.Element)

		// Add ingredients if any
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

// Helper functions remain the same
func verifyCompletePath(path []model.Node, baseElements []string, target string) bool {
	if len(path) == 0 {
		return false
	}

	// Path should end with target
	if path[len(path)-1].Element != target {
		return false
	}

	// First element should be a base element
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

// Helper function to create a path from ingredients to result
func createPathFromGraph(pathMap map[string][]model.Node, ingredients []string, result string, imagePath string) []model.Node {
	// Create a set for fast lookup
	seenElements := make(map[string]bool)

	// Create a merged path from all ingredient paths
	var mergedPath []model.Node

	// Add all nodes from ingredient paths, avoiding duplicates
	// We maintain the order by adding base elements first
	for _, ingredient := range ingredients {
		ingredientPath := pathMap[ingredient]
		for _, node := range ingredientPath {
			if !seenElements[node.Element] {
				seenElements[node.Element] = true
				mergedPath = append(mergedPath, node)
			}
		}
	}

	// Add the result node with its ingredients
	resultNode := model.Node{
		Element:     result,
		ImagePath:   imagePath,
		Ingredients: ingredients,
	}

	mergedPath = append(mergedPath, resultNode)
	return mergedPath
}
