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
    graph := &ElementGraph{
        Nodes:        make(map[string]*ElementGraphNode),
        BaseElements: []string{},
    }

    // First pass: Create all nodes
    for name, element := range elements {
        graph.Nodes[name] = &ElementGraphNode{
            Name:                      name,
            ImagePath:                 element.ImagePath,
            RecipesToMakeThisElement:  []*Recipe{},
            RecipesToMakeOtherElement: []*Recipe{},
            IsVisited:                 false,
        }

        // Mark base elements as visited and add to base elements list
        if element.Tier == 1 && (name == "Water" || name == "Fire" || name == "Earth" || name == "Air") {
            graph.BaseElements = append(graph.BaseElements, name)
            graph.Nodes[name].IsVisited = true
        }
    }

    // Second pass: Fill in recipes
    for resultName, element := range elements {
        for _, recipe := range element.Recipes {
            if len(recipe.Ingredients) == 2 {
                // Create recipe object
                recipeObj := &Recipe{
                    Result:      resultName,
                    Ingredients: recipe.Ingredients,
                }

                // Add recipe to the result element's recipes
                graph.Nodes[resultName].RecipesToMakeThisElement = append(
                    graph.Nodes[resultName].RecipesToMakeThisElement, recipeObj)

                // Add recipe to each ingredient's recipes that can make other elements
                for _, ingredientName := range recipe.Ingredients {
                    if ingredientNode, exists := graph.Nodes[ingredientName]; exists {
                        ingredientNode.RecipesToMakeOtherElement = append(
                            ingredientNode.RecipesToMakeOtherElement, recipeObj)
                    }
                }
            }
        }
    }

    return graph
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
