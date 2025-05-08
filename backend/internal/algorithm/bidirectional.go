package algorithm

import (
	"backend/model"
	"container/list"
)

func Bidirectional(elements map[string]model.Element, target string, maxResults int, debug bool) ([][]model.Node, int, []string) {
	var traversalOrder []string

	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0, nil
	}

	baseElements := getBaseElements(elements)

	if isBaseElement(target, baseElements) {
		return [][]model.Node{{
			{Element: target},
		}}, 1, nil
	}

	forwardQueue := list.New()
	backwardQueue := list.New()

	forwardVisited := make(map[string][]model.Node)
	backwardVisited := make(map[string][]model.Node)

	for _, elem := range baseElements {
		forwardQueue.PushBack(elem)
		forwardVisited[elem] = []model.Node{{Element: elem}}
	}

	backwardQueue.PushBack(target)
	backwardVisited[target] = []model.Node{{Element: target}}

	visitedNodes := len(baseElements) + 1

	var results [][]model.Node

	// Bidirectional BFS
	for forwardQueue.Len() > 0 && backwardQueue.Len() > 0 {
		intersection := findIntersection(forwardVisited, backwardVisited)
		if len(intersection) > 0 {
			for _, meetingPoint := range intersection {
				forwardPath := forwardVisited[meetingPoint]
				backwardPath := backwardVisited[meetingPoint]

				reversedPath := reversePath(backwardPath)

				fullPath := combinePaths(forwardPath, reversedPath)
				results = append(results, fullPath)

				if len(results) >= maxResults {
					return results, visitedNodes, traversalOrder
				}
			}
		}

		if forwardQueue.Len() <= backwardQueue.Len() {
			expandBidirectional(elements, forwardQueue, forwardVisited, &visitedNodes, true)
		} else {
			expandBidirectional(elements, backwardQueue, backwardVisited, &visitedNodes, false)
		}
	}

	return results, visitedNodes, traversalOrder
}

func expandBidirectional(
	elements map[string]model.Element,
	queue *list.List,
	visited map[string][]model.Node,
	visitedNodes *int,
	isForward bool,
) {
	levelSize := queue.Len()
	for i := 0; i < levelSize; i++ {
		current := queue.Front().Value.(string)
		queue.Remove(queue.Front())

		var possibleCombinations []string

		if isForward {
			for visitedElem := range visited {
				results := combineElements(current, visitedElem, elements)
				possibleCombinations = append(possibleCombinations, results...)
			}
		} else {
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

			if isForward {
				currentPath := visited[current]
				newPath := make([]model.Node, len(currentPath))
				copy(newPath, currentPath)
				newPath = append(newPath, model.Node{Element: newElem})
				visited[newElem] = newPath
			} else {
				backPath := []model.Node{{Element: newElem}, {Element: current}}
				visited[newElem] = backPath
			}
		}
	}
}

func findIntersection(forward, backward map[string][]model.Node) []string {
	var intersection []string

	for elem := range forward {
		if _, exists := backward[elem]; exists {
			intersection = append(intersection, elem)
		}
	}

	return intersection
}

func reversePath(path []model.Node) []model.Node {
	reversed := make([]model.Node, len(path))

	for i, j := 0, len(path)-1; j >= 0; i, j = i+1, j-1 {
		reversed[i] = path[j]
	}

	return reversed
}

func combinePaths(forward, backward []model.Node) []model.Node {
	combined := make([]model.Node, 0, len(forward)+len(backward)-1)
	combined = append(combined, forward...)

	if len(backward) > 1 {
		combined = append(combined, backward[1:]...)
	}

	return combined
}

func getBaseElements(elements map[string]model.Element) []string {
	var baseElements []string

	for name, element := range elements {
		if element.Tier == 1 && (name == "Water" || name == "Fire" || name == "Earth" || name == "Air") {
			baseElements = append(baseElements, name)
		}
	}

	return baseElements
}

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
