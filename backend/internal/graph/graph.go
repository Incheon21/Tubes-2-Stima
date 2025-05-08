package graph

import (
	"backend/model"
)

type ElementGraph struct {
	Nodes        map[string]*Node // map of element name to node
	BaseElements []string         // list of base elements
}

type Node struct {
	Element      model.Element
	Combinations map[string][]string
}

func NewElementGraph(elements map[string]model.Element) *ElementGraph {
	graph := &ElementGraph{
		Nodes:        make(map[string]*Node),
		BaseElements: []string{},
	}

	for name, element := range elements {
		graph.Nodes[name] = &Node{
			Element:      element,
			Combinations: make(map[string][]string),
		}

		if element.Tier == 1 && (name == "Water" || name == "Fire" || name == "Earth" || name == "Air") {
			graph.BaseElements = append(graph.BaseElements, name)
		}
	}

	for name, element := range elements {
		for _, recipe := range element.Recipes {
			if len(recipe.Ingredients) == 2 {
				ingredient1 := recipe.Ingredients[0]
				ingredient2 := recipe.Ingredients[1]

				if _, exists := graph.Nodes[ingredient1]; exists {
					graph.Nodes[ingredient1].Combinations[name] = []string{ingredient1, ingredient2}
				}

				if _, exists := graph.Nodes[ingredient2]; exists {
					graph.Nodes[ingredient2].Combinations[name] = []string{ingredient1, ingredient2}
				}
			}
		}
	}

	return graph
}

func (g *ElementGraph) GetPossibleCombinations(elem1, elem2 string) []string {
	var results []string

	node1, exists1 := g.Nodes[elem1]
	if !exists1 {
		return results
	}

	for result, ingredients := range node1.Combinations {
		if (ingredients[0] == elem1 && ingredients[1] == elem2) ||
			(ingredients[0] == elem2 && ingredients[1] == elem1) {
			results = append(results, result)
		}
	}

	return results
}
