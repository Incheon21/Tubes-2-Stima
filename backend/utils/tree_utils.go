package utils

import (
	"backend/internal/graph"
	"backend/model"
	"log"
)

func PathToTree(path []model.Node, elements map[string]model.Element, algorithm string) map[string]interface{} {
	if len(path) == 0 {
		return nil
	}
	targetElement := path[0].Element
	targetImagePath := path[0].ImagePath

	if len(path) == 1 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
		}
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	isBaseElement := false
	for _, base := range baseElements {
		if targetElement == base {
			isBaseElement = true
			break
		}
	}

	if isBaseElement {
		return map[string]interface{}{
			"name":          targetElement,
			"imagePath":     targetImagePath,
			"ingredients":   []interface{}{},
			"isBaseElement": true,
		}
	}

	g := CreateElementGraph(elements)
	node := g.Nodes[targetElement]

	if len(node.RecipesToMakeThisElement) == 0 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}
	ingredients := []interface{}{}
	for _, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) > 0 {
			ingredientMatches := 0
			ingredientTrees := []interface{}{}
			for _, ingredient := range recipe.Ingredients {
				for i := 1; i < len(path); i++ {
					if path[i].Element == ingredient {
						subtree := CreateSubtreeFromPath(path[i:], elements, algorithm)
						ingredientTrees = append(ingredientTrees, subtree)
						ingredientMatches++
						break
					}
				}
			}
			if ingredientMatches == len(recipe.Ingredients) {
				ingredients = ingredientTrees
				break
			}
		}
	}
	if len(ingredients) == 0 {
		visited := make(map[string]bool)
		visitedCount := 0
		var tree map[string]interface{}
		if algorithm == "bfs" {
			tree = BuildElementTreeBFS(g, targetElement, visited, &visitedCount)
			log.Printf("DEBUG: Using BFS to build fallback tree for %s", targetElement)
		} else if algorithm == "dfs" {
			tree = BuildElementTreeDFS(g, targetElement, visited, &visitedCount)
			log.Printf("DEBUG: Using DFS to build fallback tree for %s", targetElement)
		}
		return tree
	}
	return map[string]interface{}{
		"name":        targetElement,
		"imagePath":   targetImagePath,
		"ingredients": ingredients,
	}
}
func CreateSubtreeFromPath(subPath []model.Node, elements map[string]model.Element, algorithm string) map[string]interface{} {
	if len(subPath) == 0 {
		return nil
	}

	elementName := subPath[0].Element
	imagePath := subPath[0].ImagePath

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			return map[string]interface{}{
				"name":          elementName,
				"imagePath":     imagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}
		}
	}
	if len(subPath) == 1 {
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
		}
	}
	return PathToTree(subPath, elements, algorithm)
}

