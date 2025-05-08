package algorithm

import (
	"backend/model"
	"sync"
)

// DFS finds recipe paths from base elements to target using Depth-First Search
func DFS(elements map[string]model.Element, target string, maxResults int, singlePath bool, debug bool) ([][]model.Node, int, []string) {
	var traversalOrder []string

	// Check if target exists in the element database
	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0, nil
	}

	// Get base elements (air, water, fire, earth)
	baseElements := getBaseElements(elements)

	// Handle case where target is a base element
	if isBaseElement(target, baseElements) {
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
	visited := make(map[string]bool)
	pathMap := make(map[string][]model.Node)
	var results [][]model.Node
	visitedNodesAfterTarget := 0
	targetFound := false

	// Initialize base elements in visited and pathMap
	for _, elem := range baseElements {
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem},
		}
	}

	// DFS for each base element
	for _, startElement := range baseElements {
		if debug {
			traversalOrder = append(traversalOrder, startElement)
		}

		// Skip further exploration if we've found enough paths
		if len(results) >= maxResults && maxResults > 0 {
			break
		}

		// Start DFS from this base element
		// Create a local copy of visited for this DFS branch
		localVisited := make(map[string]bool)
		for k, v := range visited {
			localVisited[k] = v
		}

		dfsExplore(
			elements,
			startElement,
			target,
			localVisited,
			pathMap,
			&results,
			&traversalOrder,
			&visitedNodesAfterTarget,
			&targetFound,
			maxResults,
			singlePath,
			debug,
		)

		// If we only want one path and found it, stop searching
		if singlePath && len(results) > 0 {
			break
		}
	}

	return results, visitedNodesAfterTarget, traversalOrder
}

// Helper function for DFS exploration
func dfsExplore(
	elements map[string]model.Element,
	current string,
	target string,
	visited map[string]bool,
	pathMap map[string][]model.Node,
	results *[][]model.Node,
	traversalOrder *[]string,
	visitedNodesAfterTarget *int,
	targetFound *bool,
	maxResults int,
	singlePath bool,
	debug bool,
) bool {
	// Stop if we've found enough paths
	if len(*results) >= maxResults && maxResults > 0 {
		return true
	}

	// Try combining current element with all visited elements
	for otherElement := range visited {
		possibleResults := combineElements(current, otherElement, elements)

		for _, result := range possibleResults {
			if visited[result] {
				continue
			}

			visited[result] = true

			if debug {
				*traversalOrder = append(*traversalOrder, result)
			}

			// Increment counter if target already found
			if *targetFound {
				*visitedNodesAfterTarget++
			}

			// Create and store the path to this element
			path := createPath(pathMap, current, otherElement, result)
			pathMap[result] = path

			// Check if we found the target element
			if result == target {
				*targetFound = true
				*results = append(*results, path)

				// If we only want one path, return immediately
				if singlePath {
					return true
				}

				// If we have reached max results, return
				if len(*results) >= maxResults && maxResults > 0 {
					return true
				}
			}

			// Continue DFS exploration recursively
			shouldStop := dfsExplore(
				elements,
				result,
				target,
				visited,
				pathMap,
				results,
				traversalOrder,
				visitedNodesAfterTarget,
				targetFound,
				maxResults,
				singlePath,
				debug,
			)

			if shouldStop {
				return true
			}
		}
	}

	return false
}

// MultiThreadedDFS performs DFS using multiple goroutines for better performance
func MultiThreadedDFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	// Check if target exists
	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0
	}

	baseElements := getBaseElements(elements)

	// Handle case where target is a base element
	if isBaseElement(target, baseElements) {
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

			// Local copy of visited to avoid excessive locking
			localVisited := make(map[string]bool)
			for k, v := range visited {
				localVisited[k] = v
			}

			// Recursively explore from this starting element
			multiThreadedDFSExplore(
				elements,
				startElement, // FIXED: Used startElement instead of undefined 'result'
				target,
				localVisited,
				visited,
				pathMap,
				&mu, // FIXED: Pass pointer to mutex
				resultChan,
				&targetFound,             // FIXED: Pass pointer to targetFound
				&visitedNodesAfterTarget, // FIXED: Pass pointer to counter
				maxResults,
				singlePath,
			)
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

// Helper function for multithreaded DFS exploration
func multiThreadedDFSExplore(
	elements map[string]model.Element,
	current string,
	target string,
	localVisited map[string]bool,
	globalVisited map[string]bool,
	pathMap map[string][]model.Node,
	mu *sync.Mutex, // FIXED: Use pointer to mutex
	resultChan chan<- []model.Node,
	targetFound *bool, // FIXED: Use pointer
	visitedNodesAfterTarget *int, // FIXED: Use pointer
	maxResults int,
	singlePath bool,
) bool {
	// Check if we should stop (channel full)
	mu.Lock()
	shouldStop := len(resultChan) >= maxResults && maxResults > 0
	mu.Unlock()

	if shouldStop {
		return true
	}

	// Get a snapshot of visited elements
	elementsToTry := make([]string, 0, len(localVisited))
	for elem := range localVisited {
		elementsToTry = append(elementsToTry, elem)
	}

	// Try combining with all visited elements
	for _, otherElement := range elementsToTry {
		possibleResults := combineElements(current, otherElement, elements)

		for _, result := range possibleResults {
			// Skip if already visited locally
			if localVisited[result] {
				continue
			}

			// Mark as visited locally
			localVisited[result] = true

			// Update shared state under lock
			mu.Lock()
			isGloballyVisited := globalVisited[result] // FIXED: Use globalVisited parameter

			if !isGloballyVisited {
				globalVisited[result] = true // FIXED: Use globalVisited parameter

				// Increment counter if target already found
				if *targetFound {
					*visitedNodesAfterTarget++
				}

				// Create and store path
				path := createPath(pathMap, current, otherElement, result)
				pathMap[result] = path
			}

			isTarget := result == target
			if isTarget && !isGloballyVisited {
				*targetFound = true
			}

			// Check if channel is full
			channelFull := len(resultChan) >= maxResults && maxResults > 0
			mu.Unlock()

			if isGloballyVisited || channelFull {
				continue
			}

			if isTarget {
				mu.Lock()
				// Make a deep copy of the path to avoid concurrent modifications
				path := make([]model.Node, len(pathMap[result]))
				copy(path, pathMap[result])
				mu.Unlock()

				// Send result to channel
				select {
				case resultChan <- path:
					// Result sent successfully
				default:
					// Channel full, skip this result
				}

				// Stop if we only want one path
				if singlePath {
					return true
				}
			}

			// Continue DFS exploration recursively
			shouldStop := multiThreadedDFSExplore(
				elements,
				result,
				target,
				localVisited,
				globalVisited, // FIXED: Use globalVisited parameter
				pathMap,
				mu,
				resultChan,
				targetFound,
				visitedNodesAfterTarget,
				maxResults,
				singlePath,
			)

			if shouldStop {
				return true
			}
		}
	}

	return false
}
