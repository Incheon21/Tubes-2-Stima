package algorithm

import (
	"backend/model"
	"container/list"
	"fmt"
	"sync"
)

// BFS finds recipe paths from base elements to target using Breadth-First Search
// Parameters:
// - elements: map of all available elements
// - target: the element we want to find recipes for
// - maxResults: maximum number of different recipes to find
// - singlePath: whether to return only the shortest path or multiple paths
// - debug: whether to track traversal order for visualization
// Returns:
// - [][]model.Node: list of recipe paths
// - int: number of nodes visited after target found
// - []string: traversal order (only if debug is true)
func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool, debug bool) ([][]model.Node, int, []string) {
	var traversalOrder []string

	// Check if target exists in the element database
	if _, exists := elements[target]; !exists {
		fmt.Printf("Target element '%s' does not exist in the database\n", target)
		return [][]model.Node{}, 0, nil
	}

	// Get base elements (air, water, fire, earth)
	baseElements := getBaseElements(elements)

	// Log found base elements
	fmt.Printf("Base elements found: %v\n", baseElements)

	// Handle case where target is a base element
	if isBaseElement(target, baseElements) {
		fmt.Printf("Target '%s' is a base element, returning direct path\n", target)
		if debug {
			return [][]model.Node{{
				{Element: target},
			}}, 0, []string{target}
		} else {
			return [][]model.Node{{
				{Element: target},
			}}, 0, nil
		}
	}

	// Initialize data structures
	queue := list.New()
	visited := make(map[string]bool)
	pathMap := make(map[string][]model.Node)
	var results [][]model.Node
	visitedNodesAfterTarget := 0
	targetFound := false

	// Initialize queue with base elements
	for _, elem := range baseElements {
		queue.PushBack(elem)
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem},
		}

		if debug {
			traversalOrder = append(traversalOrder, elem)
		}
	}

	// BFS traversal
	for queue.Len() > 0 && (len(results) < maxResults || !targetFound) {
		current := queue.Front().Value.(string)
		queue.Remove(queue.Front())

		if debug {
			traversalOrder = append(traversalOrder, current)
		}

		// Try combining current element with all visited elements
		for otherElement := range visited {
			possibleResults := combineElements(current, otherElement, elements)

			for _, result := range possibleResults {
				if visited[result] {
					continue
				}

				visited[result] = true

				// Increment counter if target already found
				if targetFound {
					visitedNodesAfterTarget++
				}

				// Create and store the path to this element
				path := createPath(pathMap, current, otherElement, result)

				// Validate path - ensure it contains only required elements and reaches base elements
				validPath := validatePath(path, baseElements)
				if !validPath {
					fmt.Printf("Warning: Invalid path created for element '%s'\n", result)
					// Skip this result if path is invalid
					continue
				}

				pathMap[result] = path

				// Check if we found the target element
				if result == target {
					fmt.Printf("Found target '%s'! Path length: %d\n", target, len(path))
					targetFound = true
					results = append(results, path)

					// Print the path for debugging
					fmt.Printf("Path to '%s': ", target)
					for i, node := range path {
						if i > 0 {
							fmt.Printf(" -> ")
						}
						fmt.Printf("%s", node.Element)
					}
					fmt.Println()

					// If we only want one path (shortest), return immediately
					if singlePath {
						return results, visitedNodesAfterTarget, traversalOrder
					}

					// If we have reached max results, return
					if len(results) >= maxResults {
						return results, visitedNodesAfterTarget, traversalOrder
					}
				}

				// Add to queue for further exploration
				queue.PushBack(result)
			}
		}
	}

	if !targetFound {
		fmt.Printf("Could not find a path to target '%s'\n", target)
	}

	return results, visitedNodesAfterTarget, traversalOrder
}

// validatePath ensures a path is valid by checking that all elements in the path
// are properly connected and that leaf nodes are base elements
func validatePath(path []model.Node, baseElements []string) bool {
	if len(path) == 0 {
		return false
	}

	// Create a map of elements in the path for quick lookup
	elementsInPath := make(map[string]bool)
	for _, node := range path {
		elementsInPath[node.Element] = true
	}

	// Check that all ingredients referenced exist in the path or are base elements
	for _, node := range path {
		if len(node.Ingredients) > 0 {
			for _, ingredient := range node.Ingredients {
				if !elementsInPath[ingredient] && !isBaseElement(ingredient, baseElements) {
					// Found an ingredient that's neither in our path nor a base element
					return false
				}
			}
		}
	}

	return true
}

// Helper function to create a path from element1 and element2 to result
func createPath(pathMap map[string][]model.Node, element1, element2, result string) []model.Node {
	// Create a deep copy of both paths
	path1 := deepCopyPath(pathMap[element1])
	path2 := deepCopyPath(pathMap[element2])

	// Merge paths and remove duplicates
	resultPath := mergePaths(path1, path2)

	// Add the new result element with its ingredients
	resultNode := model.Node{
		Element:     result,
		Ingredients: []string{element1, element2},
	}

	resultPath = append(resultPath, resultNode)

	return resultPath
}

