package algorithm

import (
    "backend/internal/graph"
    "backend/model"
    "log"
)

// ReverseDFS finds recipe paths from target back to base elements
func DFS(elements map[string]model.Element, target string, maxResults int, debug bool) ([][]model.Node, int) {
    if debug {
        log.Printf("DEBUG: Starting ReverseDFS for target: %s (max results: %d)", target, maxResults)
    }

    // Build the graph once
    g := graph.NewElementGraph(elements)
    
    // Check if target exists in the graph
    targetNode, exists := g.Nodes[target]
    if !exists {
        if debug {
            log.Printf("DEBUG: Target element %s not found in database", target)
        }
        return [][]model.Node{}, 0
    }
    
    // Handle case where target is a base element
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    for _, base := range baseElements {
        if target == base {
            if debug {
                log.Printf("DEBUG: Target %s is a base element, returning simple result", target)
            }
            return [][]model.Node{{
                {Element: target, ImagePath: targetNode.ImagePath},
            }}, 0
        }
    }
    
    // Initialize tracking data structures
    visited := make(map[string]bool)
    visitedCount := 0
    var results [][]model.Node
    
    // Add all recipes for the target element to the queue
    if debug {
        log.Printf("DEBUG: Target %s has %d recipes to explore", target, len(targetNode.RecipesToMakeThisElement))
    }
    
    for _, recipe := range targetNode.RecipesToMakeThisElement {
        // Create a new path starting with the target
        path := []*model.Node{
            {Element: target, ImagePath: targetNode.ImagePath},
        }
        
        // Process this recipe
        Explore(g, recipe, path, visited, &visitedCount, &results, maxResults, baseElements, debug)
        
        // Stop if we've found enough paths
        if len(results) >= maxResults && maxResults > 0 {
            if debug {
                log.Printf("DEBUG: Found %d paths, stopping exploration", len(results))
            }
            break
        }
    }
    
    if debug {
        log.Printf("DEBUG: ReverseDFS complete - found %d paths after visiting %d nodes", len(results), visitedCount)
    }
    
    return results, visitedCount
}

// Helper function for reverse DFS exploration
// Helper function for reverse DFS exploration
func Explore(
    g *graph.ElementGraph,
    recipe *graph.Recipe,
    currentPath []*model.Node,
    visited map[string]bool,
    visitedCount *int,
    results *[][]model.Node,
    maxResults int,
    baseElements []string,
    debug bool,
) {
    // Stop if we've found enough paths
    if len(*results) >= maxResults && maxResults > 0 {
        return
    }
    
    if debug {
        log.Printf("DEBUG: Exploring recipe: %s from ingredients: %v", recipe.Result, recipe.Ingredients)
    }
    
    // Get ingredients for this recipe
    ingredients := recipe.Ingredients
    if len(ingredients) == 0 {
        if debug {
            log.Printf("DEBUG: Skipping recipe with no ingredients")
        }
        return
    }
    
    // Create a new path including these ingredients
    newPath := make([]*model.Node, len(currentPath))
    copy(newPath, currentPath)
    
    // Track if we need to explore deeper or if we have a complete path
    allIngredientsAreBaseElements := true
    ingredientNodes := make([]*model.Node, 0, len(ingredients))
    
    // Add all ingredients to the path and check if any need further exploration
    for _, ingredient := range ingredients {
        ingredientNode := g.Nodes[ingredient]
        *visitedCount++
        
        ingredientNodeObj := &model.Node{
            Element: ingredient,
            ImagePath: ingredientNode.ImagePath,
        }
        ingredientNodes = append(ingredientNodes, ingredientNodeObj)
        
        // Check if this is a base element
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
    
    // Add all ingredient nodes to the path
    newPath = append(newPath, ingredientNodes...)
    
    // If all ingredients are base elements, we've found a complete path
    if allIngredientsAreBaseElements {
        // Convert path of pointers to path of values
        finalPath := make([]model.Node, len(newPath))
        for i, node := range newPath {
            finalPath[i] = *node
        }
        
        // Reverse the path so it starts with base elements
        for i, j := 0, len(finalPath)-1; i < j; i, j = i+1, j-1 {
            finalPath[i], finalPath[j] = finalPath[j], finalPath[i]
        }
        
        *results = append(*results, finalPath)
        
        if debug {
            log.Printf("DEBUG: Found complete path with %d steps", len(finalPath))
        }
        return
    }
    
    // Otherwise, recursively explore each non-base ingredient
    for _, ingredient := range ingredients {
        isBase := false
        for _, base := range baseElements {
            if ingredient == base {
                isBase = true
                break
            }
        }
        
        if isBase {
            if debug {
                log.Printf("DEBUG: Ingredient %s is a base element, skipping further exploration", ingredient)
            }
            continue
        }
        
        // Skip if this ingredient has been visited in this path (avoid cycles)
        if visited[ingredient] {
            continue
        }
        
        visited[ingredient] = true
        
        ingredientNode := g.Nodes[ingredient]
        if debug {
            log.Printf("DEBUG: Exploring ingredient %s which has %d recipes", ingredient, len(ingredientNode.RecipesToMakeThisElement))
        }
        
        // For each recipe to make this ingredient, recursively explore
        for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
            // Create a path for this ingredient (start with the current path)
            ingredientPath := make([]*model.Node, len(newPath))
            copy(ingredientPath, newPath)
            
            // Recursively explore
            Explore(g, subRecipe, ingredientPath, visited, visitedCount, results, maxResults, baseElements, debug)
            
            // Stop if we've found enough paths
            if len(*results) >= maxResults && maxResults > 0 {
                break
            }
        }
        
        // Backtrack
        delete(visited, ingredient)
    }
}