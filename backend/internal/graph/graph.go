package graph

import (
	"backend/model"
)

type Recipe struct {
	Result      string   `json:"result"`
	Ingredients []string `json:"ingredients"`
}

type ElementGraphNode struct {
	Name                       string    `json:"name"`
	ImagePath                  string    `json:"image_path"`
	RecipesToMakeThisElement   []*Recipe `json:"recipes_to_make_this_element"`
	RecipesMakingOtherElements []*Recipe `json:"recipes_making_other_elements"`
	IsVisited                  bool      `json:"is_visited"`
}

type ElementGraph struct {
	Nodes        map[string]*ElementGraphNode
	BaseElements []string
}

func NewElementGraph(elements map[string]model.Element) *ElementGraph {
	g := &ElementGraph{
		Nodes:        make(map[string]*ElementGraphNode),
		BaseElements: make([]string, 0),
	}

	for name, element := range elements {
		g.Nodes[name] = &ElementGraphNode{
			Name:                       name,
			ImagePath:                  element.ImagePath,
			RecipesToMakeThisElement:   make([]*Recipe, 0),
			RecipesMakingOtherElements: make([]*Recipe, 0),
		}
	}

	for name, element := range elements {
		for _, recipe := range element.Recipes {
			g.Nodes[name].RecipesToMakeThisElement = append(
				g.Nodes[name].RecipesToMakeThisElement,
				&Recipe{
					Result:      name,
					Ingredients: recipe.Ingredients,
				},
			)

			for _, ingredient := range recipe.Ingredients {
				if ingNode, exists := g.Nodes[ingredient]; exists {
					ingNode.RecipesMakingOtherElements = append(
						ingNode.RecipesMakingOtherElements,
						&Recipe{
							Result:      name,
							Ingredients: recipe.Ingredients,
						},
					)
				}
			}
		}
	}

	standardBaseElements := []string{"Water", "Fire", "Earth", "Air"}

	for _, baseName := range standardBaseElements {
		if _, exists := g.Nodes[baseName]; exists {
			g.BaseElements = append(g.BaseElements, baseName)
		}
	}

	if len(g.BaseElements) == 0 {
		for name, node := range g.Nodes {
			if len(node.RecipesToMakeThisElement) == 0 {
				g.BaseElements = append(g.BaseElements, name)
			}
		}
	}

	return g
}

func (g *ElementGraph) GetPossibleCombinations(elem1, elem2 string) []string {
	var results []string

	node1, exists1 := g.Nodes[elem1]
	if !exists1 {
		return results
	}

	for _, recipe := range node1.RecipesMakingOtherElements {
		if (recipe.Ingredients[0] == elem1 && recipe.Ingredients[1] == elem2) ||
			(recipe.Ingredients[0] == elem2 && recipe.Ingredients[1] == elem1) {
			results = append(results, recipe.Result)
		}
	}

	return results
}