// Deep copy a path to avoid modifying the original
func deepCopyPath(nodes []model.Node) []model.Node {
	result := make([]model.Node, len(nodes))
	for i, node := range nodes {
		// Copy the node
		result[i].Element = node.Element

		// Copy ingredients if any
		if len(node.Ingredients) > 0 {
			result[i].Ingredients = make([]string, len(node.Ingredients))
			copy(result[i].Ingredients, node.Ingredients)
		}
	}
	return result
}

// Merge two paths, removing duplicates
func mergePaths(path1, path2 []model.Node) []model.Node {
	// Use a map to track elements we've already included
	seen := make(map[string]bool)
	var result []model.Node

	// Process path1
	for _, node := range path1 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	// Process path2
	for _, node := range path2 {
		if !seen[node.Element] {
			seen[node.Element] = true
			result = append(result, node)
		}
	}

	return result
}

// MultiThreadedBFS performs BFS using multiple goroutines for better performance
// This is optimized for finding multiple paths
func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Check if target exists
	if _, exists := elements[target]; !exists {
		fmt.Printf("Target element '%s' does not exist in the database\n", target)
		return [][]model.Node{}, 0
	}

	baseElements := getBaseElements(elements)

	// Handle case where target is a base element
	if isBaseElement(target, baseElements) {
		fmt.Printf("Target '%s' is a base element, returning direct path\n", target)
		return [][]model.Node{{
			{Element: target},
		}}, 0
	}

	// Channels for communication between goroutines
	resultChan := make(chan []model.Node, maxResults)
	visitedNodesChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex // For synchronizing access to shared data

	// Shared maps for visited elements and paths
	visited := make(map[string]bool)
	pathMap := make(map[string][]model.Node)

	// Initialize base elements in shared maps
	for _, elem := range baseElements {
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem},
		}
	}

	targetFound := false
	visitedNodesAfterTarget := 0

	// Start a goroutine for each base element
	for _, elem := range baseElements {
		wg.Add(1)
		go func(startElement string) {
			defer wg.Done()

			localQueue := list.New()
			localQueue.PushBack(startElement)

			for localQueue.Len() > 0 {
				// Check if we have found enough results
				mu.Lock()
				if len(resultChan) >= maxResults && maxResults > 0 {
					mu.Unlock()
					return
				}
				mu.Unlock()

				current := localQueue.Front().Value.(string)
				localQueue.Remove(localQueue.Front())

				// Get a snapshot of visited elements to avoid locking during iteration
				mu.Lock()
				elementsCopy := make([]string, 0, len(visited))
				for e := range visited {
					elementsCopy = append(elementsCopy, e)
				}
				isTargetFound := targetFound
				mu.Unlock()

				for _, otherElement := range elementsCopy {
					possibleResults := combineElements(current, otherElement, elements)

					for _, result := range possibleResults {
						mu.Lock()
						alreadyVisited := visited[result]

						if !alreadyVisited {
							visited[result] = true
							if isTargetFound {
								visitedNodesAfterTarget++
							}

							path := createPath(pathMap, current, otherElement, result)
							// Validate the path
							validPath := validatePath(path, baseElements)
							if validPath {
								pathMap[result] = path
							} else {
								// If path is invalid, mark as not visited and continue
								visited[result] = false
								mu.Unlock()
								continue
							}
						}

						isTarget := result == target
						if isTarget {
							targetFound = true
						}

						mu.Unlock()

						if alreadyVisited {
							continue
						}

						if isTarget {
							mu.Lock()
							path := make([]model.Node, len(pathMap[result]))
							copy(path, pathMap[result])
							mu.Unlock()

							// Send the result through the channel
							select {
							case resultChan <- path:
								fmt.Printf("Found path to target '%s', path length: %d\n", target, len(path))
							default:
								// Channel full, skip this result
							}

							// Stop exploring if we only want the shortest path
							if singlePath {
								mu.Lock()
								if len(resultChan) > 0 {
									mu.Unlock()
									return
								}
								mu.Unlock()
							}
						}

						// Continue BFS exploration
						localQueue.PushBack(result)
					}
				}
			}
		}(elem)
	}

	// Wait for all goroutines to finish and close channels
	go func() {
		wg.Wait()
		close(resultChan)
		visitedNodesChan <- visitedNodesAfterTarget
		close(visitedNodesChan)
	}()

	// Collect results from the channel
	results := make([][]model.Node, 0, maxResults)
	for path := range resultChan {
		results = append(results, path)
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}

	totalVisited := <-visitedNodesChan
	return results, totalVisited
}
