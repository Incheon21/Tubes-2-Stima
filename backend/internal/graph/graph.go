package graph

import (
	"backend/model"
)

// Recipe represents a combination of elements that produces a result
type Recipe struct {
	Result      string   `json:"result"`
	Ingredients []string `json:"ingredients"`
}

// ElementGraphNode stores an element and its connections in the graph
type ElementGraphNode struct {
	Name                      string    `json:"name"`
	ImagePath                 string    `json:"image_path"`
    Tier                      int       `json:"tier"`  // Added tier field
	RecipesToMakeThisElement  []*Recipe `json:"recipes_to_make_this_element"`
	RecipesToMakeOtherElement []*Recipe `json:"recipes_to_make_other_element"`
	IsVisited                 bool      `json:"is_visited"`
}

// ElementGraph is the graph of all elements and their relationships
type ElementGraph struct {
	Nodes        map[string]*ElementGraphNode // map of element name to node
	BaseElements []string                     // list of base elements
}

func NewElementGraph(elements map[string]model.Element) *ElementGraph {
	// Create a new graph
	g := &ElementGraph{
		Nodes:        make(map[string]*ElementGraphNode),
		BaseElements: make([]string, 0),
	}

	// First, create nodes for all elements
	for name, element := range elements {
		g.Nodes[name] = &ElementGraphNode{
			Name:                      name,
			ImagePath:                 element.ImagePath,
            Tier:                      element.Tier,
			RecipesToMakeThisElement:  make([]*Recipe, 0),
			RecipesToMakeOtherElement: make([]*Recipe, 0),
		}
	}

	// Then, add recipe connections
	for name, element := range elements {
		for _, recipe := range element.Recipes {
			// Add this recipe as a way to make this element
			g.Nodes[name].RecipesToMakeThisElement = append(
				g.Nodes[name].RecipesToMakeThisElement,
				&Recipe{
					Result:      name,
					Ingredients: recipe.Ingredients,
				},
			)

			// Also add this recipe to each ingredient as a way to make other elements
			for _, ingredient := range recipe.Ingredients {
				if ingNode, exists := g.Nodes[ingredient]; exists {
					ingNode.RecipesToMakeOtherElement = append(
						ingNode.RecipesToMakeOtherElement,
						&Recipe{
							Result:      name,
							Ingredients: recipe.Ingredients,
						},
					)
				}
			}
		}
	}

	// FIXED: Explicitly mark the four base elements regardless of recipes
	standardBaseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Add standard base elements if they exist
	for _, baseName := range standardBaseElements {
		if _, exists := g.Nodes[baseName]; exists {
			g.BaseElements = append(g.BaseElements, baseName)
		}
	}

	// If no standard base elements found, then use the original method
	if len(g.BaseElements) == 0 {
		// Find elements without recipes as fallback base elements
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

	// Check if elem1 exists in the graph
	node1, exists1 := g.Nodes[elem1]
	if !exists1 {
		return results
	}

	// Look through all recipes where elem1 is an ingredient
	for _, recipe := range node1.RecipesToMakeOtherElement {
		// Check if elem2 is the other ingredient in this recipe
		if (recipe.Ingredients[0] == elem1 && recipe.Ingredients[1] == elem2) ||
			(recipe.Ingredients[0] == elem2 && recipe.Ingredients[1] == elem1) {
			results = append(results, recipe.Result)
		}
	}

	return results
}
