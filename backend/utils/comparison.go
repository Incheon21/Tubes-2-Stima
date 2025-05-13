package utils

import (
	"backend/internal/graph"
	"backend/model"
	"sort"
	"strings"
)

func CompareTreeIngredientsDeep(tree1, tree2 map[string]interface{}) bool {
	name1, _ := tree1["name"].(string)
	name2, _ := tree2["name"].(string)

	if name1 != name2 {
		return false
	}

	ingredients1, ok1 := tree1["ingredients"].([]interface{})
	ingredients2, ok2 := tree2["ingredients"].([]interface{})
	if !ok1 || !ok2 || len(ingredients1) != len(ingredients2) {
		return false
	}
	if len(ingredients1) == 0 {
		return true
	}
	ingMap1 := make(map[string]map[string]interface{})
	ingMap2 := make(map[string]map[string]interface{})
	for _, ing := range ingredients1 {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				ingMap1[name] = ingMap
			}
		}
	}
	for _, ing := range ingredients2 {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				ingMap2[name] = ingMap
			}
		}
	}
	if len(ingMap1) != len(ingMap2) {
		return false
	}
	for name, ing1 := range ingMap1 {
		ing2, exists := ingMap2[name]
		if !exists {
			return false
		}
		if !CompareTreeIngredientsDeep(ing1, ing2) {
			return false
		}
	}
	return true
}

func DeepCopyTree(tree map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range tree {
		if key == "ingredients" {
			if ingredients, ok := value.([]interface{}); ok {
				copiedIngredients := make([]interface{}, 0, len(ingredients))
				for _, ing := range ingredients {
					if ingMap, ok := ing.(map[string]interface{}); ok {
						copiedIngredients = append(copiedIngredients, DeepCopyTree(ingMap))
					}
				}
				result[key] = copiedIngredients
			} else {
				result[key] = []interface{}{}
			}
		} else {
			result[key] = value
		}
	}

	return result
}

func CompareTreeIngredients(tree1, tree2 map[string]interface{}) bool {
	name1, _ := tree1["name"].(string)
	name2, _ := tree2["name"].(string)

	if name1 != name2 {
		return false
	}

	ingredients1, _ := tree1["ingredients"].([]interface{})
	ingredients2, _ := tree2["ingredients"].([]interface{})

	if len(ingredients1) != len(ingredients2) {
		return false
	}

	ingNames1 := make([]string, 0)
	ingNames2 := make([]string, 0)

	for _, ing := range ingredients1 {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				ingNames1 = append(ingNames1, name)
			}
		}
	}

	for _, ing := range ingredients2 {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				ingNames2 = append(ingNames2, name)
			}
		}
	}

	sort.Strings(ingNames1)
	sort.Strings(ingNames2)

	if len(ingNames1) != len(ingNames2) {
		return false
	}

	for i := range ingNames1 {
		if ingNames1[i] != ingNames2[i] {
			return false
		}
	}

	return true
}

func GeneratePathFingerprint(path []model.Node) string {
	elements := make([]string, 0, len(path))
	for _, node := range path {
		elements = append(elements, node.Element)
	}
	sort.Strings(elements)
	return strings.Join(elements, ",")
}

func GenerateTreeSignature(tree map[string]interface{}) string {
	rootName := tree["name"].(string)

	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return rootName + "|no_ingredients"
	}

	names := make([]string, 0, len(ingredients))
	for _, ing := range ingredients {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				names = append(names, name)
			}
		}
	}

	sort.Strings(names)

	return rootName + "|" + strings.Join(names, ",")
}

func VerifyTreeIngredientsComplete(tree map[string]interface{}, availableRecipes []*graph.Recipe) bool {
	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok {
		return false
	}

	treeIngredientNames := make([]string, 0)
	for _, ing := range ingredients {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				treeIngredientNames = append(treeIngredientNames, name)
			}
		}
	}

	for _, recipe := range availableRecipes {
		if len(recipe.Ingredients) != len(treeIngredientNames) {
			continue
		}

		recipeMatches := true
		for _, recipeIng := range recipe.Ingredients {
			found := false
			for _, treeIng := range treeIngredientNames {
				if recipeIng == treeIng {
					found = true
					break
				}
			}

			if !found {
				recipeMatches = false
				break
			}
		}

		if recipeMatches {
			return true
		}
	}

	return false
}

func VerifyCompletePath(path []model.Node, baseElements []string, target string) bool {
	if len(path) == 0 {
		return false
	}

	if path[len(path)-1].Element != target {
		return false
	}

	// Ensure no duplicates in path
	seen := make(map[string]bool)
	for _, node := range path {
		if seen[node.Element] {
			continue
		}
		seen[node.Element] = true
	}

	firstElem := path[0].Element
	isBaseElement := false
	for _, base := range baseElements {
		if firstElem == base {
			isBaseElement = true
			break
		}
	}

	return isBaseElement
}
