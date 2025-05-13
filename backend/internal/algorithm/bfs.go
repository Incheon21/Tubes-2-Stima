package algorithm

import (
	"backend/internal/graph"
	"backend/model"
	"fmt"
	"log"
	"sort"
	"strconv"
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
	var completePaths [][]model.Node
	var incompletePaths [][]model.Node

	log.Printf("DEBUG: Target %s has %d recipes to explore", target, len(targetNode.RecipesToMakeThisElement))

	type queueItem struct {
		recipe *graph.Recipe
		path   []*model.Node
	}

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

		visited := make(map[string]bool)
		visited[target] = true

		for len(queue) > 0 && (len(completePaths) < maxResults || !singlePath) {
			current := queue[0]
			queue = queue[1:]

			currentRecipe := current.recipe
			currentPath := current.path

			visitedCount++

			allIngredientsAreBaseElements := true
			hasUnmakeableElement := false
			ingredientNodes := make([]*model.Node, 0, len(currentRecipe.Ingredients))

			for _, ingredient := range currentRecipe.Ingredients {
				ingredientNode := g.Nodes[ingredient]
				if ingredientNode == nil {
					hasUnmakeableElement = true
					continue
				}

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

				if !isBase && len(ingredientNode.RecipesToMakeThisElement) == 0 {
					hasUnmakeableElement = true
				}

				if !isBase && len(ingredientNode.RecipesToMakeThisElement) > 0 {
					allIngredientsAreBaseElements = false
				}
			}

			if singlePath && hasUnmakeableElement {
				continue
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

					if !hasUnmakeableElement {
						completePaths = append(completePaths, finalPath)
						log.Printf("DEBUG: Found complete path with %d steps", len(finalPath))

						if singlePath {
							return [][]model.Node{finalPath}, visitedCount
						}
					} else if !singlePath {
						incompletePaths = append(incompletePaths, finalPath)
						log.Printf("DEBUG: Found incomplete path with %d steps (has unmakeable elements)", len(finalPath))
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

				ingredientNode := g.Nodes[ingredient]
				if ingredientNode == nil || len(ingredientNode.RecipesToMakeThisElement) == 0 {
					continue
				}

				visited[ingredient] = true

				for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
					if len(subRecipe.Ingredients) == 0 {
						continue
					}

					ingredientPath := make([]*model.Node, len(newPath))
					copy(ingredientPath, newPath)

					queue = append(queue, queueItem{
						recipe: subRecipe,
						path:   ingredientPath,
					})
				}

				delete(visited, ingredient)
			}
		}
	}

	var results [][]model.Node

	if len(completePaths) > 0 {

		if singlePath && len(completePaths) > 0 {
			var bestPath []model.Node
			bestLength := int(^uint(0) >> 1)

			for _, path := range completePaths {
				if IsFullyComposablePath(path, baseElements, g) && len(path) < bestLength {
					bestPath = path
					bestLength = len(path)
					log.Printf("DEBUG: Found fully composable path with %d steps", len(path))
				}
			}

			if bestPath != nil {
				log.Printf("DEBUG: Returning best fully composable path with %d steps", len(bestPath))
				return [][]model.Node{bestPath}, visitedCount
			}

			log.Printf("DEBUG: No fully composable path found, returning first complete path")
			return [][]model.Node{completePaths[0]}, visitedCount
		}

		results = completePaths
		log.Printf("DEBUG: Returning %d complete paths", len(results))
	} else if !singlePath && len(incompletePaths) > 0 {
		results = incompletePaths
		log.Printf("DEBUG: No complete paths found, returning %d incomplete paths", len(results))
	}

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	log.Printf("DEBUG: BFS completed - found %d paths after visiting %d nodes", len(results), visitedCount)

	return results, visitedCount
}