func ConvertPathToCompleteTree(path []model.Node, elements map[string]model.Element, visitCount *int, algorithm string) map[string]interface{} {
	if len(path) == 0 {
		return nil
	}

	*visitCount += len(path)

	targetElement := path[0].Element
	targetImagePath := path[0].ImagePath

	if len(path) == 1 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
		}
	}

	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	isBaseElement := false
	for _, base := range baseElements {
		if targetElement == base {
			isBaseElement = true
			break
		}
	}

	if isBaseElement {
		return map[string]interface{}{
			"name":          targetElement,
			"imagePath":     targetImagePath,
			"ingredients":   []interface{}{},
			"isBaseElement": true,
		}
	}

	g := CreateElementGraph(elements)
	node := g.Nodes[targetElement]

	if len(node.RecipesToMakeThisElement) == 0 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	var matchedRecipe *graph.Recipe
	var matchedIngredients []interface{}

	for _, recipe := range node.RecipesToMakeThisElement {
		ingredientMatches := 0
		ingredientTrees := make([]interface{}, 0, len(recipe.Ingredients))

		for _, ingredientName := range recipe.Ingredients {
			for i := 1; i < len(path); i++ {
				if path[i].Element == ingredientName {
					subVisitCount := 0
					subTree := ConvertPathToSubtree(path[i:], elements, &subVisitCount, algorithm)
					*visitCount += subVisitCount

					ingredientTrees = append(ingredientTrees, subTree)
					ingredientMatches++
					break
				}
			}
		}

		if ingredientMatches == len(recipe.Ingredients) {
			matchedRecipe = recipe
			matchedIngredients = ingredientTrees
			break
		}
	}

	if matchedRecipe != nil && len(matchedIngredients) == len(matchedRecipe.Ingredients) {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": matchedIngredients,
		}
	}

	var bestRecipe *graph.Recipe
	bestIngredientCount := 999

	for _, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) < bestIngredientCount {
			bestRecipe = recipe
			bestIngredientCount = len(recipe.Ingredients)
		}
	}

	ingredients := make([]interface{}, 0, len(bestRecipe.Ingredients))
	for _, ingredientName := range bestRecipe.Ingredients {
		subVisitCount := 0
		visited := make(map[string]bool)

		var ingredientTree map[string]interface{}
		if algorithm == "bfs" {
			ingredientTree = BuildElementTreeBFS(g, ingredientName, visited, &subVisitCount)
		} else if algorithm == "dfs" {
			ingredientTree = BuildElementTreeDFS(g, ingredientName, visited, &subVisitCount)
		}

		*visitCount += subVisitCount
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        targetElement,
		"imagePath":   targetImagePath,
		"ingredients": ingredients,
	}
}

func ConvertPathToSubtree(subPath []model.Node, elements map[string]model.Element, visitCount *int, algorithm string) map[string]interface{} {
	if len(subPath) == 0 {
		return nil
	}

	*visitCount += 1

	elementName := subPath[0].Element
	imagePath := subPath[0].ImagePath
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			return map[string]interface{}{
				"name":          elementName,
				"imagePath":     imagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}
		}
	}

	if len(subPath) == 1 {
		g := CreateElementGraph(elements)
		visited := make(map[string]bool)
		subVisitCount := 0
		var tree map[string]interface{}
		if algorithm == "bfs" {
			tree = BuildElementTreeBFS(g, elementName, visited, &subVisitCount)
			log.Printf("DEBUG: Using BFS for leaf node %s", elementName)
		} else if algorithm == "dfs" {
			tree = BuildElementTreeDFS(g, elementName, visited, &subVisitCount)
			log.Printf("DEBUG: Using DFS for leaf node %s", elementName)
		}
		*visitCount += subVisitCount
		return tree
	}
	return ConvertPathToCompleteTree(subPath, elements, visitCount, algorithm)
}

func GenerateTreesForRecipe(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	recipe *graph.Recipe,
	visitedNodesCount *int,
	maxCount int,
	algorithm string,
) []map[string]interface{} {
	if len(recipe.Ingredients) == 0 {
		return []map[string]interface{}{{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
		}}
	}
	baseTree := map[string]interface{}{
		"name":        elementName,
		"imagePath":   imagePath,
		"ingredients": []interface{}{},
	}
	ingredients := make([]interface{}, 0, len(recipe.Ingredients))

	for _, ingredient := range recipe.Ingredients {
		ingNode := g.Nodes[ingredient]
		if ingNode == nil {
			log.Printf("DEBUG: Ingredient %s not found in graph", ingredient)
			continue
		}
		*visitedNodesCount++
		visited := make(map[string]bool)
		ingVisitCount := 0
		var ingredientTree map[string]interface{}

		if algorithm == "bfs" {
			ingredientTree = BuildElementTreeBFS(g, ingredient, visited, &ingVisitCount)
		} else if algorithm == "dfs" {
			ingredientTree = BuildElementTreeDFS(g, ingredient, visited, &ingVisitCount)
		}
		*visitedNodesCount += ingVisitCount
		ingredients = append(ingredients, ingredientTree)
	}
	if len(ingredients) != len(recipe.Ingredients) {
		log.Printf("DEBUG: Not all ingredients could be processed for recipe %s", elementName)
		return nil
	}
	baseTree["ingredients"] = ingredients
	return []map[string]interface{}{baseTree}
}

