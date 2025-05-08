package algorithm

import (
	"backend/model"
	"container/list"
)

// Bidirectional performs bidirectional search to find recipes
func Bidirectional(elements map[string]model.Element, target string, maxResults int) ([][]model.Node, int) {
	// Check if target exists
	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0
	}

	baseElements := getBaseElements(elements)

	// If the target is a base element, return immediately
	if isBaseElement(target, baseElements) {
		return [][]model.Node{{
			{Element: target},
		}}, 1
	}

	// Initialize forward and backward queues
	forwardQueue := list.New()
	backwardQueue := list.New()

	// Track visited nodes from both directions
	forwardVisited := make(map[string][]model.Node)
	backwardVisited := make(map[string][]model.Node)

	// Start from base elements (forward direction)
	for _, elem := range baseElements {
		forwardQueue.PushBack(elem)
		forwardVisited[elem] = []model.Node{{Element: elem}}
	}

	// Start from target (backward direction)
	backwardQueue.PushBack(target)
	backwardVisited[target] = []model.Node{{Element: target}}

	// Track visited nodes count
	visitedNodes := len(baseElements) + 1 // base elements + target

	// Results storage
	var results [][]model.Node

	// Bidirectional BFS
	for forwardQueue.Len() > 0 && backwardQueue.Len() > 0 {
		// Check for intersection after each expansion
		intersection := findIntersection(forwardVisited, backwardVisited)
		if len(intersection) > 0 {
			// Found paths, combine them
			for _, meetingPoint := range intersection {
				forwardPath := forwardVisited[meetingPoint]
				backwardPath := backwardVisited[meetingPoint]

				// Reverse the backward path
				reversedPath := reversePath(backwardPath)

				// Combine paths
				fullPath := combinePaths(forwardPath, reversedPath)
				results = append(results, fullPath)

				if len(results) >= maxResults {
					return results, visitedNodes
				}
			}
		}

		// Expand forward (from base elements toward target)
		if forwardQueue.Len() <= backwardQueue.Len() {
			expandBidirectional(elements, forwardQueue, forwardVisited, &visitedNodes, true)
		} else {
			// Expand backward (from target toward base elements)
			expandBidirectional(elements, backwardQueue, backwardVisited, &visitedNodes, false)
		}
	}

	return results, visitedNodes
}

// expandBidirectional expands one level in either the forward or backward direction
func expandBidirectional(
	elements map[string]model.Element,
	queue *list.List,
	visited map[string][]model.Node,
	visitedNodes *int,
	isForward bool,
) {
	// Process one level
	levelSize := queue.Len()
	for i := 0; i < levelSize; i++ {
		// Dequeue element
		current := queue.Front().Value.(string)
		queue.Remove(queue.Front())

		// Get all elements we've visited so far
		var possibleCombinations []string

		if isForward {
			// Forward direction: try combining with all visited elements
			for visitedElem := range visited {
				// Try combining current with visitedElem
				results := combineElements(current, visitedElem, elements)
				possibleCombinations = append(possibleCombinations, results...)
			}
		} else {
			// Backward direction: find all elements that can produce the current element
			for _, elem := range elements {
				for _, recipe := range elem.Recipes {
					if len(recipe.Ingredients) == 2 && recipe.Ingredients[0] == current {
						possibleCombinations = append(possibleCombinations, recipe.Ingredients[1])
					} else if len(recipe.Ingredients) == 2 && recipe.Ingredients[1] == current {
						possibleCombinations = append(possibleCombinations, recipe.Ingredients[0])
					}
				}
			}
		}

		// Process all new combinations
		for _, newElem := range possibleCombinations {
			// Skip if already visited
			if _, exists := visited[newElem]; exists {
				continue
			}

			// Mark as visited and add to queue
			queue.PushBack(newElem)
			(*visitedNodes)++

			// Create path for this new element
			if isForward {
				// Forward path: append new element to current's path
				currentPath := visited[current]
				newPath := make([]model.Node, len(currentPath))
				copy(newPath, currentPath)
				newPath = append(newPath, model.Node{Element: newElem})
				visited[newElem] = newPath
			} else {
				// Backward path: append current to new element's expected path
				backPath := []model.Node{{Element: newElem}, {Element: current}}
				visited[newElem] = backPath
			}
		}
	}
}

// findIntersection finds elements that have been visited from both directions
func findIntersection(forward, backward map[string][]model.Node) []string {
	var intersection []string

	// Find elements that appear in both maps
	for elem := range forward {
		if _, exists := backward[elem]; exists {
			intersection = append(intersection, elem)
		}
	}

	return intersection
}

// reversePath reverses the order of nodes in a path
func reversePath(path []model.Node) []model.Node {
	reversed := make([]model.Node, len(path))

	for i, j := 0, len(path)-1; j >= 0; i, j = i+1, j-1 {
		reversed[i] = path[j]
	}

	return reversed
}

// combinePaths combines forward and backward paths into a complete path
func combinePaths(forward, backward []model.Node) []model.Node {
	// Create a new combined path
	// Skip the duplicate node at the connection point
	combined := make([]model.Node, 0, len(forward)+len(backward)-1)
	combined = append(combined, forward...)

	// Skip the first node of the backward path (it's the same as the last of forward)
	if len(backward) > 1 {
		combined = append(combined, backward[1:]...)
	}

	return combined
}

// Helper functions for all algorithms

// getBaseElements returns the list of base elements (tier 1)
func getBaseElements(elements map[string]model.Element) []string {
	var baseElements []string

	for name, element := range elements {
		if element.Tier == 1 && (name == "Water" || name == "Fire" || name == "Earth" || name == "Air") {
			baseElements = append(baseElements, name)
		}
	}

	return baseElements
}

// isBaseElement checks if an element is a base element
func isBaseElement(element string, baseElements []string) bool {
	for _, base := range baseElements {
		if base == element {
			return true
		}
	}
	return false
}

// combineElements tries to combine two elements and returns all possible results
func combineElements(elem1, elem2 string, elements map[string]model.Element) []string {
	var results []string

	// Check all elements for recipes containing both elements
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
