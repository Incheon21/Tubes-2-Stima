package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"log"
	"sort"
	"strings"
	"sync"
)

func BFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	log.Printf("DEBUG: Starting top-down BFS for target: %s (max results: %d)", target, maxResults)

	g := graph.NewElementGraph(elements)

	targetNode, exists := g.Nodes[target]
	if !exists {
		log.Printf("DEBUG: Target element %s not found in database", target)
		return [][]model.Node{}, 0
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 0
		}
	}

	visitedCount := 0
	var results [][]model.Node

	log.Printf("DEBUG: Target %s has %d recipes to explore", target, len(targetNode.RecipesToMakeThisElement))

	type queueItem struct {
		recipe *graph.Recipe
		path   []*model.Node
	}

	visited := make(map[string]bool)

	uniquePaths := make(map[string]bool)

	for _, recipe := range targetNode.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		startPath := []*model.Node{
			{Element: target, ImagePath: targetNode.ImagePath},
		}

		queue := []queueItem{
			{recipe: recipe, path: startPath},
		}

		for k := range visited {
			delete(visited, k)
		}
		visited[target] = true

		for len(queue) > 0 && len(results) < maxResults {
			current := queue[0]
			queue = queue[1:]

			currentRecipe := current.recipe
			currentPath := current.path

			visitedCount++

			allIngredientsAreBaseElements := true
			ingredientNodes := make([]*model.Node, 0, len(currentRecipe.Ingredients))

			for _, ingredient := range currentRecipe.Ingredients {
				ingredientNode := g.Nodes[ingredient]

				ingredientNodeObj := &model.Node{
					Element:   ingredient,
					ImagePath: ingredientNode.ImagePath,
				}
				ingredientNodes = append(ingredientNodes, ingredientNodeObj)

				isBase := false
				for _, base := range baseElements {
					if ingredient == base {
						isBase = true
						break
					}
				}

				if !isBase && len(ingredientNode.RecipesToMakeThisElement) > 0 {
					allIngredientsAreBaseElements = false
				}
			}

			newPath := make([]*model.Node, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, ingredientNodes...)

			if allIngredientsAreBaseElements {
				finalPath := make([]model.Node, len(newPath))
				for i, node := range newPath {
					finalPath[i] = *node
				}

				for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
					finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
				}

				pathSignature := GeneratePathSignature(finalPath)
				if !uniquePaths[pathSignature] {
					uniquePaths[pathSignature] = true
					results = append(results, finalPath)
					log.Printf("DEBUG: Found complete path with %d steps", len(finalPath))

					if singlePath {
						break
					}
				}

				continue
			}

			for _, ingredient := range currentRecipe.Ingredients {
				isBase := false
				for _, base := range baseElements {
					if ingredient == base {
						isBase = true
						break
					}
				}

				if isBase || visited[ingredient] {
					continue
				}

				visited[ingredient] = true

				ingredientNode := g.Nodes[ingredient]
				for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
					if len(subRecipe.Ingredients) == 0 {
						continue
					}

					// Create new path for this recipe branch
					ingredientPath := make([]*model.Node, len(newPath))
					copy(ingredientPath, newPath)

					// Add to queue
					queue = append(queue, queueItem{
						recipe: subRecipe,
						path:   ingredientPath,
					})
				}

				delete(visited, ingredient)
			}
		}

		if singlePath && len(results) > 0 {
			break
		}
	}

	log.Printf("DEBUG: BFS completed - found %d paths after visiting %d nodes", len(results), visitedCount)

	return results, visitedCount
}
func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)
	targetNode, ok := g.Nodes[target]
	if !ok {

		return nil, 0
	}

	validRecipes := make([]*graph.Recipe, 0)
	for _, r := range targetNode.RecipesToMakeThisElement {
		if len(r.Ingredients) > 0 {
			validRecipes = append(validRecipes, r)
		}
	}

	resultChan := make(chan []model.Node, maxResults*10)
	var mu sync.Mutex
	var wg sync.WaitGroup
	visitedCount := 0

	for _, recipe := range validRecipes {
		wg.Add(1)
		go func(rcp *graph.Recipe) {
			defer wg.Done()

			type queueItem struct {
				path   []model.Node
				recipe *graph.Recipe
			}

			localVisited := 0
			queue := []queueItem{{
				path: []model.Node{{
					Element:     target,
					ImagePath:   targetNode.ImagePath,
					Ingredients: rcp.Ingredients,
				}},
				recipe: rcp,
			}}

			for len(queue) > 0 {
				item := queue[0]
				queue = queue[1:]
				localVisited++

				allBase := true
				newPath := item.path

				for _, ing := range item.recipe.Ingredients {
					ingNode := g.Nodes[ing]
					if ingNode == nil {
						continue
					}
					if len(ingNode.RecipesToMakeThisElement) > 0 {
						allBase = false
					}
				}

				if allBase {
					mu.Lock()
					resultChan <- newPath
					mu.Unlock()
					continue
				}

				for _, ing := range item.recipe.Ingredients {
					ingNode := g.Nodes[ing]
					if ingNode == nil {
						continue
					}
					for _, ingRecipe := range ingNode.RecipesToMakeThisElement {
						if len(ingRecipe.Ingredients) > 0 {
							newItem := queueItem{
								path:   appendPath(newPath, ing, ingNode.ImagePath, ingRecipe.Ingredients),
								recipe: ingRecipe,
							}
							queue = append(queue, newItem)
						}
					}
				}
			}
			mu.Lock()
			visitedCount += localVisited
			mu.Unlock()
		}(recipe)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := [][]model.Node{}
	seenSignatures := map[string]bool{}
	for p := range resultChan {
		sig := GeneratePathSignature(p)
		if !seenSignatures[sig] {
			seenSignatures[sig] = true
			results = append(results, p)
			if maxResults > 0 && len(results) >= maxResults {
				break
			}
		}
	}
	return results, visitedCount
}

func appendPath(path []model.Node, element, imgPath string, ingredients []string) []model.Node {
	newp := make([]model.Node, len(path), len(path)+1)
	copy(newp, path)
	newp = append(newp, model.Node{
		Element:     element,
		ImagePath:   imgPath,
		Ingredients: ingredients,
	})
	return newp
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
func GeneratePathSignature(path []model.Node) string {
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
