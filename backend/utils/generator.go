package utils

import (
	"backend/internal/graph"
	"log"
	"sort"
	"strings"
)

func GenerateAllRecipeVariations(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	maxCount int) ([]map[string]interface{}, int) {

	totalVisitedCount := 0
	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	for _, base := range baseElements {
		if elementName == base {
			return []map[string]interface{}{{
				"name":          elementName,
				"imagePath":     imagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}}, 1
		}
	}
	if node == nil || len(node.RecipesToMakeThisElement) == 0 {
		return []map[string]interface{}{{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}}, 1
	}

	allTrees := make([]map[string]interface{}, 0)
	uniqueSignatures := make(map[string]bool)

	for _, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) == 0 {
			continue
		}

		GenerateRecipeVariationsWithSubIngredients(
			g, elementName, imagePath, recipe, &allTrees, &uniqueSignatures,
			&totalVisitedCount, maxCount, 0, 3)
	}

	if len(allTrees) == 0 {
		visitCount := 0
		visited := make(map[string]bool)
		tree := BuildElementTreeDFS(g, elementName, visited, &visitCount)
		allTrees = append(allTrees, tree)
		totalVisitedCount += visitCount
	}

	if len(allTrees) > maxCount {
		allTrees = allTrees[:maxCount]
	}

	return allTrees, totalVisitedCount
}

func GenerateRecipeVariationsWithSubIngredients(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	recipe *graph.Recipe,
	allTrees *[]map[string]interface{},
	uniqueSignatures *map[string]bool,
	totalVisitedCount *int,
	maxCount int,
	currentDepth int,
	maxDepth int) {

	if currentDepth > maxDepth || (len(*allTrees) >= maxCount && maxCount > 0) {
		return
	}

	ingredientVariations := make([][]map[string]interface{}, len(recipe.Ingredients))

	for i, ingredient := range recipe.Ingredients {
		ingredientNode := g.Nodes[ingredient]
		*totalVisitedCount++

		if ingredientNode == nil {
			continue
		}

		var ingredientTrees []map[string]interface{}

		baseElements := []string{"Water", "Fire", "Earth", "Air"}
		isBase := false
		for _, base := range baseElements {
			if ingredient == base {
				isBase = true
				ingredientTrees = []map[string]interface{}{{
					"name":          ingredient,
					"imagePath":     ingredientNode.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}}
				break
			}
		}

		if !isBase {
			if len(ingredientNode.RecipesToMakeThisElement) == 0 {
				ingredientTrees = []map[string]interface{}{{
					"name":        ingredient,
					"imagePath":   ingredientNode.ImagePath,
					"ingredients": []interface{}{},
					"noRecipe":    true,
				}}
			} else if currentDepth >= maxDepth-1 {
				visited := make(map[string]bool)
				visitCount := 0
				tree := BuildElementTreeDFS(g, ingredient, visited, &visitCount)
				*totalVisitedCount += visitCount
				ingredientTrees = []map[string]interface{}{tree}
			} else {
				for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
					subVariations := make([]map[string]interface{}, 0)
					tempSigs := make(map[string]bool)

					GenerateRecipeVariationsWithSubIngredients(
						g, ingredient, ingredientNode.ImagePath, subRecipe,
						&subVariations, &tempSigs, totalVisitedCount,
						2, currentDepth+1, maxDepth)

					for _, variation := range subVariations {
						ingredientTrees = append(ingredientTrees, variation)
					}

					if len(subVariations) == 0 {
						visited := make(map[string]bool)
						visitCount := 0
						tree := BuildElementTreeDFS(g, ingredient, visited, &visitCount)
						*totalVisitedCount += visitCount
						ingredientTrees = append(ingredientTrees, tree)
					}
				}
			}
		}

		if len(ingredientTrees) == 0 {
			ingredientTrees = []map[string]interface{}{{
				"name":        ingredient,
				"imagePath":   ingredientNode.ImagePath,
				"ingredients": []interface{}{},
			}}
		}

		ingredientVariations[i] = ingredientTrees
	}

	GenerateTreeCombinations(
		g, elementName, imagePath, recipe.Ingredients,
		ingredientVariations, 0, []map[string]interface{}{},
		allTrees, uniqueSignatures, totalVisitedCount, maxCount)
}

func GenerateTreeCombinations(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	ingredientNames []string,
	ingredientVariations [][]map[string]interface{},
	currentIndex int,
	currentCombination []map[string]interface{},
	allTrees *[]map[string]interface{},
	uniqueSignatures *map[string]bool,
	totalVisitedCount *int,
	maxCount int) {

	if len(*allTrees) >= maxCount && maxCount > 0 {
		return
	}

	if currentIndex >= len(ingredientVariations) {
		tree := map[string]interface{}{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": make([]interface{}, len(currentCombination)),
		}

		for i, ingTree := range currentCombination {
			tree["ingredients"].([]interface{})[i] = ingTree
		}

		signature := GenerateDetailedTreeSignature(tree)
		if !(*uniqueSignatures)[signature] {
			(*uniqueSignatures)[signature] = true
			*allTrees = append(*allTrees, tree)

			log.Printf("DEBUG: Generated unique recipe tree variation with signature: %s", signature)
		}

		return
	}

	if len(ingredientVariations[currentIndex]) == 0 {
		GenerateTreeCombinations(
			g, elementName, imagePath, ingredientNames,
			ingredientVariations, currentIndex+1, currentCombination,
			allTrees, uniqueSignatures, totalVisitedCount, maxCount)
		return
	}

	for _, variation := range ingredientVariations[currentIndex] {
		newCombination := append(currentCombination, variation)

		GenerateTreeCombinations(
			g, elementName, imagePath, ingredientNames,
			ingredientVariations, currentIndex+1, newCombination,
			allTrees, uniqueSignatures, totalVisitedCount, maxCount)

		if len(*allTrees) >= maxCount && maxCount > 0 {
			return
		}
	}
}

