package algorithm

import (
	"backend/model"
	"container/list"
	"fmt"
)

// BidirectionalBFS finds recipe paths from base elements to target using Bidirectional BFS
func BidirectionalBFS(elements map[string]model.Element, target string, maxResults int) ([][]model.Node, int) {
	debug := false
	singlePath := maxResults == 1  // Jika maxResults = 1, gunakan singlePath = true
	var traversalOrder []string    // Kita tetap membuat traversalOrder untuk kompatibilitas internal

	// Validate target exists
	if _, exists := elements[target]; !exists {
		fmt.Printf("Target element '%s' does not exist\n", target)
		return nil, 0
	}

	// Get base elements and check if target is base
	baseElements := getBaseElements(elements)
	if isBaseElement(target, baseElements) {
		path := []model.Node{{Element: target}}
		return [][]model.Node{path}, 0
	}

	// Check if target can be created
	if !canElementBeCreated(target, elements) {
		fmt.Printf("Target '%s' cannot be created\n", target)
		return nil, 0
	}

	// Initialize search structures
	forwardQueue, forwardVisited, forwardPaths := initSearch(baseElements, elements, "F:", debug, &traversalOrder)
	backwardQueue, backwardVisited, backwardPaths := initSearch([]string{target}, elements, "B:", debug, &traversalOrder)

	// Main search loop
	results, visitedAfterIntersection := executeBidirectionalSearch(
		forwardQueue, forwardVisited, forwardPaths,
		backwardQueue, backwardVisited, backwardPaths,
		elements, baseElements, target, maxResults, singlePath,
		debug, &traversalOrder,
	)

	// Fallback to standard BFS if no results
	if len(results) == 0 {
		fmt.Printf("Falling back to BFS for target '%s'\n", target)
		bfsResults, bfsVisited := BFS(elements, target, maxResults, singlePath)
		return bfsResults, bfsVisited
	}

	return results, visitedAfterIntersection
}

// Helper functions

func handleBaseElement(target string, imagePath string, debug bool) ([][]model.Node, int, []string) {
	fmt.Printf("Target '%s' is base element\n", target)
	path := []model.Node{{Element: target}}
	if debug {
		return [][]model.Node{path}, 0, []string{target}
	}
	return [][]model.Node{path}, 0, nil
}

func canElementBeCreated(target string, elements map[string]model.Element) bool {
	targetElement, exists := elements[target]
	if !exists {
		return false
	}
	
	// Periksa jika elemen memiliki setidaknya satu resep
	return len(targetElement.Recipes) > 0
}

func initSearch(startElements []string, elements map[string]model.Element, prefix string, debug bool, traversalOrder *[]string) (*list.List, map[string]bool, map[string][]model.Node) {
	queue := list.New()
	visited := make(map[string]bool)
	paths := make(map[string][]model.Node)

	for _, elemName := range startElements {
		queue.PushBack(elemName)
		visited[elemName] = true
		
		paths[elemName] = []model.Node{{Element: elemName}}
		
		if debug {
			*traversalOrder = append(*traversalOrder, prefix+elemName)
		}
	}

	return queue, visited, paths
}

func executeBidirectionalSearch(
	forwardQueue *list.List, forwardVisited map[string]bool, forwardPaths map[string][]model.Node,
	backwardQueue *list.List, backwardVisited map[string]bool, backwardPaths map[string][]model.Node,
	elements map[string]model.Element, baseElements []string, target string,
	maxResults int, singlePath bool, debug bool, traversalOrder *[]string,
) ([][]model.Node, int) {
	var results [][]model.Node
	visitedAfterIntersection := 0
	maxIterations := 5000

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Forward search
		if newResults := processSearchLevel(
			forwardQueue, forwardVisited, backwardVisited, forwardPaths,
			elements, baseElements, "F:", debug, traversalOrder,
			len(results) > 0, &visitedAfterIntersection,
		); len(newResults) > 0 {
			results = append(results, newResults...)
			if shouldReturn(results, maxResults, singlePath) {
				return results, visitedAfterIntersection
			}
		}

		// Backward search
		if newResults := processSearchLevel(
			backwardQueue, backwardVisited, forwardVisited, backwardPaths,
			elements, baseElements, "B:", debug, traversalOrder,
			len(results) > 0, &visitedAfterIntersection,
		); len(newResults) > 0 {
			results = append(results, newResults...)
			if shouldReturn(results, maxResults, singlePath) {
				return results, visitedAfterIntersection
			}
		}

		// Termination conditions
		if forwardQueue.Len() == 0 || backwardQueue.Len() == 0 {
			break
		}
		if len(results) >= maxResults && maxResults > 0 {
			break
		}
	}

	return results, visitedAfterIntersection
}