func BuildElementTreeBFS(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
	if visited[elementName] {
		node := g.Nodes[elementName]
		return map[string]interface{}{
			"name":                elementName,
			"imagePath":           node.ImagePath,
			"isCircularReference": true,
			"ingredients":         []interface{}{},
		}
	}

	visited[elementName] = true
	*visitedCount++

	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	for _, base := range baseElements {
		if elementName == base {
			return map[string]interface{}{
				"name":          elementName,
				"imagePath":     node.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}
		}
	}

	if len(node.RecipesToMakeThisElement) == 0 {
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	var bestRecipe *graph.Recipe
	bestBaseCount := -1

	for _, recipe := range node.RecipesToMakeThisElement {
		baseCount := 0
		for _, ingredient := range recipe.Ingredients {
			for _, base := range baseElements {
				if ingredient == base {
					baseCount++
					break
				}
			}
		}

		if baseCount > bestBaseCount {
			bestBaseCount = baseCount
			bestRecipe = recipe
		}
	}

	if bestRecipe == nil && len(node.RecipesToMakeThisElement) > 0 {
		bestRecipe = node.RecipesToMakeThisElement[0]
	}

	resultTree := map[string]interface{}{
		"name":        elementName,
		"imagePath":   node.ImagePath,
		"ingredients": []interface{}{},
	}

	for _, ingredientName := range bestRecipe.Ingredients {
		childVisited := make(map[string]bool)
		for k, v := range visited {
			childVisited[k] = v
		}

		ingredientTree := BuildElementTreeBFS(g, ingredientName, childVisited, visitedCount)
		resultTree["ingredients"] = append(resultTree["ingredients"].([]interface{}), ingredientTree)
	}

	return resultTree
}

func BuildElementTreeDFS(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
	*visitedCount++
	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	for _, base := range baseElements {
		if elementName == base {
			return map[string]interface{}{
				"name":          elementName,
				"imagePath":     node.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}
		}
	}

	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	var bestRecipe *graph.Recipe
	var bestPathLength = 9999

	for _, recipe := range node.RecipesToMakeThisElement {
		totalPathLength := 0
		for _, ingredient := range recipe.Ingredients {
			if IsBaseElementName(ingredient, baseElements) {
				totalPathLength += 1
			} else if ingNode, exists := g.Nodes[ingredient]; exists {
				if len(ingNode.RecipesToMakeThisElement) > 0 {
					totalPathLength += 2
				} else {
					totalPathLength += 1
				}
			}
		}

		if totalPathLength < bestPathLength {
			bestPathLength = totalPathLength
			bestRecipe = recipe
		}
	}

	if bestRecipe == nil && len(node.RecipesToMakeThisElement) > 0 {
		bestRecipe = node.RecipesToMakeThisElement[0]
	}

	ingredients := make([]interface{}, 0, len(bestRecipe.Ingredients))
	for _, ingredientName := range bestRecipe.Ingredients {
		ingredientTree := BuildElementTreeDFS(g, ingredientName, visited, visitedCount)
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        elementName,
		"imagePath":   node.ImagePath,
		"ingredients": ingredients,
	}
}

func CreateElementGraph(elements map[string]model.Element) *graph.ElementGraph {
	return graph.NewElementGraph(elements)
}

func GetElementTreeBFS(g *graph.ElementGraph, elementName string) (map[string]interface{}, int) {
	visited := make(map[string]bool)
	visitedCount := 0

	result := BuildElementTreeBFS(g, elementName, visited, &visitedCount)
	return result, visitedCount
}
