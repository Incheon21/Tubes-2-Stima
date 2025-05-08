package algorithm

import (
	"backend/model"
	"container/list"
	"sync"
)

func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	if _, exists := elements[target]; !exists {
		return [][]model.Node{}, 0
	}

	baseElements := getBaseElements(elements)

	var results [][]model.Node

	queue := list.New()
	visited := make(map[string]bool)

	pathMap := make(map[string][]model.Node)

	for _, elem := range baseElements {
		queue.PushBack(elem)
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem},
		}
	}

	visitedNodes := len(baseElements)

	if isBaseElement(target, baseElements) {
		return [][]model.Node{{
			{Element: target},
		}}, visitedNodes
	}

	for queue.Len() > 0 {
		current := queue.Front().Value.(string)
		queue.Remove(queue.Front())

		for otherElement := range visited {
			possibleResults := combineElements(current, otherElement, elements)

			for _, result := range possibleResults {
				if visited[result] {
					continue
				}

				visited[result] = true
				visitedNodes++

				path := createPath(pathMap, current, otherElement, result)
				pathMap[result] = path

				if result == target {
					results = append(results, path)

					if singlePath || len(results) >= maxResults {
						return results, visitedNodes
					}
				} else {
					queue.PushBack(result)
				}
			}
		}
	}

	return results, visitedNodes
}

func createPath(pathMap map[string][]model.Node, element1, element2, result string) []model.Node {
	path1 := pathMap[element1]
	path2 := pathMap[element2]

	newPath := make([]model.Node, 0, len(path1)+len(path2)+1)
	newPath = append(newPath, path1...)
	newPath = append(newPath, path2...)

	newPath = append(newPath, model.Node{
		Element:     result,
		Ingredients: []string{element1, element2},
	})

	return newPath
}

func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int) ([][]model.Node, int) {

	resultChan := make(chan []model.Node, maxResults)
	visitedNodesChan := make(chan int, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex

	baseElements := getBaseElements(elements)
	visited := make(map[string]bool)
	pathMap := make(map[string][]model.Node)

	for _, elem := range baseElements {
		visited[elem] = true
		pathMap[elem] = []model.Node{
			{Element: elem},
		}
	}

	visitedNodes := len(baseElements)

	for _, elem := range baseElements {
		wg.Add(1)
		go func(startElement string) {
			defer wg.Done()

			localQueue := list.New()
			localQueue.PushBack(startElement)

			for localQueue.Len() > 0 {
				if len(resultChan) >= maxResults {
					return
				}

				current := localQueue.Front().Value.(string)
				localQueue.Remove(localQueue.Front())

				mu.Lock()
				elementsCopy := make([]string, 0, len(visited))
				for e := range visited {
					elementsCopy = append(elementsCopy, e)
				}
				mu.Unlock()

				for _, otherElement := range elementsCopy {
					possibleResults := combineElements(current, otherElement, elements)

					for _, result := range possibleResults {
						mu.Lock()
						isVisited := visited[result]
						if !isVisited {
							visited[result] = true
							visitedNodes++
							path := createPath(pathMap, current, otherElement, result)
							pathMap[result] = path
						}
						mu.Unlock()

						if isVisited {
							continue
						}

						if result == target {
							mu.Lock()
							path := pathMap[result]
							mu.Unlock()
							resultChan <- path
						} else {
							localQueue.PushBack(result)
						}
					}
				}
			}
		}(elem)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		visitedNodesChan <- visitedNodes
		close(visitedNodesChan)
	}()

	results := make([][]model.Node, 0, maxResults)
	for path := range resultChan {
		results = append(results, path)
		if len(results) >= maxResults {
			break
		}
	}

	totalVisitedNodes := <-visitedNodesChan

	return results, totalVisitedNodes
}