func processSearchLevel(
	queue *list.List, visited, otherVisited map[string]bool,
	paths map[string][]model.Node, elements map[string]model.Element,
	baseElements []string, prefix string, debug bool,
	traversalOrder *[]string, intersectionFound bool,
	visitedAfterIntersection *int,
) [][]model.Node {
	var results [][]model.Node
	levelSize := queue.Len()

	for i := 0; i < levelSize; i++ {
		current := queue.Front().Value.(string)
		queue.Remove(queue.Front())

		if debug {
			*traversalOrder = append(*traversalOrder, prefix+current)
		}

		// Explore all possible combinations
		for otherElement := range elements {
			for _, result := range combineElements(current, otherElement, elements) {
				// Skip if we already have a path and it's the same as current
				if existingPath, exists := paths[result]; exists {
					currentPath := createPath(paths, current, otherElement, result)
					if pathsEqual(existingPath, currentPath) {
						continue
					}
				}

				// Create and validate path
				newPath := createPath(paths, current, otherElement, result)
				if !validatePath(newPath, baseElements) {
					continue
				}

				// Store path (even if already exists, we want alternatives)
				paths[result] = newPath

				// Mark as visited if new
				if !visited[result] {
					visited[result] = true
					if intersectionFound {
						*visitedAfterIntersection++
					}
				}

				// Always add to queue to explore further
				queue.PushBack(result)

				// Check for intersection
				if otherVisited[result] {
					forwardPath := paths[result]
					if backwardPath, exists := paths[result]; exists {
						reversedBackwardPath := reversePathFromTarget(backwardPath)
						combinedPath := combinePaths(forwardPath, reversedBackwardPath)
						if validateCompletePath(combinedPath, baseElements) {
							results = append(results, combinedPath)
						}
					}
				}
			}
		}
	}

	return results
}

func shouldReturn(results [][]model.Node, maxResults int, singlePath bool) bool {
	return (singlePath && len(results) > 0) || (maxResults > 0 && len(results) >= maxResults)
}

func fallbackToBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool, debug bool, traversalOrder []string) ([][]model.Node, int, []string) {
	fmt.Printf("Falling back to BFS for target '%s'\n", target)
	bfsResults, bfsVisited := BFS(elements, target, maxResults, singlePath)
	return bfsResults, bfsVisited, traversalOrder
}

// Path manipulation functions

func pathsEqual(a, b []model.Node) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Element != b[i].Element {
			return false
		}
	}
	return true
}

func reversePathFromTarget(path []model.Node) []model.Node {
	reversed := make([]model.Node, len(path))
	for i := 0; i < len(path); i++ {
		reversed[i] = path[len(path)-1-i]
	}
	return reversed
}

func combinePaths(forward, backward []model.Node) []model.Node {
	if len(backward) > 0 {
		backward = backward[1:]
	}
	return append(forward, backward...)
}

func validateCompletePath(path []model.Node, baseElements []string) bool {
	if len(path) < 2 {
		return false
	}
	return validatePath(path[:len(path)/2], baseElements) &&
		validatePath(reversePathFromTarget(path[len(path)/2:]), baseElements)
}

// Fungsi yang perlu diimplementasikan



func validatePath(path []model.Node, baseElements []string) bool {
	if len(path) == 0 {
		return false
	}
	
	// Periksa apakah elemen pertama adalah elemen dasar
	firstElement := path[0].Element
	if !isBaseElement(firstElement, baseElements) {
		return false
	}
	
	// Periksa apakah setiap node memiliki ingredient yang valid
	for i := 1; i < len(path); i++ {
		if len(path[i].Ingredients) != 2 {
			return false
		}
	}
	
	return true
}

// Element utility functions

func isBaseElement(element string, baseElements []string) bool {
	for _, base := range baseElements {
		if base == element {
			return true
		}
	}
	return false
}

func combineElements(elem1, elem2 string, elements map[string]model.Element) []string {
	var results []string
	for name, element := range elements {
		for _, recipe := range element.Recipes {
			if len(recipe.Ingredients) == 2 &&
				((recipe.Ingredients[0] == elem1 && recipe.Ingredients[1] == elem2) ||
					(recipe.Ingredients[0] == elem2 && recipe.Ingredients[1] == elem1)) {
				results = append(results, name)
				break
			}
		}
	}
	return results
}

func getBaseElements(elements map[string]model.Element) []string {
	var bases []string
	for name, element := range elements {
		if element.Tier == 1 && (name == "Water" || name == "Fire" || name == "Earth" || name == "Air") {
			bases = append(bases, name)
		}
	}
	return bases
}

