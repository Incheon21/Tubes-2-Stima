package algorithm

import (
	"backend/model"
	"sync"
)

// DFS performs Depth First Search to find recipes for the target element
func DFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Check if target exists
	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0
	}

	baseElements := getBaseElements(elements)

	// Initialize result array
	var results [][]model.Node

	// Initialize visited set and path tracking
	visited := make(map[string]bool)
	visitedNodes := 0

	// If the target is a base element, return immediately
	if isBaseElement(target, baseElements) {
		return [][]model.Node{{
			{Element: target},
		}}, 1
	}

	// For each base element, start DFS
	for _, elem := range baseElements {
		// Reset visited for each new search
		for k := range visited {
			delete(visited, k)
		}

		// Mark base element as visited
		visited[elem] = true
		visitedNodes++

		// Start DFS with this element
		currentPath := []model.Node{{Element: elem}}
		dfsSearch(elements, target, elem, visited, &visitedNodes, currentPath, &results, maxResults, singlePath, baseElements)

		// Check if we have enough results
		if (singlePath && len(results) > 0) || len(results) >= maxResults {
			break
		}
	}

	return results, visitedNodes
}

func dfsSearch(
	elements map[string]model.Element,
	target string,
	current string,
	visited map[string]bool,
	visitedNodes *int,
	currentPath []model.Node,
	results *[][]model.Node,
	maxResults int,
	singlePath bool,
	baseElements []string,
) bool {
	if current == target {
		pathCopy := make([]model.Node, len(currentPath))
		copy(pathCopy, currentPath)
		*results = append(*results, pathCopy)

		return singlePath || len(*results) >= maxResults
	}

	for visitedElem := range visited {
		possibleResults := combineElements(current, visitedElem, elements)

		for _, result := range possibleResults {
			if visited[result] {
				continue
			}

			visited[result] = true
			(*visitedNodes)++

			// Add to path
			currentPath = append(currentPath, model.Node{
				Element:     result,
				Ingredients: []string{current, visitedElem},
			})

			//recursive DFS
			if dfsSearch(elements, target, result, visited, visitedNodes, currentPath, results, maxResults, singlePath, baseElements) {
				return true
			}

			//backtrack
			currentPath = currentPath[:len(currentPath)-1]
		}
	}

	return false
}

func MultiThreadedDFS(elements map[string]model.Element, target string, maxResults int) ([][]model.Node, int) {
	baseElements := getBaseElements(elements)

	var results [][]model.Node
	resultMutex := sync.Mutex{}
	visitedNodesMutex := sync.Mutex{}
	visitedNodes := 0

	var wg sync.WaitGroup

	doneChan := make(chan struct{})

	for _, elem := range baseElements {
		wg.Add(1)
		go func(startElem string) {
			defer wg.Done()

			localVisited := make(map[string]bool)
			localVisited[startElem] = true

			localVisitedCount := 1

			currentPath := []model.Node{{Element: startElem}}

			select {
			case <-doneChan:
				return
			default:
			}

			localResults := [][]model.Node{}

			multiThreadDfsSearch(elements, target, startElem, localVisited, &localVisitedCount,
				currentPath, &localResults, maxResults, false, baseElements, doneChan)

			resultMutex.Lock()
			results = append(results, localResults...)
			if len(results) >= maxResults {
				close(doneChan)
			}
			resultMutex.Unlock()

			visitedNodesMutex.Lock()
			visitedNodes += localVisitedCount
			visitedNodesMutex.Unlock()
		}(elem)
	}

	wg.Wait()

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, visitedNodes
}

func multiThreadDfsSearch(
	elements map[string]model.Element,
	target string,
	current string,
	visited map[string]bool,
	visitedNodes *int,
	currentPath []model.Node,
	results *[][]model.Node,
	maxResults int,
	singlePath bool,
	baseElements []string,
	doneChan chan struct{},
) bool {
	select {
	case <-doneChan:
		return true
	default:
	}

	if current == target {
		pathCopy := make([]model.Node, len(currentPath))
		copy(pathCopy, currentPath)
		*results = append(*results, pathCopy)

		if len(*results) >= maxResults {
			return true
		}
	}

	for visitedElem := range visited {
		possibleResults := combineElements(current, visitedElem, elements)

		for _, result := range possibleResults {
			if visited[result] {
				continue
			}

			visited[result] = true
			(*visitedNodes)++

			newPath := make([]model.Node, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, model.Node{
				Element:     result,
				Ingredients: []string{current, visitedElem},
			})

			// Recursive DFS
			if multiThreadDfsSearch(elements, target, result, visited, visitedNodes, newPath, results, maxResults, singlePath, baseElements, doneChan) {
				return true
			}
		}
	}

	return false
}