func MultiThreadedBFS(elements map[string]model.Element, target string, maxResults int, singlePath bool) ([][]model.Node, int) {
	g := graph.NewElementGraph(elements)
	targetNode, ok := g.Nodes[target]
	if !ok {
		log.Printf("DEBUG: Target element %s not found in database", target)
		return nil, 1
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if target == base {
			log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
			return [][]model.Node{{
				{Element: target, ImagePath: targetNode.ImagePath},
			}}, 1
		}
	}

	validRecipes := make([]*graph.Recipe, 0)
	for _, r := range targetNode.RecipesToMakeThisElement {
		if len(r.Ingredients) > 0 {
			validRecipes = append(validRecipes, r)
		}
	}

	if len(validRecipes) == 0 {
		log.Printf("DEBUG: No valid recipes found for %s", target)
		return nil, 0
	}

	log.Printf("DEBUG: Starting MultiThreaded BFS for target: %s with %d recipes", target, len(validRecipes))

	for i, recipe := range validRecipes {
		log.Printf("DEBUG: Target %s Recipe %d: %v", target, i, recipe.Ingredients)
	}

	resultChan := make(chan []model.Node, maxResults*10)
	completePathChan := make(chan []model.Node, maxResults*5)
	stopChan := make(chan struct{})

	var mu sync.Mutex
	var wg sync.WaitGroup
	visitedCount := 0

	for i, recipe := range validRecipes {
		wg.Add(1)
		go func(rcp *graph.Recipe, recipeIdx int) {
			defer wg.Done()
			log.Printf("DEBUG: Goroutine %d starting with recipe: %v", recipeIdx, rcp.Ingredients)

			type queueItem struct {
				path       []model.Node
				recipe     *graph.Recipe
				deadEndIng map[string]bool
			}

			localVisited := 0
			queue := []queueItem{{
				path: []model.Node{{
					Element:     target,
					ImagePath:   targetNode.ImagePath,
					Ingredients: rcp.Ingredients,
				}},
				recipe:     rcp,
				deadEndIng: make(map[string]bool),
			}}

			visited := make(map[string]bool)

			for len(queue) > 0 {
				select {
				case <-stopChan:
					return
				default:
				}

				item := queue[0]
				queue = queue[1:]
				localVisited++

				allBase := true
				hasDeadEnd := false
				newPath := item.path
				pathSignature := GeneratePathSignature(newPath)

				nextNodes := make([]model.Node, 0, len(item.recipe.Ingredients))

				for _, ing := range item.recipe.Ingredients {
					ingNode := g.Nodes[ing]
					if ingNode == nil {
						hasDeadEnd = true
						item.deadEndIng[ing] = true
						continue
					}

					isBase := false
					for _, base := range baseElements {
						if ing == base {
							isBase = true
							break
						}
					}

					nextNodes = append(nextNodes, model.Node{
						Element:   ing,
						ImagePath: ingNode.ImagePath,
					})

					if !isBase && len(ingNode.RecipesToMakeThisElement) == 0 {
						hasDeadEnd = true
						item.deadEndIng[ing] = true
					}

					if !isBase && len(ingNode.RecipesToMakeThisElement) > 0 {
						allBase = false
					}
				}

				if singlePath && hasDeadEnd {
					continue
				}

				newPath = append(newPath, nextNodes[i])

				if allBase || (hasDeadEnd && !singlePath) {

					reversedPath := make([]model.Node, len(newPath))
					for i, j := 0, len(newPath)-1; i < len(newPath); i, j = i+1, j-1 {
						reversedPath[i] = newPath[j]
					}

					mu.Lock()
					log.Printf("DEBUG: Found complete path in goroutine %d: %s", recipeIdx, pathToString(reversedPath))

					if allBase && !hasDeadEnd {
						completePathChan <- reversedPath
						if singlePath {
							close(stopChan)
						}
					} else if !singlePath {
						resultChan <- reversedPath
					}
					mu.Unlock()
					continue
				}

				for idx, ing := range item.recipe.Ingredients {
					isBase := false
					for _, base := range baseElements {
						if ing == base {
							isBase = true
							break
						}
					}

					if isBase || item.deadEndIng[ing] {
						continue
					}

					ingNode := g.Nodes[ing]
					if ingNode == nil || len(ingNode.RecipesToMakeThisElement) == 0 {
						continue
					}

					ingRecipes := ingNode.RecipesToMakeThisElement

					if len(ingRecipes) == 0 {
						continue
					}

					log.Printf("DEBUG: Exploring ingredient %s with %d possible recipes", ing, len(ingRecipes))

					ingCount := 0
					for _, recipeIng := range item.recipe.Ingredients {
						if recipeIng == ing {
							ingCount++
						}
					}

					if ingCount > 1 && len(ingNode.RecipesToMakeThisElement) > 1 {
						log.Printf("DEBUG: Special case: %s appears %d times in recipe and has %d ways to make it",
							ing, ingCount, len(ingNode.RecipesToMakeThisElement))

						for i := 0; i < len(ingRecipes); i++ {
							recipeIdx := i
							ingRecipe := ingRecipes[recipeIdx]

							if len(ingRecipe.Ingredients) == 0 {
								continue
							}

							log.Printf("DEBUG: Trying ingredient %s recipe permutation %d: %v",
								ing, recipeIdx, ingRecipe.Ingredients)

							basePath := make([]model.Node, len(item.path))
							copy(basePath, item.path)

							nextNode := model.Node{
								Element:     ing,
								ImagePath:   ingNode.ImagePath,
								Ingredients: ingRecipe.Ingredients,
							}

							newItem := queueItem{
								path:       append(basePath, nextNode),
								recipe:     ingRecipe,
								deadEndIng: make(map[string]bool),
							}

							for k, v := range item.deadEndIng {
								newItem.deadEndIng[k] = v
							}

							positionSig := strconv.Itoa(idx)
							newPathSig := ing + ":" + pathSignature + ":" + strconv.Itoa(i) + ":" + positionSig

							if !visited[newPathSig] {
								visited[newPathSig] = true
								queue = append(queue, newItem)
								log.Printf("DEBUG: Adding unique permutation for %s (recipe %d of %d) at position %d",
									ing, i+1, len(ingRecipes), idx)
							}
						}
					} else {
						permutationSeed := (recipeIdx*31 + localVisited*17 + idx*7) % max(1, len(ingRecipes))

						for i := 0; i < len(ingRecipes); i++ {
							recipeIdx := (permutationSeed + i) % len(ingRecipes)
							ingRecipe := ingRecipes[recipeIdx]

							if len(ingRecipe.Ingredients) == 0 {
								continue
							}

							basePath := make([]model.Node, len(item.path))
							copy(basePath, item.path)

							nextNode := model.Node{
								Element:     ing,
								ImagePath:   ingNode.ImagePath,
								Ingredients: ingRecipe.Ingredients,
							}

							newItem := queueItem{
								path:       append(basePath, nextNode),
								recipe:     ingRecipe,
								deadEndIng: make(map[string]bool),
							}

							for k, v := range item.deadEndIng {
								newItem.deadEndIng[k] = v
							}

							newPathSig := ing + ":" + pathSignature + ":" + strconv.Itoa(recipeIdx)
							if !visited[newPathSig] {
								visited[newPathSig] = true
								queue = append(queue, newItem)
							}
						}
					}
				}
			}

			mu.Lock()
			visitedCount += localVisited
			mu.Unlock()
		}(recipe, i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(completePathChan)
		close(resultChan)
	}()

	results := [][]model.Node{}
	seenSignatures := map[string]bool{}

	completePaths := [][]model.Node{}
collectLoop:
	for {
		select {
		case path, ok := <-completePathChan:
			if !ok {
				break collectLoop
			}
			sig := GeneratePathSignature(path)
			if !seenSignatures[sig] {
				seenSignatures[sig] = true
				completePaths = append(completePaths, path)

				if !singlePath && maxResults > 0 && len(completePaths) >= maxResults {
					break collectLoop
				}
			}
		case <-done:
			break collectLoop
		case <-stopChan:
			break collectLoop
		}
	}

	log.Printf("DEBUG: Initial collection found %d paths for %s", len(completePaths), target)

	uniqueRecipeSignatures := make(map[string]bool)
	diversePaths := make([][]model.Node, 0)

	for _, path := range completePaths {
		recipeSignature := generateRecipeSignature(path)
		if !uniqueRecipeSignatures[recipeSignature] {
			uniqueRecipeSignatures[recipeSignature] = true
			diversePaths = append(diversePaths, path)
			log.Printf("DEBUG: Added diverse path: %s with recipe signature: %s",
				pathToString(path), recipeSignature)
		}
	}

	log.Printf("DEBUG: Found %d unique recipe combinations after filtering", len(diversePaths))

	if len(diversePaths) > 0 {
		completePaths = diversePaths
	}

	if singlePath && len(completePaths) > 0 {
		var composablePaths [][]model.Node
		log.Printf("DEBUG: Checking %d complete paths for full composability", len(completePaths))

		for _, path := range completePaths {
			if IsFullyComposablePath(path, baseElements, g) {
				log.Printf("DEBUG: Found fully composable path with %d steps", len(path))
				composablePaths = append(composablePaths, path)
			} else {
				log.Printf("DEBUG: Rejecting path with unmakeable elements (%d steps)", len(path))
			}
		}

		if len(composablePaths) > 0 {
			sort.Slice(composablePaths, func(i, j int) bool {
				return len(composablePaths[i]) < len(composablePaths[j])
			})

			middleIndex := len(composablePaths) / 2
			selectedPath := composablePaths[middleIndex]

			log.Printf("DEBUG: Selected middle fully composable path with %d steps (path %d of %d)",
				len(selectedPath), middleIndex+1, len(composablePaths))

			return [][]model.Node{selectedPath}, visitedCount
		}

		log.Printf("DEBUG: No fully composable paths found, trying to find a best effort path")
		var bestPath []model.Node
		var bestScore int = -1

		for _, path := range completePaths {
			score := scorePathTraceability(path, baseElements, g)
			if score > bestScore {
				bestScore = score
				bestPath = path
			}
		}

		if bestPath != nil {
			log.Printf("DEBUG: Found best effort path with traceability score %d", bestScore)
			return [][]model.Node{bestPath}, visitedCount
		}

		log.Printf("DEBUG: No good path found, falling back to first complete path")
		return [][]model.Node{completePaths[0]}, visitedCount
	}

	if !singlePath {
		sort.Slice(completePaths, func(i, j int) bool {
			iComposable := IsFullyComposablePath(completePaths[i], baseElements, g)
			jComposable := IsFullyComposablePath(completePaths[j], baseElements, g)

			if iComposable != jComposable {
				return iComposable
			}

			return len(completePaths[i]) < len(completePaths[j])
		})

		results = completePaths
		if maxResults > 0 && len(results) > maxResults {
			results = results[:maxResults]
		}
	}

	if !singlePath && (maxResults == 0 || len(results) < maxResults) {
		remainingLimit := 0
		if maxResults > 0 {
			remainingLimit = maxResults - len(results)
		}

	incompleteLoop:
		for {
			select {
			case path, ok := <-resultChan:
				if !ok {
					break incompleteLoop
				}
				sig := GeneratePathSignature(path)
				if !seenSignatures[sig] {
					seenSignatures[sig] = true
					results = append(results, path)
					if remainingLimit > 0 && len(results) >= maxResults {
						break incompleteLoop
					}
				}
			case <-done:
				break incompleteLoop
			default:
				if len(results) > 0 {
					break incompleteLoop
				}
			}
		}
	}

	select {
	case <-done:
	default:
		select {
		case <-stopChan:
		default:
			close(stopChan)
		}
		<-done
	}

	if visitedCount == 0 && len(results) > 0 {
		visitCount := 0
		for _, path := range results {
			visitCount += len(path)

			baseElements := []string{"Water", "Fire", "Earth", "Air"}
			for _, node := range path {
				isBase := false
				for _, base := range baseElements {
					if node.Element == base {
						isBase = true
						break
					}
				}
				if !isBase && len(node.Ingredients) > 0 {
					visitCount += len(node.Ingredients)
				}
			}
		}
		visitedCount = visitCount
		log.Printf("DEBUG: Corrected visitedCount from 0 to %d based on found paths", visitedCount)
	}
	return results, visitedCount
}

func pathToString(path []model.Node) string {
	elements := make([]string, len(path))
	for i, node := range path {
		elemStr := node.Element
		if len(node.Ingredients) > 0 {
			ingredStr := strings.Join(node.Ingredients, "+")
			elemStr = fmt.Sprintf("%s(%s)", elemStr, ingredStr)
		}
		elements[i] = elemStr
	}
	return strings.Join(elements, " -> ")
}

func generateRecipeSignature(path []model.Node) string {
	var sig strings.Builder

	for _, node := range path {
		if len(node.Ingredients) > 0 {
			sortedIngs := make([]string, len(node.Ingredients))
			copy(sortedIngs, node.Ingredients)
			sort.Strings(sortedIngs)

			sig.WriteString(node.Element)
			sig.WriteString("(")
			sig.WriteString(strings.Join(sortedIngs, "+"))
			sig.WriteString("),")
		}
	}

	return sig.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsFullyComposablePath(path []model.Node, baseElements []string, g *graph.ElementGraph) bool {
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	processingStack := make(map[string]bool)

	validityCache := make(map[string]bool)

	var isElementTraceable func(element string) bool

	isElementTraceable = func(element string) bool {
		if baseMap[element] {
			return true
		}

		if result, exists := validityCache[element]; exists {
			return result
		}

		if processingStack[element] {
			log.Printf("DEBUG: Circular reference detected for element %s", element)
			validityCache[element] = false
			return false
		}

		elementNode := g.Nodes[element]
		if elementNode == nil {
			log.Printf("DEBUG: Element %s not found in graph", element)
			validityCache[element] = false
			return false
		}

		if len(elementNode.RecipesToMakeThisElement) == 0 {
			log.Printf("DEBUG: Element %s has no recipes, not traceable", element)
			validityCache[element] = false
			return false
		}

		processingStack[element] = true
		defer delete(processingStack, element)

		recipeValid := false
		for _, recipe := range elementNode.RecipesToMakeThisElement {
			if len(recipe.Ingredients) == 0 {
				continue
			}

			allIngredientsTraceable := true
			for _, ing := range recipe.Ingredients {
				if !isElementTraceable(ing) {
					allIngredientsTraceable = false
					break
				}
			}

			if allIngredientsTraceable {
				recipeValid = true
				break
			}
		}

		if !recipeValid {
			log.Printf("DEBUG: Element %s has no valid recipes (all lead to unmakeable elements)", element)
			validityCache[element] = false
			return false
		}

		validityCache[element] = true
		return true
	}

	for _, node := range path {
		if baseMap[node.Element] {
			continue
		}

		if !isElementTraceable(node.Element) {
			log.Printf("DEBUG: Path element %s cannot be traced to base elements", node.Element)
			return false
		}
	}

	return true
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

func IsElementTraceable(element string, baseElements []string, g *graph.ElementGraph) bool {
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	if baseMap[element] {
		return true
	}

	validityCache := make(map[string]bool)
	processingStack := make(map[string]bool)

	var isTraceable func(string) bool
	isTraceable = func(elem string) bool {
		if baseMap[elem] {
			return true
		}

		if result, exists := validityCache[elem]; exists {
			return result
		}

		if processingStack[elem] {
			validityCache[elem] = false
			return false
		}

		elementNode := g.Nodes[elem]
		if elementNode == nil {
			validityCache[elem] = false
			return false
		}

		if len(elementNode.RecipesToMakeThisElement) == 0 {
			validityCache[elem] = false
			return false
		}

		processingStack[elem] = true
		defer delete(processingStack, elem)

		recipeValid := false
		for _, recipe := range elementNode.RecipesToMakeThisElement {
			if len(recipe.Ingredients) == 0 {
				continue
			}

			allIngredientsTraceable := true
			for _, ing := range recipe.Ingredients {
				if !isTraceable(ing) {
					allIngredientsTraceable = false
					break
				}
			}

			if allIngredientsTraceable {
				recipeValid = true
				break
			}
		}

		validityCache[elem] = recipeValid
		return recipeValid
	}

	return isTraceable(element)
}

func scorePathTraceability(path []model.Node, baseElements []string, g *graph.ElementGraph) int {
	baseMap := make(map[string]bool)
	for _, base := range baseElements {
		baseMap[base] = true
	}

	traceableCount := 0
	unmakeableCount := 0

	for _, node := range path {
		if baseMap[node.Element] {
			traceableCount++
		} else if IsElementTraceable(node.Element, baseElements, g) {
			traceableCount++
		} else {
			unmakeableCount++
		}
	}

	if unmakeableCount == 0 {
		return 1000 + traceableCount
	}

	return traceableCount - (unmakeableCount * 10)
}