func GenerateDetailedTreeSignature(tree map[string]interface{}) string {
	name := tree["name"].(string)
	ingredients, ok := tree["ingredients"].([]interface{})

	if !ok || len(ingredients) == 0 {
		return name + "|[]"
	}

	subSigs := make([]string, 0, len(ingredients))

	for _, ing := range ingredients {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			subSig := GenerateDetailedTreeSignature(ingMap)
			subSigs = append(subSigs, subSig)
		}
	}

	sort.Strings(subSigs)

	return name + "|[" + strings.Join(subSigs, ";") + "]"
}

func GenerateRecipeVariations(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	recipe *graph.Recipe,
	activeGoroutines *int,
	resultChan chan<- map[string]interface{},
	visitCountChan chan<- int,
	depth int,
	maxCount int,
) {
	if depth >= 2 {
		*activeGoroutines++
		go func() {
			visitCount := 0
			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   imagePath,
				"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
			}

			for _, ingredientName := range recipe.Ingredients {
				visited := make(map[string]bool)
				ingredientVisitCount := 0
				ingredientTree := BuildElementTreeDFS(g, ingredientName, visited, &ingredientVisitCount)
				tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingredientTree)
				visitCount += ingredientVisitCount
			}

			resultChan <- tree
			visitCountChan <- visitCount
		}()
		return
	}

	hasMultipleRecipes := false
	for _, ingredient := range recipe.Ingredients {
		if ingNode := g.Nodes[ingredient]; ingNode != nil && len(ingNode.RecipesToMakeThisElement) > 1 {
			hasMultipleRecipes = true
			break
		}
	}

	if !hasMultipleRecipes {
		*activeGoroutines++
		go func() {
			visitCount := 0
			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   imagePath,
				"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
			}

			for _, ingredientName := range recipe.Ingredients {
				visited := make(map[string]bool)
				ingredientVisitCount := 0
				ingredientTree := BuildElementTreeDFS(g, ingredientName, visited, &ingredientVisitCount)
				tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingredientTree)
				visitCount += ingredientVisitCount
			}

			resultChan <- tree
			visitCountChan <- visitCount
		}()
		return
	}

	ingredientsWithMultipleRecipes := make([]string, 0)
	for _, ingredient := range recipe.Ingredients {
		if ingNode := g.Nodes[ingredient]; ingNode != nil && len(ingNode.RecipesToMakeThisElement) > 1 {
			ingredientsWithMultipleRecipes = append(ingredientsWithMultipleRecipes, ingredient)
		}
	}

	maxVariations := maxCount / 2
	if maxVariations < 1 {
		maxVariations = 1
	}

	if len(ingredientsWithMultipleRecipes) > 0 {
		variationIngredient := ingredientsWithMultipleRecipes[0]
		ingNode := g.Nodes[variationIngredient]

		numRecipes := len(ingNode.RecipesToMakeThisElement)
		recipesToExplore := numRecipes
		if recipesToExplore > maxVariations {
			recipesToExplore = maxVariations
		}

		for i := 0; i < recipesToExplore; i++ {
			*activeGoroutines++

			recipeIndex := i % numRecipes

			go func(ingredientRecipeIndex int) {
				visitCount := 0
				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   imagePath,
					"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
				}

				for _, ingredientName := range recipe.Ingredients {
					var ingredientTree map[string]interface{}
					visited := make(map[string]bool)
					ingredientVisitCount := 0

					if ingredientName == variationIngredient {
						ingredientTree = BuildIngredientTreeWithSpecificRecipe(
							g, ingredientName, ingNode.ImagePath,
							ingredientRecipeIndex, visited, &ingredientVisitCount)
					} else {
						ingredientTree = BuildElementTreeDFS(
							g, ingredientName, visited, &ingredientVisitCount)
					}

					tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingredientTree)
					visitCount += ingredientVisitCount
				}

				resultChan <- tree
				visitCountChan <- visitCount
			}(recipeIndex)
		}
	}
}

func BuildIngredientTreeWithSpecificRecipe(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	recipeIndex int,
	visited map[string]bool,
	visitedCount *int,
) map[string]interface{} {
	*visitedCount++
	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Check if it's a base element
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

	if len(node.RecipesToMakeThisElement) == 0 {
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	var recipe *graph.Recipe
	if recipeIndex >= 0 && recipeIndex < len(node.RecipesToMakeThisElement) {
		recipe = node.RecipesToMakeThisElement[recipeIndex]
	} else {
		recipe = node.RecipesToMakeThisElement[0]
	}

	ingredients := make([]interface{}, 0, len(recipe.Ingredients))
	for _, ingredientName := range recipe.Ingredients {
		ingredientTree := BuildElementTreeDFS(g, ingredientName, visited, visitedCount)
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        elementName,
		"imagePath":   imagePath,
		"ingredients": ingredients,
	}
}
