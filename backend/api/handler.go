package api

import (
	alg "backend/internal/algorithm"
	"backend/internal/graph"
	"backend/model"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv" // Add this
	"strings"
	"time"
)

// Helper function to generate recipe variations considering all possible subtrees
func GenerateAllRecipeVariations(
    g *graph.ElementGraph, 
    elementName string,
    imagePath string,
    maxCount int) ([]map[string]interface{}, int) {
    
    totalVisitedCount := 0
    node := g.Nodes[elementName]
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    
    // Check if it's a base element
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
    
    // If no recipes, return empty ingredient list
    if node == nil || len(node.RecipesToMakeThisElement) == 0 {
        return []map[string]interface{}{{
            "name":        elementName,
            "imagePath":   imagePath,
            "ingredients": []interface{}{},
            "noRecipe":    true,
        }}, 1
    }
    
    // For each recipe of the target element, generate all possible variations
    allTrees := make([]map[string]interface{}, 0)
    uniqueSignatures := make(map[string]bool)
    
    // Process each recipe
    for _, recipe := range node.RecipesToMakeThisElement {
        if len(recipe.Ingredients) == 0 {
            continue // Skip empty recipes
        }
        
        // Generate all variations for this recipe's ingredients
        generateRecipeVariationsWithSubIngredients(
            g, elementName, imagePath, recipe, &allTrees, &uniqueSignatures, 
            &totalVisitedCount, maxCount, 0, 3) // Limit depth to 3 levels to prevent explosion
    }
    
    // If we ended up with no trees, create a simple one
    if len(allTrees) == 0 {
        visitCount := 0
        visited := make(map[string]bool)
        tree := buildElementTreeDFSNoCircular(g, elementName, visited, &visitCount)
        allTrees = append(allTrees, tree)
        totalVisitedCount += visitCount
    }
    
    // Limit to requested count
    if len(allTrees) > maxCount {
        allTrees = allTrees[:maxCount]
    }
    
    return allTrees, totalVisitedCount
}

// Helper function to recursively generate all variations of recipes with all possible sub-ingredient combinations
func generateRecipeVariationsWithSubIngredients(
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
    
    // Stop if we've reached depth limit or found enough trees
    if currentDepth > maxDepth || (len(*allTrees) >= maxCount && maxCount > 0) {
        return
    }
    
    // For each ingredient in this recipe, get all possible ways to make it
    ingredientVariations := make([][]map[string]interface{}, len(recipe.Ingredients))
    
    // First, gather all variations for each ingredient
    for i, ingredient := range recipe.Ingredients {
        ingredientNode := g.Nodes[ingredient]
        *totalVisitedCount++
        
        if ingredientNode == nil {
            // Skip invalid ingredients
            continue
        }
        
        // Generate all trees for this ingredient
        var ingredientTrees []map[string]interface{}
        
        // For base elements, just create one variation
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
        
        // If not a base element, get all possible ways to make it
        if !isBase {
            if len(ingredientNode.RecipesToMakeThisElement) == 0 {
                // No recipes for this ingredient
                ingredientTrees = []map[string]interface{}{{
                    "name":        ingredient,
                    "imagePath":   ingredientNode.ImagePath,
                    "ingredients": []interface{}{},
                    "noRecipe":    true,
                }}
            } else if currentDepth >= maxDepth-1 {
                // At max depth, just use one standard recipe
                visited := make(map[string]bool)
                visitCount := 0
                tree := buildElementTreeDFSNoCircular(g, ingredient, visited, &visitCount)
                *totalVisitedCount += visitCount
                ingredientTrees = []map[string]interface{}{tree}
            } else {
                // Generate all variations for each recipe of this ingredient
                for _, subRecipe := range ingredientNode.RecipesToMakeThisElement {
                    // Recursively generate variations for this sub-recipe
                    subVariations := make([]map[string]interface{}, 0)
                    tempSigs := make(map[string]bool)
                    
                    // Use a temporary slice to avoid modifying allTrees directly
                    generateRecipeVariationsWithSubIngredients(
                        g, ingredient, ingredientNode.ImagePath, subRecipe, 
                        &subVariations, &tempSigs, totalVisitedCount, 
                        2, currentDepth+1, maxDepth)
                    
                    // Add all generated variations
                    for _, variation := range subVariations {
                        ingredientTrees = append(ingredientTrees, variation)
                    }
                    
                    // If we don't have any variations, create a simple tree
                    if len(subVariations) == 0 {
                        visited := make(map[string]bool)
                        visitCount := 0
                        tree := buildElementTreeDFSNoCircular(g, ingredient, visited, &visitCount)
                        *totalVisitedCount += visitCount
                        ingredientTrees = append(ingredientTrees, tree)
                    }
                }
            }
        }
        
        // If we couldn't generate any trees for this ingredient, create a simple one
        if len(ingredientTrees) == 0 {
            ingredientTrees = []map[string]interface{}{{
                "name":        ingredient,
                "imagePath":   ingredientNode.ImagePath,
                "ingredients": []interface{}{},
            }}
        }
        
        // Store all variations for this ingredient
        ingredientVariations[i] = ingredientTrees
    }
    
    // Now, generate all combinations of ingredient variations
    generateTreeCombinations(
        g, elementName, imagePath, recipe.Ingredients, 
        ingredientVariations, 0, []map[string]interface{}{}, 
        allTrees, uniqueSignatures, totalVisitedCount, maxCount)
}

// Helper function to generate all possible combinations of ingredient trees
func generateTreeCombinations(
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
    
    // Stop if we've found enough trees
    if len(*allTrees) >= maxCount && maxCount > 0 {
        return
    }
    
    // If we've processed all ingredients, create a tree from this combination
    if currentIndex >= len(ingredientVariations) {
        // Create a tree for this combination
        tree := map[string]interface{}{
            "name":        elementName,
            "imagePath":   imagePath,
            "ingredients": make([]interface{}, len(currentCombination)),
        }
        
        // Add all ingredients
        for i, ingTree := range currentCombination {
            tree["ingredients"].([]interface{})[i] = ingTree
        }
        
        // Check if this tree is unique
        signature := generateDetailedTreeSignature(tree)
        if !(*uniqueSignatures)[signature] {
            (*uniqueSignatures)[signature] = true
            *allTrees = append(*allTrees, tree)
            
            // Log that we found a new unique tree
            log.Printf("DEBUG: Generated unique recipe tree variation with signature: %s", signature)
        }
        
        return
    }
    
    // If there are no variations for this ingredient, skip it
    if len(ingredientVariations[currentIndex]) == 0 {
        generateTreeCombinations(
            g, elementName, imagePath, ingredientNames,
            ingredientVariations, currentIndex+1, currentCombination,
            allTrees, uniqueSignatures, totalVisitedCount, maxCount)
        return
    }
    
    // Try each variation of the current ingredient
    for _, variation := range ingredientVariations[currentIndex] {
        // Add this variation to the current combination
        newCombination := append(currentCombination, variation)
        
        // Recursive call to process next ingredient
        generateTreeCombinations(
            g, elementName, imagePath, ingredientNames,
            ingredientVariations, currentIndex+1, newCombination,
            allTrees, uniqueSignatures, totalVisitedCount, maxCount)
        
        // Stop if we've found enough trees
        if len(*allTrees) >= maxCount && maxCount > 0 {
            return
        }
    }
}

// Generate a detailed signature for a tree that includes recursive ingredient paths
func generateDetailedTreeSignature(tree map[string]interface{}) string {
    name := tree["name"].(string)
    ingredients, ok := tree["ingredients"].([]interface{})
    
    if !ok || len(ingredients) == 0 {
        return name + "|[]"
    }
    
    // Recursively generate signatures for all ingredients
    subSigs := make([]string, 0, len(ingredients))
    
    for _, ing := range ingredients {
        if ingMap, ok := ing.(map[string]interface{}); ok {
            subSig := generateDetailedTreeSignature(ingMap)
            subSigs = append(subSigs, subSig)
        }
    }
    
    // Sort for consistent signatures regardless of order
    sort.Strings(subSigs)
    
    return name + "|[" + strings.Join(subSigs, ";") + "]"
}

// buildElementTreeDFSNoCircular is a variation of DFS tree building without circular reference checks
func buildElementTreeDFSNoCircular(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
    // No need to check for circular references since tier rules prevent it
    
    *visitedCount++
    node := g.Nodes[elementName]
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    
    // Check if it's a base element
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
    
    // Get the recipes to make this element
    if len(node.RecipesToMakeThisElement) == 0 {
        // No recipe found
        return map[string]interface{}{
            "name":        elementName,
            "imagePath":   node.ImagePath,
            "ingredients": []interface{}{},
            "noRecipe":    true,
        }
    }
    
    // Find the recipe with the shortest combined ingredient path length
    var bestRecipe *graph.Recipe
    var bestPathLength = 9999 // Start with a high value
    
    // Try all recipes
    for _, recipe := range node.RecipesToMakeThisElement {
        // Calculate approximate path length without fully exploring
        totalPathLength := 0
        for _, ingredient := range recipe.Ingredients {
            // Base elements have path length 1
            if isBaseElementName(ingredient, baseElements) {
                totalPathLength += 1
            } else if ingNode, exists := g.Nodes[ingredient]; exists {
                // Add 1 for each level of recipes needed
                if len(ingNode.RecipesToMakeThisElement) > 0 {
                    totalPathLength += 2
                } else {
                    totalPathLength += 1
                }
            }
        }
        
        // Choose this recipe if it's the shortest so far
        if totalPathLength < bestPathLength {
            bestPathLength = totalPathLength
            bestRecipe = recipe
        }
    }
    
    // If no valid recipe was found, use the first one
    if bestRecipe == nil && len(node.RecipesToMakeThisElement) > 0 {
        bestRecipe = node.RecipesToMakeThisElement[0]
    }
    
    // Build the ingredients tree
    ingredients := make([]interface{}, 0, len(bestRecipe.Ingredients))
    for _, ingredientName := range bestRecipe.Ingredients {
        ingredientTree := buildElementTreeDFSNoCircular(g, ingredientName, visited, visitedCount)
        ingredients = append(ingredients, ingredientTree)
    }
    
    return map[string]interface{}{
        "name":        elementName,
        "imagePath":   node.ImagePath,
        "ingredients": ingredients,
    }
}
type Handler struct {
	elements map[string]model.Element
}

func NewHandler(elements map[string]model.Element) *Handler {
	return &Handler{elements: elements}
}

// pathToTree converts a linear path to a tree structure
func pathToTree(path []model.Node, elements map[string]model.Element, algorithm string) map[string]interface{} {
	if len(path) == 0 {
		return nil
	}

	// The path from DFS is in reverse order (target element first, base elements last)
	// For a tree, we want to start with the target element
	targetElement := path[0].Element
	targetImagePath := path[0].ImagePath

	// Base case: if only one element, return it as a leaf node
	if len(path) == 1 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
		}
	}

	// Base elements check
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

	// For non-base elements, we need to determine the recipe
	// Build a graph to access recipe information
	g := createElementGraph(elements)
	node := g.Nodes[targetElement]

	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	// Find the ingredients for this element from the path
	// In DFS paths, the elements after the target are its ingredients
	ingredients := []interface{}{}

	// Look for matching recipes in the graph
	for _, recipe := range node.RecipesToMakeThisElement {
		// Try to match this recipe with the path
		if len(recipe.Ingredients) > 0 {
			ingredientMatches := 0
			ingredientTrees := []interface{}{}

			// Check if the ingredients in this recipe match elements in our path
			for _, ingredient := range recipe.Ingredients {
				// Find this ingredient in the path
				for i := 1; i < len(path); i++ {
					if path[i].Element == ingredient {
						// Found a matching ingredient, create a subtree for it
						subtree := createSubtreeFromPath(path[i:], elements, algorithm)
						ingredientTrees = append(ingredientTrees, subtree)
						ingredientMatches++
						break
					}
				}
			}

			// If we matched all ingredients in this recipe, use it
			if ingredientMatches == len(recipe.Ingredients) {
				ingredients = ingredientTrees
				break
			}
		}
	}

	// If we couldn't match ingredients from the path, use the standard tree building approach
	// If we couldn't match ingredients from the path, use the standard tree building approach
	if len(ingredients) == 0 {
		visited := make(map[string]bool)
		visitedCount := 0
		var tree map[string]interface{}

		// Use appropriate algorithm to build the tree
		if algorithm == "bfs" {
			tree = buildElementTreeBFS(g, targetElement, visited, &visitedCount)
			log.Printf("DEBUG: Using BFS to build fallback tree for %s", targetElement)
		} else if algorithm == "dfs" {
			tree = buildElementTreeDFS(g, targetElement, visited, &visitedCount)
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

// createSubtreeFromPath creates a subtree for an ingredient starting from its position in the path
func createSubtreeFromPath(subPath []model.Node, elements map[string]model.Element, algorithm string) map[string]interface{} {
	if len(subPath) == 0 {
		return nil
	}

	elementName := subPath[0].Element
	imagePath := subPath[0].ImagePath

	// Check if it's a base element
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

	// If this is the only element left in the path, it's a leaf node
	if len(subPath) == 1 {
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
		}
	}

	// Otherwise, recursively build a tree using the full path-to-tree conversion
	return pathToTree(subPath, elements, algorithm)
}

// HandleBestRecipesTree returns the best recipe for an element in tree format
func (h *Handler) HandleBestRecipesTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleBestRecipesTree request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/best-recipes-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/best-recipes-tree/{elementName}?count=N&algorithm=algo", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	// Get number of recipes to return
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)

	// Get algorithm to use (default to DFS if not specified)
	algorithm := "bfs"
	if algoParam := r.URL.Query().Get("algorithm"); algoParam != "" {
		algorithm = strings.ToLower(algoParam)
	}
	log.Printf("DEBUG: Using algorithm: %s", algorithm)
	// Limit maximum count to prevent performance issues
	if count > 5 {
		count = 5
		log.Printf("DEBUG: Limiting count to maximum of 5 for tree format")
	}

	// Check if element exists
	element, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// For base elements, return simple result
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			result := map[string]interface{}{
				"trees": []map[string]interface{}{{
					"name":          elementName,
					"imagePath":     element.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}},
				"nodesVisited": 1,
				"timeElapsed":  0,
				"algorithm":    algorithm,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	startTime := time.Now()

	// Build complete recipe trees using specified algorithm
	g := createElementGraph(h.elements)
	recipeTrees := make([]map[string]interface{}, 0, count)
	visitedNodesCount := 0

	// Try finding recipe trees for different recipes of the element
	node := g.Nodes[elementName]

	// If there's no recipe, return empty tree
	if len(node.RecipesToMakeThisElement) == 0 {
		tree := map[string]interface{}{
			"name":        elementName,
			"imagePath":   element.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
		recipeTrees = append(recipeTrees, tree)
	} else {
		// Try each recipe to make this element, until we have 'count' trees
		for _, recipe := range node.RecipesToMakeThisElement {
			if len(recipeTrees) >= count {
				break
			}
			localVisitCount := 0

			// Build tree starting with this recipe
			tree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   element.ImagePath,
				"ingredients": []interface{}{},
			}

			// Add all ingredients as subtrees
			ingredients := make([]interface{}, 0, len(recipe.Ingredients))
			for _, ingredientName := range recipe.Ingredients {
				ingredientVisited := make(map[string]bool)
				ingredientVisitCount := 0

				var ingredientTree map[string]interface{}
				if algorithm == "bfs" {
					ingredientTree = buildElementTreeBFS(g, ingredientName, ingredientVisited, &ingredientVisitCount)
				} else {
					ingredientTree = buildElementTreeDFS(g, ingredientName, ingredientVisited, &ingredientVisitCount)
				}

				ingredients = append(ingredients, ingredientTree)
				localVisitCount += ingredientVisitCount
			}

			tree["ingredients"] = ingredients
			visitedNodesCount += localVisitCount

			// Check if this tree is unique compared to existing trees
			isUnique := true
			for _, existingTree := range recipeTrees {
				if compareTreeIngredients(existingTree, tree) {
					isUnique = false
					break
				}
			}

			if isUnique {
				recipeTrees = append(recipeTrees, tree)
				log.Printf("")
				log.Printf("DEBUG: Added recipe tree using %s algorithm with recipe containing %d ingredients", algorithm, len(recipe.Ingredients))
			}
		}

		// If we still don't have enough trees, try the selected algorithm for alternative paths
		if len(recipeTrees) < count {
			maxResults := count * 2
			if maxResults > 10 {
				maxResults = 10
			}

			var paths [][]model.Node
			var visited int

			switch algorithm {
			case "bfs":
				// Dari:
				paths, visited = alg.BFS(h.elements, elementName, maxResults, false)
			case "dfs":
				// Default to DFS
				paths, visited = alg.DFS(h.elements, elementName, maxResults, false)
			}

			log.Printf("DEBUG: %s found %d paths after visiting %d nodes",
				strings.ToUpper(algorithm), len(paths), visited)
			visitedNodesCount += visited

			// Sort paths by length (shorter paths first)
			sort.Slice(paths, func(i, j int) bool {
				return len(paths[i]) < len(paths[j])
			})

			// Convert remaining needed paths to tree format
			for i, path := range paths {
				if len(recipeTrees) >= count {
					break
				}

				// Skip too short paths
				if len(path) < 2 {
					continue
				}

				// Convert path to a proper tree structure
				g := createElementGraph(h.elements)

				// Build tree starting with this recipe
				tree := map[string]interface{}{
					"name":        elementName,
					"imagePath":   element.ImagePath,
					"ingredients": []interface{}{},
				}

				// Extract unique ingredients from the path
				ingredientSet := make(map[string]bool)
				for i := 1; i < len(path); i++ {
					if path[i].Element != elementName {
						ingredientSet[path[i].Element] = true
					}
				}

				// For each ingredient, build a complete tree
				ingredients := make([]interface{}, 0)
				for ingredient := range ingredientSet {
					// Check if the element has this as a direct ingredient in any recipe
					isDirectIngredient := false
					for _, recipe := range node.RecipesToMakeThisElement {
						for _, ing := range recipe.Ingredients {
							if ing == ingredient {
								isDirectIngredient = true
								break
							}
						}
						if isDirectIngredient {
							break
						}
					}

					if isDirectIngredient {
						ingredientVisited := make(map[string]bool)
						ingredientVisitCount := 0

						var ingredientTree map[string]interface{}
						if algorithm == "bfs" {
							ingredientTree = buildElementTreeBFS(g, ingredient, ingredientVisited, &ingredientVisitCount)
						} else {
							ingredientTree = buildElementTreeDFS(g, ingredient, ingredientVisited, &ingredientVisitCount)
						}

						ingredients = append(ingredients, ingredientTree)
						visitedNodesCount += ingredientVisitCount
					}
				}

				// Only use this path if we found ingredients
				if len(ingredients) > 0 {
					tree["ingredients"] = ingredients

					// Check if this tree is unique compared to existing trees
					isUnique := true
					for _, existingTree := range recipeTrees {
						if compareTreeIngredients(existingTree, tree) {
							isUnique = false
							break
						}
					}

					if isUnique {
						recipeTrees = append(recipeTrees, tree)
						log.Printf("DEBUG: Added alternative recipe tree from path %d", i+1)
					}
				}
			}
		}
	}

	// If we still don't have any trees, create one with the standard tree builder
	if len(recipeTrees) == 0 {
		visited := make(map[string]bool)
		visitCount := 0
		var mainTree map[string]interface{}

		if algorithm == "bfs" {
			mainTree = buildElementTreeBFS(g, elementName, visited, &visitCount)
		} else {
			mainTree = buildElementTreeDFS(g, elementName, visited, &visitCount)
		}

		recipeTrees = append(recipeTrees, mainTree)
		visitedNodesCount += visitCount
		log.Printf("DEBUG: Added fallback element tree using %s", strings.ToUpper(algorithm))
	}

	result := map[string]interface{}{
		"trees":        recipeTrees,
		"nodesVisited": visitedNodesCount,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
		"algorithm":    algorithm,
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response with %d recipe trees", len(recipeTrees))
}

// MultiThreadedElementTreeDFS generates multiple recipe trees using multiple goroutines
// It considers variations in child element recipes as well
func MultiThreadedElementTreeDFS(g *graph.ElementGraph, elementName string, count int) ([]map[string]interface{}, int) {
    totalVisitedCount := 0
    resultTrees := make([]map[string]interface{}, 0, count)
    uniqueSignatures := make(map[string]bool)
    
    // Create a channel to collect results from goroutines
    resultChan := make(chan map[string]interface{}, count*3) // Increased buffer size
    visitCountChan := make(chan int, count*3)
    
    // Generate trees for each recipe of the target element
    node := g.Nodes[elementName]
    if node == nil || len(node.RecipesToMakeThisElement) == 0 {
        // Handle case with no recipes
        visitCount := 0
        visited := make(map[string]bool)
        tree := buildElementTreeDFSNoCircular(g, elementName, visited, &visitCount)
        return []map[string]interface{}{tree}, visitCount
    }
    
    // Start a separate goroutine for each recipe of the target element
    activeGoroutines := 0
    for _, recipe := range node.RecipesToMakeThisElement {
        // Skip recipes with no ingredients
        if len(recipe.Ingredients) == 0 {
            continue
        }
        
        // For each recipe of the target element, generate multiple variations
        // based on different ways to make each ingredient
        generateRecipeVariations(g, elementName, node.ImagePath, recipe, &activeGoroutines, 
                                resultChan, visitCountChan, 0, count)
    }
    
    log.Printf("DEBUG: Started %d goroutines to explore recipe variations", activeGoroutines)
    
    // If no goroutines were started, return a default tree
    if activeGoroutines == 0 {
        visitCount := 0
        visited := make(map[string]bool)
        tree := buildElementTreeDFSNoCircular(g, elementName, visited, &visitCount)
        return []map[string]interface{}{tree}, visitCount
    }
    
    // Collect results from goroutines
    for i := 0; i < activeGoroutines; i++ {
        tree := <-resultChan
        visitCount := <-visitCountChan
        
        // Check if this tree is unique
        signature := generateTreeSignature(tree)
        if !uniqueSignatures[signature] {
            uniqueSignatures[signature] = true
            resultTrees = append(resultTrees, tree)
            totalVisitedCount += visitCount
            
            // If we have enough unique trees, we can stop collecting
            // but we still need to drain the channels
            if len(resultTrees) >= count {
                log.Printf("DEBUG: Reached target count of %d unique trees, will stop adding more", count)
                // Continue receiving from channels to avoid goroutine leaks
                continue
            }
        }
    }
    
    // If we didn't get enough trees, generate more using the standard approach
    if len(resultTrees) < count {
        log.Printf("DEBUG: Only found %d unique trees from goroutines, generating %d more trees", 
                  len(resultTrees), count-len(resultTrees))
        
        for i := len(resultTrees); i < count; i++ {
            visitCount := 0
            visited := make(map[string]bool)
            tree := buildElementTreeDFSNoCircular(g, elementName, visited, &visitCount)
            
            // Check if this tree is unique
            signature := generateTreeSignature(tree)
            if !uniqueSignatures[signature] {
                uniqueSignatures[signature] = true
                resultTrees = append(resultTrees, tree)
                totalVisitedCount += visitCount
            }
            
            // Break if we can't generate more unique trees
            if len(resultTrees) >= count || len(uniqueSignatures) >= count*2 {
                break
            }
        }
    }
    
    log.Printf("DEBUG: Final result contains %d unique trees", len(resultTrees))
    return resultTrees, totalVisitedCount
}

// Helper function to generate variations of recipes based on different paths to make ingredients
func generateRecipeVariations(
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
    // Base case: if we're too deep in the recursion, just generate one variation
    if depth >= 2 {
        *activeGoroutines++
        go func() {
            // Create a tree for this recipe
            visitCount := 0
            tree := map[string]interface{}{
                "name":        elementName,
                "imagePath":   imagePath,
                "ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
            }
            
            // Build subtrees for each ingredient
            for _, ingredientName := range recipe.Ingredients {
                visited := make(map[string]bool)
                ingredientVisitCount := 0
                ingredientTree := buildElementTreeDFSNoCircular(g, ingredientName, visited, &ingredientVisitCount)
                tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingredientTree)
                visitCount += ingredientVisitCount
            }
            
            resultChan <- tree
            visitCountChan <- visitCount
        }()
        return
    }
    
    // Check if this recipe has any ingredient with multiple recipes
    hasMultipleRecipes := false
    for _, ingredient := range recipe.Ingredients {
        if ingNode := g.Nodes[ingredient]; ingNode != nil && len(ingNode.RecipesToMakeThisElement) > 1 {
            hasMultipleRecipes = true
            break
        }
    }
    
    // If no ingredient has multiple recipes, just generate one variation
    if !hasMultipleRecipes {
        *activeGoroutines++
        go func() {
            // Create a tree for this recipe
            visitCount := 0
            tree := map[string]interface{}{
                "name":        elementName,
                "imagePath":   imagePath,
                "ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
            }
            
            // Build subtrees for each ingredient
            for _, ingredientName := range recipe.Ingredients {
                visited := make(map[string]bool)
                ingredientVisitCount := 0
                ingredientTree := buildElementTreeDFSNoCircular(g, ingredientName, visited, &ingredientVisitCount)
                tree["ingredients"] = append(tree["ingredients"].([]interface{}), ingredientTree)
                visitCount += ingredientVisitCount
            }
            
            resultChan <- tree
            visitCountChan <- visitCount
        }()
        return
    }
    
    // Find ingredients with multiple recipes
    ingredientsWithMultipleRecipes := make([]string, 0)
    for _, ingredient := range recipe.Ingredients {
        if ingNode := g.Nodes[ingredient]; ingNode != nil && len(ingNode.RecipesToMakeThisElement) > 1 {
            ingredientsWithMultipleRecipes = append(ingredientsWithMultipleRecipes, ingredient)
        }
    }
    
    // Limit the number of variations to explore
    maxVariations := maxCount / 2
    if maxVariations < 1 {
        maxVariations = 1
    }
    
    // Generate variations based on the first ingredient with multiple recipes
    if len(ingredientsWithMultipleRecipes) > 0 {
        variationIngredient := ingredientsWithMultipleRecipes[0]
        ingNode := g.Nodes[variationIngredient]
        
        // For each recipe of this ingredient, generate a variation
        numRecipes := len(ingNode.RecipesToMakeThisElement)
        recipesToExplore := numRecipes
        if recipesToExplore > maxVariations {
            recipesToExplore = maxVariations
        }
        
        for i := 0; i < recipesToExplore; i++ {
            *activeGoroutines++
            
            // Capture the recipe index to use
            recipeIndex := i % numRecipes
            
            go func(ingredientRecipeIndex int) {
                // Create a tree for this recipe
                visitCount := 0
                tree := map[string]interface{}{
                    "name":        elementName,
                    "imagePath":   imagePath,
                    "ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
                }
                
                // Build subtrees for each ingredient
                for _, ingredientName := range recipe.Ingredients {
                    var ingredientTree map[string]interface{}
                    visited := make(map[string]bool)
                    ingredientVisitCount := 0
                    
                    // For the variation ingredient, use a specific recipe
                    if ingredientName == variationIngredient {
                        // Use specific recipe for this ingredient
                        ingredientTree = buildIngredientTreeWithSpecificRecipe(
                            g, ingredientName, ingNode.ImagePath, 
                            ingredientRecipeIndex, visited, &ingredientVisitCount)
                    } else {
                        // For other ingredients, use the standard approach
                        ingredientTree = buildElementTreeDFSNoCircular(
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

// Helper function to build an ingredient tree using a specific recipe
func buildIngredientTreeWithSpecificRecipe(
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
    
    // Get the recipes to make this element
    if len(node.RecipesToMakeThisElement) == 0 {
        // No recipe found
        return map[string]interface{}{
            "name":        elementName,
            "imagePath":   imagePath,
            "ingredients": []interface{}{},
            "noRecipe":    true,
        }
    }
    
    // Use the specified recipe if available, otherwise use the first one
    var recipe *graph.Recipe
    if recipeIndex >= 0 && recipeIndex < len(node.RecipesToMakeThisElement) {
        recipe = node.RecipesToMakeThisElement[recipeIndex]
    } else {
        recipe = node.RecipesToMakeThisElement[0]
    }
    
    // Build the ingredients tree
    ingredients := make([]interface{}, 0, len(recipe.Ingredients))
    for _, ingredientName := range recipe.Ingredients {
        ingredientTree := buildElementTreeDFSNoCircular(g, ingredientName, visited, visitedCount)
        ingredients = append(ingredients, ingredientTree)
    }
    
    return map[string]interface{}{
        "name":        elementName,
        "imagePath":   imagePath,
        "ingredients": ingredients,
    }
}
// Modified function to handle multiple recipes tree with support for BFS and DFS algorithms
// Modified function to handle multiple recipes tree with support for BFS and DFS algorithms

func (h *Handler) HandleMultipleRecipesTree(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    log.Printf("DEBUG: Starting HandleMultipleRecipesTree request")
    
    // Extract parameters
    pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/multiple-recipes-tree/"), "/")
    if len(pathParts) < 1 {
        http.Error(w, "Invalid URL format. Use /api/multiple-recipes-tree/{elementName}?count=N&algorithm=algo", http.StatusBadRequest)
        return
    }
    
    elementName := strings.Join(pathParts, "/")
    log.Printf("DEBUG: Requested element: %s", elementName)
    
    // Get number of recipes to return
    count := 3 // Default to 3 different recipes
    if countParam := r.URL.Query().Get("count"); countParam != "" {
        if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
            count = parsedCount
        }
    }
    log.Printf("DEBUG: Requested recipe tree count: %d", count)
    
    // Get algorithm to use (default to DFS if not specified)
    algorithm := "dfs" // Default to DFS
    if algoParam := r.URL.Query().Get("algorithm"); algoParam != "" {
        algorithm = strings.ToLower(algoParam)
    }
    log.Printf("DEBUG: Using algorithm: %s", algorithm)
    
    // Limit maximum recipes to prevent performance issues
    if count > 10 {
        count = 10
        log.Printf("DEBUG: Limiting count to maximum of 10 for tree format")
    }
    
    // Check if element exists
    element, exists := h.elements[elementName]
    if !exists {
        http.Error(w, "Element not found", http.StatusNotFound)
        log.Printf("DEBUG: Element '%s' not found in database", elementName)
        return
    }
    
    // For base elements, return simple result
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    for _, base := range baseElements {
        if elementName == base {
            log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
            result := map[string]interface{}{
                "trees": []map[string]interface{}{{
                    "name":          elementName,
                    "imagePath":     element.ImagePath,
                    "ingredients":   []interface{}{},
                    "isBaseElement": true,
                }},
                "nodesVisited": 1,
                "timeElapsed":  0,
                "algorithm":    algorithm,
            }
            
            if err := json.NewEncoder(w).Encode(result); err != nil {
                http.Error(w, "Failed to encode response", http.StatusInternalServerError)
                log.Printf("Error encoding response: %v", err)
            }
            return
        }
    }
    
    startTime := time.Now()
    g := createElementGraph(h.elements)
    
    var finalTrees []map[string]interface{}
    var totalVisitedNodesCount int
    
    // Generate all recipe trees including all variations from child recipes
    if algorithm == "bfs" {
        // Use multithreaded BFS
        paths, visited := alg.MultiThreadedBFS(h.elements, elementName, count*3, false)
        totalVisitedNodesCount = visited
        
        // Convert paths to trees
        uniqueTrees := make([]map[string]interface{}, 0)
        uniqueSignatures := make(map[string]bool)
        
        for _, path := range paths {
            if len(path) < 2 {
                continue // Skip paths that are too short
            }
            
            // Create a tree from this path
            pathVisitCount := 0
            tree := convertPathToCompleteTree(path, h.elements, &pathVisitCount, algorithm)
            
            // Check if this tree is unique
            signature := generateDetailedTreeSignature(tree)
            if !uniqueSignatures[signature] {
                uniqueSignatures[signature] = true
                uniqueTrees = append(uniqueTrees, tree)
                totalVisitedNodesCount += pathVisitCount
                
                if len(uniqueTrees) >= count {
                    break
                }
            }
        }
        
        finalTrees = uniqueTrees
    } else {
        // Use the new recursive recipe variation generator
        trees, visited := GenerateAllRecipeVariations(g, elementName, element.ImagePath, count)
        finalTrees = trees
        totalVisitedNodesCount = visited
        log.Printf("DEBUG: Generated %d unique recipe trees after visiting %d nodes", 
                 len(trees), visited)
    }
    
    // If we still don't have any trees, add a fallback
    if len(finalTrees) == 0 {
        visited := make(map[string]bool)
        visitCount := 0
        var tree map[string]interface{}
        
        if algorithm == "bfs" {
            tree = buildElementTreeBFS(g, elementName, visited, &visitCount)
        } else {
            tree = buildElementTreeDFSNoCircular(g, elementName, visited, &visitCount)
        }
        
        finalTrees = []map[string]interface{}{tree}
        totalVisitedNodesCount += visitCount
        log.Printf("DEBUG: Added fallback element tree using %s (nodes visited: %d)",
            strings.ToUpper(algorithm), visitCount)
    }
    
    result := map[string]interface{}{
        "trees":        finalTrees,
        "nodesVisited": totalVisitedNodesCount,
        "timeElapsed":  time.Since(startTime).Milliseconds(),
        "algorithm":    algorithm,
    }
    
    if err := json.NewEncoder(w).Encode(result); err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        log.Printf("Error encoding response: %v", err)
        return
    }
    
    log.Printf("DEBUG: Successfully sent response with %d recipe trees", len(finalTrees))
}

// New helper function to verify that a tree has all ingredients for one of the available recipes
func verifyTreeIngredientsComplete(tree map[string]interface{}, availableRecipes []*graph.Recipe) bool {
	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok {
		return false
	}

	// Extract ingredient names from the tree
	treeIngredientNames := make([]string, 0)
	for _, ing := range ingredients {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				treeIngredientNames = append(treeIngredientNames, name)
			}
		}
	}

	// Check if the ingredient set matches any of the available recipes
	for _, recipe := range availableRecipes {
		if len(recipe.Ingredients) != len(treeIngredientNames) {
			continue // Skip if ingredient count doesn't match
		}

		// Check if all recipe ingredients are in the tree
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

// Helper function to convert a path to a complete tree, ensuring all ingredients are included
func convertPathToCompleteTree(path []model.Node, elements map[string]model.Element, visitCount *int, algorithm string) map[string]interface{} {
	if len(path) == 0 {
		return nil
	}

	*visitCount += len(path)

	// Process the first node in the path (target element)
	targetElement := path[0].Element
	targetImagePath := path[0].ImagePath

	// Base case: if only one element, return it as a leaf node
	if len(path) == 1 {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
		}
	}

	// Base elements check
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

	// Build graph to find recipe information
	g := createElementGraph(elements)
	node := g.Nodes[targetElement]

	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	// Find a recipe that matches ingredients in the path
	var matchedRecipe *graph.Recipe
	var matchedIngredients []interface{}

	for _, recipe := range node.RecipesToMakeThisElement {
		// Track how many ingredients we've matched
		ingredientMatches := 0
		ingredientTrees := make([]interface{}, 0, len(recipe.Ingredients))

		// Try to find each recipe ingredient in the path
		for _, ingredientName := range recipe.Ingredients {
			// Find this ingredient in the path
			for i := 1; i < len(path); i++ {
				if path[i].Element == ingredientName {
					// Create a subtree for this ingredient
					subVisitCount := 0
					subTree := convertPathToSubtree(path[i:], elements, &subVisitCount, algorithm)
					*visitCount += subVisitCount

					ingredientTrees = append(ingredientTrees, subTree)
					ingredientMatches++
					break
				}
			}
		}

		// If we matched all ingredients, use this recipe
		if ingredientMatches == len(recipe.Ingredients) {
			matchedRecipe = recipe
			matchedIngredients = ingredientTrees
			break
		}
	}

	// If we found a matching recipe, use it
	if matchedRecipe != nil && len(matchedIngredients) == len(matchedRecipe.Ingredients) {
		return map[string]interface{}{
			"name":        targetElement,
			"imagePath":   targetImagePath,
			"ingredients": matchedIngredients,
		}
	}

	// If we couldn't match a recipe from the path, try to construct one
	// First, get the most common recipe (the one with fewest ingredients)
	var bestRecipe *graph.Recipe
	bestIngredientCount := 999

	for _, recipe := range node.RecipesToMakeThisElement {
		if len(recipe.Ingredients) < bestIngredientCount {
			bestRecipe = recipe
			bestIngredientCount = len(recipe.Ingredients)
		}
	}

	// Build a tree using this recipe
	ingredients := make([]interface{}, 0, len(bestRecipe.Ingredients))
	for _, ingredientName := range bestRecipe.Ingredients {
		subVisitCount := 0
		visited := make(map[string]bool)

		var ingredientTree map[string]interface{}
		if algorithm == "bfs" {
			ingredientTree = buildElementTreeBFS(g, ingredientName, visited, &subVisitCount)
		} else if algorithm == "dfs" {
			ingredientTree = buildElementTreeDFS(g, ingredientName, visited, &subVisitCount)
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

// Helper to convert a subpath to a tree
func convertPathToSubtree(subPath []model.Node, elements map[string]model.Element, visitCount *int, algorithm string) map[string]interface{} {
	if len(subPath) == 0 {
		return nil
	}

	*visitCount += 1

	elementName := subPath[0].Element
	imagePath := subPath[0].ImagePath

	// Check if it's a base element
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

	// If this is the only element left in the path, it's a leaf node
	if len(subPath) == 1 {
		// Build a proper tree for it to ensure it has the right ingredients
		g := createElementGraph(elements)
		visited := make(map[string]bool)
		subVisitCount := 0

		// Use algorithm passed from parent
		var tree map[string]interface{}
		if algorithm == "bfs" {
			tree = buildElementTreeBFS(g, elementName, visited, &subVisitCount)
			log.Printf("DEBUG: Using BFS for leaf node %s", elementName)
		} else if algorithm == "dfs" {
			tree = buildElementTreeDFS(g, elementName, visited, &subVisitCount)
			log.Printf("DEBUG: Using DFS for leaf node %s", elementName)
		}

		*visitCount += subVisitCount
		return tree
	}

	// Otherwise, recursively build a tree
	return convertPathToCompleteTree(subPath, elements, visitCount, algorithm)
}

// Modified to accept algorithm parameter
func generateTreesForRecipe(
	g *graph.ElementGraph,
	elementName string,
	imagePath string,
	recipe *graph.Recipe,
	visitedNodesCount *int,
	maxCount int,
	algorithm string,
) []map[string]interface{} {
	// Base case: no more ingredients to process
	if len(recipe.Ingredients) == 0 {
		return []map[string]interface{}{{
			"name":        elementName,
			"imagePath":   imagePath,
			"ingredients": []interface{}{},
		}}
	}

	// Create a tree structure for this element
	baseTree := map[string]interface{}{
		"name":        elementName,
		"imagePath":   imagePath,
		"ingredients": []interface{}{},
	}

	// Iterate through all ingredients and build their trees
	ingredients := make([]interface{}, 0, len(recipe.Ingredients))

	for _, ingredient := range recipe.Ingredients {
		// Skip null ingredients
		ingNode := g.Nodes[ingredient]
		if ingNode == nil {
			log.Printf("DEBUG: Ingredient %s not found in graph", ingredient)
			continue
		}

		*visitedNodesCount++

		// Generate a tree for this ingredient using the specified algorithm
		visited := make(map[string]bool)
		ingVisitCount := 0
		var ingredientTree map[string]interface{}

		if algorithm == "bfs" {
			ingredientTree = buildElementTreeBFS(g, ingredient, visited, &ingVisitCount)
		} else if algorithm == "dfs" {
			ingredientTree = buildElementTreeDFS(g, ingredient, visited, &ingVisitCount)
		}

		*visitedNodesCount += ingVisitCount

		ingredients = append(ingredients, ingredientTree)
	}

	// Make sure all ingredients are included
	if len(ingredients) != len(recipe.Ingredients) {
		log.Printf("DEBUG: Not all ingredients could be processed for recipe %s", elementName)
		return nil
	}

	// Create the complete tree with all ingredients
	baseTree["ingredients"] = ingredients

	return []map[string]interface{}{baseTree}
}

// Helper to compare trees deeply (including all ingredient paths)
func compareTreeIngredientsDeep(tree1, tree2 map[string]interface{}) bool {
	// Check if the trees have the same name
	name1, _ := tree1["name"].(string)
	name2, _ := tree2["name"].(string)

	if name1 != name2 {
		return false
	}

	// Get ingredients for both trees
	ingredients1, ok1 := tree1["ingredients"].([]interface{})
	ingredients2, ok2 := tree2["ingredients"].([]interface{})

	// Different number of ingredients means different trees
	if !ok1 || !ok2 || len(ingredients1) != len(ingredients2) {
		return false
	}

	if len(ingredients1) == 0 {
		return true // Empty ingredients means same tree
	}

	// Compare each ingredient recursively
	// Create maps of ingredient trees by name for comparison
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

	// Different ingredient names means different trees
	if len(ingMap1) != len(ingMap2) {
		return false
	}

	// Check if each ingredient in tree1 has a matching ingredient in tree2
	for name, ing1 := range ingMap1 {
		ing2, exists := ingMap2[name]
		if !exists {
			return false // Ingredient not found in tree2
		}

		// Recursively compare this ingredient's subtrees
		if !compareTreeIngredientsDeep(ing1, ing2) {
			return false
		}
	}

	return true
}

// Helper to deep copy a tree
func deepCopyTree(tree map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range tree {
		if key == "ingredients" {
			if ingredients, ok := value.([]interface{}); ok {
				copiedIngredients := make([]interface{}, 0, len(ingredients))
				for _, ing := range ingredients {
					if ingMap, ok := ing.(map[string]interface{}); ok {
						copiedIngredients = append(copiedIngredients, deepCopyTree(ingMap))
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

// Helper function to compare if two trees have the same ingredients
func compareTreeIngredients(tree1, tree2 map[string]interface{}) bool {
	// Check if the trees have the same name
	name1, _ := tree1["name"].(string)
	name2, _ := tree2["name"].(string)

	if name1 != name2 {
		return false
	}

	// Get ingredients for both trees
	ingredients1, _ := tree1["ingredients"].([]interface{})
	ingredients2, _ := tree2["ingredients"].([]interface{})

	// Different number of ingredients means different trees
	if len(ingredients1) != len(ingredients2) {
		return false
	}

	// Compare each ingredient by name
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

	// Sort names to ensure we're comparing properly
	sort.Strings(ingNames1)
	sort.Strings(ingNames2)

	// Check if ingredient lists match
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

// HandleMultipleRecipes gets multiple different recipe paths using DFS
// HandleMultipleRecipes gets multiple different recipe paths using DFS
// HandleMultipleRecipes gets multiple different recipe paths using DFS
func (h *Handler) HandleMultipleRecipes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Start a debug log for this request
	log.Printf("DEBUG: Starting HandleMultipleRecipes request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/multiple-recipes/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/multiple-recipes/{elementName}?count=N", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	// Get number of recipes to return
	count := 5 // Default to 5 different recipes
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)

	// Limit maximum recipes to prevent performance issues
	if count > 10 {
		count = 10
		log.Printf("DEBUG: Limiting count to maximum of 10")
	}

	// Check if element exists
	element, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// For base elements, return simple result
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			result := model.SearchResult{
				Paths: [][]model.Node{{{
					Element:   elementName,
					ImagePath: element.ImagePath,
				}}},
				NodesVisited: 0,
				TimeElapsed:  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	log.Printf("DEBUG: Starting DFS search for element '%s'", elementName)
	startTime := time.Now()

	// Use DFS to find paths with a reasonable limit
	explorationLimit := count * 2
	if explorationLimit > 20 {
		explorationLimit = 20 // Cap at 20 to prevent runaway processes
		log.Printf("DEBUG: Limiting exploration to 20 paths")
	} else {
		log.Printf("DEBUG: Setting exploration limit to %d paths", explorationLimit)
	}

	// Enable debug mode for backward DFS
	paths, visited := alg.DFS(h.elements, elementName, explorationLimit, true)

	// Log search information
	log.Printf("DEBUG: DFS visited %d nodes", visited)
	log.Printf("DEBUG: DFS found %d paths", len(paths))

	log.Printf("DEBUG: Grouping paths by base elements used")
	// Group paths by their base elements (the leaf nodes they use)
	pathGroups := make(map[string][]model.Node)

	for i, path := range paths {
		if len(path) < 3 {
			log.Printf("DEBUG: Skipping path %d (too short, only %d nodes)", i, len(path))
			continue // Skip paths that are too short
		}

		// Create a fingerprint based on the base elements used
		var baseElementsUsed []string
		for _, node := range path {
			isBaseElement := false
			for _, base := range baseElements {
				if node.Element == base {
					baseElementsUsed = append(baseElementsUsed, base)
					isBaseElement = true
				}
			}
			if !isBaseElement && len(node.Ingredients) == 0 {
				baseElementsUsed = append(baseElementsUsed, node.Element)
			}
		}

		sort.Strings(baseElementsUsed)
		fingerprint := strings.Join(baseElementsUsed, ",")
		log.Printf("DEBUG: Path %d has fingerprint: %s", i, fingerprint)

		if _, exists := pathGroups[fingerprint]; !exists {
			pathGroups[fingerprint] = path
			log.Printf("DEBUG: Added path with unique fingerprint: %s", fingerprint)
		}
	}

	log.Printf("DEBUG: Found %d unique path groups", len(pathGroups))

	// Collect diverse paths
	diversePaths := make([][]model.Node, 0)
	for fingerprint, path := range pathGroups {
		diversePaths = append(diversePaths, path)
		log.Printf("DEBUG: Selected path with fingerprint: %s", fingerprint)
		if len(diversePaths) >= count {
			log.Printf("DEBUG: Reached requested count of %d diverse paths", count)
			break
		}
	}

	// If we don't have enough diverse paths, add more from the original paths
	if len(diversePaths) < count && len(paths) > len(diversePaths) {
		log.Printf("DEBUG: Not enough diverse paths (%d/%d), adding more from original paths",
			len(diversePaths), count)

		// Sort paths by length to prioritize simpler recipes
		sort.Slice(paths, func(i, j int) bool {
			return len(paths[i]) < len(paths[j])
		})
		log.Printf("DEBUG: Sorted original paths by length (shortest first)")

		// Add paths that aren't already included
		for i, path := range paths {
			if len(diversePaths) >= count {
				break
			}

			// Skip already included paths
			isIncluded := false
			for _, dp := range diversePaths {
				// Simple comparison - if they have the same start and end elements
				if len(path) > 0 && len(dp) > 0 &&
					path[0].Element == dp[0].Element &&
					path[len(path)-1].Element == dp[len(dp)-1].Element {
					isIncluded = true
					break
				}
			}

			if !isIncluded {
				diversePaths = append(diversePaths, path)
				log.Printf("DEBUG: Added additional path %d (length: %d)", i, len(path))
			}
		}
	}

	log.Printf("DEBUG: Final diverse path count: %d", len(diversePaths))

	// Ensure all nodes have image paths
	for i := range diversePaths {
		for j := range diversePaths[i] {
			elem := diversePaths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && diversePaths[i][j].ImagePath == "" {
				diversePaths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}

	log.Printf("DEBUG: Processing completed in %d ms", time.Since(startTime).Milliseconds())

	result := model.SearchResult{
		Paths:        diversePaths,
		NodesVisited: visited,
		TimeElapsed:  time.Since(startTime).Milliseconds(),
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response with %d recipes", len(diversePaths))
}

// Add this new function to your Handler struct
func (h *Handler) HandleElementTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract the element name and algorithm from the URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tree/"), "/")

	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL format. Use /api/tree/{algorithm}/{elementName}", http.StatusBadRequest)
		return
	}

	algorithm := pathParts[0]
	elementName := strings.Join(pathParts[1:], "/") // In case element name has slashes

	// Validate the element exists
	targetElement, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		return
	}

	// For base elements, return just the element itself with empty ingredients
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			result := map[string]interface{}{
				"name":          elementName,
				"imagePath":     targetElement.ImagePath,
				"ingredients":   []interface{}{},
				"isBaseElement": true,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	startTime := time.Now()

	var result map[string]interface{}
	var visitedNodes int

	// Build a graph first
	g := createElementGraph(h.elements)

	// Choose algorithm
	switch strings.ToLower(algorithm) {
	case "bfs":
		result, visitedNodes = getElementTreeBFS(g, elementName)
	case "dfs":
		result, visitedNodes = getElementTreeDFS(g, elementName)
	default:
		http.Error(w, "Invalid algorithm. Use 'bfs' or 'dfs'", http.StatusBadRequest)
		return
	}

	// Add metadata
	finalResult := map[string]interface{}{
		"tree":         result,
		"nodesVisited": visitedNodes,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
	}

	if err := json.NewEncoder(w).Encode(finalResult); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

func createElementGraph(elements map[string]model.Element) *graph.ElementGraph {
	return graph.NewElementGraph(elements)
}

// Gets element tree using BFS approach
func getElementTreeBFS(g *graph.ElementGraph, elementName string) (map[string]interface{}, int) {
	visited := make(map[string]bool)
	visitedCount := 0

	result := buildElementTreeBFS(g, elementName, visited, &visitedCount)
	return result, visitedCount
}

// Gets element tree using DFS approach
func getElementTreeDFS(g *graph.ElementGraph, elementName string) (map[string]interface{}, int) {
	visited := make(map[string]bool)
	visitedCount := 0

	result := buildElementTreeDFS(g, elementName, visited, &visitedCount)
	return result, visitedCount
}

func buildElementTreeBFS(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
	if visited[elementName] {
		// If we've seen this element before, just return its info without recursion
		node := g.Nodes[elementName]
		return map[string]interface{}{
			"name":                elementName,
			"imagePath":           node.ImagePath,
			"isCircularReference": true,
		}
	}

	visited[elementName] = true
	*visitedCount++

	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Check if it's a base element
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

	// Get the first recipe to make this element (BFS takes the first recipe found)
	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found, might be a base element not in our list
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	// Choose first recipe (BFS approach)
	recipe := node.RecipesToMakeThisElement[0]
	ingredients := make([]interface{}, 0, len(recipe.Ingredients))

	// Process ingredients in order
	for _, ingredientName := range recipe.Ingredients {
		ingredientTree := buildElementTreeBFS(g, ingredientName, visited, visitedCount)
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        elementName,
		"imagePath":   node.ImagePath,
		"ingredients": ingredients,
	}
}

// Recursive function to build element tree using DFS (deeper exploration)
// Recursive function to build element tree using DFS (deeper exploration)
func buildElementTreeDFS(g *graph.ElementGraph, elementName string, visited map[string]bool, visitedCount *int) map[string]interface{} {
	if visited[elementName] {
		// If we've seen this element before, just return its info without recursion
		node := g.Nodes[elementName]
		return map[string]interface{}{
			"name":                elementName,
			"imagePath":           node.ImagePath,
			"isCircularReference": true,
		}
	}

	visited[elementName] = true
	*visitedCount++

	node := g.Nodes[elementName]
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Check if it's a base element
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

	// Get the recipes to make this element
	if len(node.RecipesToMakeThisElement) == 0 {
		// No recipe found, might be a base element not in our list
		return map[string]interface{}{
			"name":        elementName,
			"imagePath":   node.ImagePath,
			"ingredients": []interface{}{},
			"noRecipe":    true,
		}
	}

	// Find the recipe with the shortest combined ingredient path length
	// This uses DFS to find the recipe requiring fewest steps
	var bestRecipe *graph.Recipe
	var bestPathLength = 9999 // Start with a high value

	// Try all recipes
	for _, recipe := range node.RecipesToMakeThisElement {
		// Skip recursive recipes (where an element is used to make itself)
		selfReferential := false
		for _, ing := range recipe.Ingredients {
			if ing == elementName {
				selfReferential = true
				break
			}
		}
		if selfReferential {
			continue
		}

		// Calculate approximate path length without fully exploring
		// This is just a heuristic to pick a reasonable recipe
		totalPathLength := 0
		for _, ingredient := range recipe.Ingredients {
			// Base elements have path length 1
			if isBaseElementName(ingredient, baseElements) {
				totalPathLength += 1
			} else if ingNode, exists := g.Nodes[ingredient]; exists {
				// Add 1 for each level of recipes needed
				if len(ingNode.RecipesToMakeThisElement) > 0 {
					totalPathLength += 2
				} else {
					totalPathLength += 1
				}
			}
		}

		// Choose this recipe if it's the shortest so far
		if totalPathLength < bestPathLength {
			bestPathLength = totalPathLength
			bestRecipe = recipe
		}
	}

	// If no valid recipe was found, use the first one
	if bestRecipe == nil && len(node.RecipesToMakeThisElement) > 0 {
		bestRecipe = node.RecipesToMakeThisElement[0]
	}

	// Build the ingredients tree
	ingredients := make([]interface{}, 0, len(bestRecipe.Ingredients))
	for _, ingredientName := range bestRecipe.Ingredients {
		ingredientTree := buildElementTreeDFS(g, ingredientName, visited, visitedCount)
		ingredients = append(ingredients, ingredientTree)
	}

	return map[string]interface{}{
		"name":        elementName,
		"imagePath":   node.ImagePath,
		"ingredients": ingredients,
	}
}

func isBaseElementName(name string, baseElements []string) bool {
	for _, base := range baseElements {
		if name == base {
			return true
		}
	}
	return false
}

// Helper function to check if an element is a base element
func (h *Handler) HandleBestRecipes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleBestRecipes request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/best-recipes/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/best-recipes/{elementName}?count=N", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	// Get number of recipes to return
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe count: %d", count)

	// Limit maximum count to prevent performance issues
	if count > 10 {
		count = 10
		log.Printf("DEBUG: Limiting count to maximum of 10")
	}

	// Check if element exists
	element, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// For base elements, return simple result
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			result := model.SearchResult{
				Paths: [][]model.Node{{{
					Element:   elementName,
					ImagePath: element.ImagePath,
				}}},
				NodesVisited: 0,
				TimeElapsed:  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	log.Printf("DEBUG: Starting DFS search for element '%s'", elementName)
	startTime := time.Now()

	// Find multiple paths using DFS
	// Set a reasonable maxResults to not explore too many paths
	maxResults := count + 5
	if maxResults > 20 {
		maxResults = 20 // Cap at 20 to prevent excessive exploration
		log.Printf("DEBUG: Limiting exploration to 20 paths")
	}

	paths, visited := alg.DFS(h.elements, elementName, maxResults, false)
	log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)

	// Sort paths by length (shorter paths first)
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j])
	})
	log.Printf("DEBUG: Sorted paths by length (shortest first)")

	// Take only the requested number of best paths
	if len(paths) > count {
		paths = paths[:count]
		log.Printf("DEBUG: Taking only the top %d shortest paths", count)
	}

	// Ensure all nodes have image paths
	for i := range paths {
		for j := range paths[i] {
			elem := paths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && paths[i][j].ImagePath == "" {
				paths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}

	result := model.SearchResult{
		Paths:        paths,
		NodesVisited: visited,
		TimeElapsed:  time.Since(startTime).Milliseconds(),
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response with %d recipes", len(paths))
}

func (h *Handler) HandleRecipePath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleRecipePath request")

	// Extract the element name and algorithm from the URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/recipes/"), "/")

	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL format. Use /api/recipes/{algorithm}/{elementName}", http.StatusBadRequest)
		return
	}

	algorithm := pathParts[0]
	elementName := strings.Join(pathParts[1:], "/") // In case element name has slashes

	log.Printf("DEBUG: Requested algorithm: %s, element: %s", algorithm, elementName)

	// Validate the element exists
	if _, exists := h.elements[elementName]; !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// Skip calculation for base elements
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			// For base elements, return simple path
			result := model.SearchResult{
				Paths:        [][]model.Node{{{Element: elementName}}},
				NodesVisited: 1,
				TimeElapsed:  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	// Set search parameters
	config := model.SearchConfig{
		MaxResults: 1,    // Get one path by default
		SinglePath: true, // Stop after finding one path
	}

	// Parse additional query parameters if provided
	if maxResults := r.URL.Query().Get("maxResults"); maxResults != "" {
		var err error
		if config.MaxResults, err = strconv.Atoi(maxResults); err != nil {
			config.MaxResults = 1 // Default to 1 if invalid
		}
	}

	startTime := time.Now()
	var result model.SearchResult

	log.Printf("DEBUG: Starting search with algorithm: %s for element: %s (max results: %d)",
		algorithm, elementName, config.MaxResults)

	// Choose algorithm
	switch strings.ToLower(algorithm) {
	case "bfs":
		paths, visited := alg.BFS(h.elements, elementName, config.MaxResults, true)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: BFS found %d paths after visiting %d nodes", len(paths), visited)
	case "dfs":
		paths, visited := alg.DFS(h.elements, elementName, config.MaxResults, false)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)
	case "bidirectional":
		paths, visited := alg.Bidirectional(h.elements, elementName, config.MaxResults)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: Bidirectional search found %d paths after visiting %d nodes", len(paths), visited)
	default:
		http.Error(w, "Invalid algorithm. Use 'bfs', 'dfs', or 'bidirectional'", http.StatusBadRequest)
		return
	}

	result.TimeElapsed = time.Since(startTime).Milliseconds()

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response in %d ms", time.Since(startTime).Milliseconds())
}

func (h *Handler) HandleGetElements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if path contains a specific element name
	path := strings.TrimPrefix(r.URL.Path, "/api/elements/")
	if path != "" && path != "elements" {
		// If we have an element name in the URL, return that specific element
		elementName := strings.TrimSpace(path)
		element, exists := h.elements[elementName]
		if !exists {
			http.Error(w, "Element not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(element); err != nil {
			http.Error(w, "Failed to encode element", http.StatusInternalServerError)
			log.Printf("Error encoding element: %v", err)
		}
		return
	}

	// Otherwise return all elements
	elementList := make([]model.Element, 0, len(h.elements))
	for _, elem := range h.elements {
		elementList = append(elementList, elem)
	}

	if err := json.NewEncoder(w).Encode(elementList); err != nil {
		http.Error(w, "Failed to encode elements", http.StatusInternalServerError)
		log.Printf("Error encoding elements: %v", err)
		return
	}
}

func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleSearch request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config model.SearchConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request: %v", err)
		return
	}

	if config.TargetElement == "" {
		http.Error(w, "Target element is required", http.StatusBadRequest)
		return
	}

	if config.Algorithm == "" {
		config.Algorithm = "bfs"
	}
	if config.MaxResults <= 0 {
		config.MaxResults = 1
	}

	log.Printf("DEBUG: Searching for %s using %s algorithm (max results: %d, single path: %v)",
		config.TargetElement, config.Algorithm, config.MaxResults, config.SinglePath)

	startTime := time.Now()
	var result model.SearchResult

	switch config.Algorithm {
	case "bfs":
		paths, visited := alg.BFS(h.elements, config.TargetElement, config.MaxResults, true)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: BFS found %d paths after visiting %d nodes", len(paths), visited)
	case "dfs":
		paths, visited := alg.DFS(h.elements, config.TargetElement, config.MaxResults, false)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: DFS found %d paths after visiting %d nodes", len(paths), visited)
	case "bidirectional":
		paths, visited := alg.Bidirectional(h.elements, config.TargetElement, config.MaxResults)
		result.Paths = paths
		result.NodesVisited = visited
		log.Printf("DEBUG: Bidirectional search found %d paths after visiting %d nodes", len(paths), visited)
	default:
		http.Error(w, "Invalid algorithm", http.StatusBadRequest)
		return
	}

	result.TimeElapsed = time.Since(startTime).Milliseconds()

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response in %d ms", time.Since(startTime).Milliseconds())
}

// HandleBFS untuk mencari path dengan algoritma BFS
func (h *Handler) HandleBFS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleBFS request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bfs/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bfs/{elementName}?count=N&singlePath=true", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	// Get number of paths to return
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested path count: %d", count)

	// Get whether to return single path or multiple paths
	singlePath := true
	if singlePathParam := r.URL.Query().Get("singlePath"); singlePathParam != "" {
		parsedValue, err := strconv.ParseBool(singlePathParam)
		if err == nil {
			singlePath = parsedValue
			log.Printf("DEBUG: Parsed singlePath parameter: %v (original value: %s)",
				singlePath, singlePathParam)
		} else {
			log.Printf("DEBUG: Error parsing singlePath parameter: %v", err)
		}
	} else {
		log.Printf("DEBUG: No singlePath parameter provided, using default: %v", singlePath)
	}
	log.Printf("DEBUG: Single path mode: %v", singlePath)

	// Check if element exists
	if _, exists := h.elements[elementName]; !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// For base elements, return simple result
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			element := h.elements[base]
			result := model.SearchResult{
				Paths:        [][]model.Node{{{Element: elementName, ImagePath: element.ImagePath}}},
				NodesVisited: 1,
				TimeElapsed:  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	// Start BFS search with increased exploration for multiple paths
	startTime := time.Now()
	var allPaths [][]model.Node
	var visited int

	// PERUBAHAN UTAMA: Gunakan kombinasi algoritma untuk mendapatkan lebih banyak jalur
	if !singlePath && count > 1 {
		log.Printf("DEBUG: Requesting multiple diverse paths")

		// 1. Jalankan MultiThreadedBFS dengan eksplorasi tinggi
		explorationCount := count * 10 // Meningkatkan eksplorasi
		if explorationCount > 40 {
			explorationCount = 40
		}

		paths1, visited1 := alg.MultiThreadedBFS(h.elements, elementName, explorationCount, false)
		log.Printf("DEBUG: MultiThreadedBFS found %d paths", len(paths1))

		// 2. Jalankan BFS standar untuk mendapatkan jalur alternatif
		paths2, visited2 := alg.BFS(h.elements, elementName, count*2, false)
		log.Printf("DEBUG: Standard BFS found %d additional paths", len(paths2))

		// Gabungkan hasil dan update jumlah node yang dikunjungi
		allPaths = append(paths1, paths2...)
		visited = visited1 + visited2

		log.Printf("DEBUG: Combined %d total paths before filtering", len(allPaths))
	} else {
		log.Printf("DEBUG: Using standard BFS to find a single path")
		allPaths, visited = alg.BFS(h.elements, elementName, 1, true)
	}

	// TAMBAHAN: Validasi tier untuk semua jalur
	log.Printf("DEBUG: Validating tier constraints for %d paths", len(allPaths))
	var validPaths [][]model.Node
	// Ensure this statement is inside a function
	targetTier := h.elements[elementName].Tier

	for i, path := range allPaths {
		valid := true

		// Periksa tier untuk setiap node dalam jalur
		for _, node := range path {
			if node.Element == elementName {
				// Skip target element
				continue
			}

			// Cek apakah tier ingredient lebih tinggi dari target
			if ingredient, exists := h.elements[node.Element]; exists {
				if ingredient.Tier > targetTier {
					log.Printf("DEBUG: Path %d invalid: ingredient %s (tier %d) > target %s (tier %d)",
						i, node.Element, ingredient.Tier, elementName, targetTier)
					valid = false
					break
				}
			}
		}

		if valid {
			validPaths = append(validPaths, path)
		}
	}

	log.Printf("DEBUG: %d paths passed tier validation out of %d total paths",
		len(validPaths), len(allPaths))

	allPaths = validPaths
	timeElapsed := time.Since(startTime).Milliseconds()

	// Ensure all nodes have image paths
	for i := range allPaths {
		for j := range allPaths[i] {
			elem := allPaths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && allPaths[i][j].ImagePath == "" {
				allPaths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}

	// Filter untuk mendapatkan jalur yang beragam
	var finalPaths [][]model.Node

	if !singlePath && len(allPaths) > 1 {
		// Kelompokkan jalur berdasarkan akar mereka untuk memastikan keberagaman
		pathGroups := make(map[string][]model.Node)
		log.Printf("DEBUG: Grouping paths by base elements for diversity")

		for i, path := range allPaths {
			if len(path) < 2 {
				continue // Skip jalur yang terlalu pendek
			}

			// Buat tanda tangan unik berdasarkan elemen dasar yang digunakan
			var baseElementsUsed []string
			for _, node := range path {
				isBase := false
				for _, base := range baseElements {
					if node.Element == base {
						baseElementsUsed = append(baseElementsUsed, base)
						isBase = true
						break
					}
				}

				// Juga tambahkan elemen pertengahan utama untuk keberagaman
				if !isBase && len(node.Ingredients) > 0 {
					// Tambahkan paling banyak 2 elemen pertengahan untuk tanda tangan
					if len(baseElementsUsed) < 5 {
						baseElementsUsed = append(baseElementsUsed, node.Element)
					}
				}
			}

			sort.Strings(baseElementsUsed)
			signature := strings.Join(baseElementsUsed, ",") + fmt.Sprintf("|len:%d", len(path))
			log.Printf("DEBUG: Path %d has signature: %s", i, signature)

			// Jika ini adalah tanda tangan unik, tambahkan ke grup
			if _, exists := pathGroups[signature]; !exists {
				pathGroups[signature] = path
				log.Printf("DEBUG: Added path with unique signature: %s", signature)
			}
		}

		// Ambil jalur yang beragam dari grup
		for _, path := range pathGroups {
			finalPaths = append(finalPaths, path)
			if len(finalPaths) >= count {
				log.Printf("DEBUG: Selected %d diverse paths, stopping", count)
				break
			}
		}

		// Jika masih belum cukup, tambahkan lebih banyak dari semua jalur
		if len(finalPaths) < count && len(allPaths) > len(finalPaths) {
			log.Printf("DEBUG: Still need more paths, adding from all paths")

			// Urutkan jalur berdasarkan panjang (prioritaskan yang lebih pendek)
			sort.Slice(allPaths, func(i, j int) bool {
				return len(allPaths[i]) < len(allPaths[j])
			})

			for _, path := range allPaths {
				if len(finalPaths) >= count {
					break
				}

				// Periksa apakah jalur ini sudah termasuk
				alreadyIncluded := false
				for _, existingPath := range finalPaths {
					if generatePathFingerprint(existingPath) == generatePathFingerprint(path) {
						alreadyIncluded = true
						break
					}
				}

				if !alreadyIncluded {
					finalPaths = append(finalPaths, path)
				}
			}
		}
	} else {
		// Dalam mode jalur tunggal, gunakan semua jalur yang ditemukan
		finalPaths = allPaths
	}

	// Pastikan kita memiliki minimal satu jalur
	if len(finalPaths) == 0 && len(allPaths) > 0 {
		finalPaths = allPaths[:1]
		log.Printf("DEBUG: No diverse paths found, using first available path")
	} else if len(finalPaths) == 0 {
		// Fallback terakhir - gunakan resep langsung untuk membangun jalur
		element := h.elements[elementName]
		if len(element.Recipes) > 0 {
			log.Printf("DEBUG: Creating manual path from first recipe")

			// Buat jalur sederhana dari resep pertama
			recipe := element.Recipes[0]
			path := []model.Node{{Element: elementName, ImagePath: element.ImagePath}}

			// Tambahkan ingredients sebagai node sebelumnya dalam jalur
			for _, ing := range recipe.Ingredients {
				if ingElement, exists := h.elements[ing]; exists {
					// Periksa tier
					if ingElement.Tier <= targetTier {
						path = append([]model.Node{{
							Element:   ing,
							ImagePath: ingElement.ImagePath,
						}}, path...)
					}
				}
			}

			finalPaths = [][]model.Node{path}
		} else {
			// Jika tidak ada resep, kembalikan hanya elemen targetnya
			finalPaths = [][]model.Node{{{Element: elementName, ImagePath: element.ImagePath}}}
			log.Printf("DEBUG: No recipes available, returning just the target element")
		}
	}

	log.Printf("DEBUG: Final result contains %d paths", len(finalPaths))

	result := model.SearchResult{
		Paths:        finalPaths,
		NodesVisited: visited,
		TimeElapsed:  timeElapsed,
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent BFS response with %d paths", len(finalPaths))
}

// Helper function to generate a unique fingerprint for a path
func generatePathFingerprint(path []model.Node) string {
	// Extract all elements and sort them for a consistent signature
	elements := make([]string, 0, len(path))
	for _, node := range path {
		elements = append(elements, node.Element)
	}

	sort.Strings(elements)
	return strings.Join(elements, ",")
}

// HandleMultiThreadedBFSRecipesTree untuk mencari multiple recipes dengan MultiThreadedBFS
func (h *Handler) HandleMultiThreadedBFSRecipesTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("DEBUG: Starting HandleMultiThreadedBFSRecipesTree request")

	// Extract parameters
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/mt-bfs-recipes-tree/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/mt-bfs-recipes-tree/{elementName}?count=N", http.StatusBadRequest)
		return
	}

	elementName := strings.Join(pathParts, "/")
	log.Printf("DEBUG: Requested element: %s", elementName)

	// Get number of recipes to return
	count := 3 // Default to 3 different recipes
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	log.Printf("DEBUG: Requested recipe tree count: %d", count)

	// Limit maximum recipes to prevent performance issues
	if count > 10 {
		count = 10
		log.Printf("DEBUG: Limiting count to maximum of 10 for tree format")
	}

	// Check if element exists
	element, exists := h.elements[elementName]
	if !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		log.Printf("DEBUG: Element '%s' not found in database", elementName)
		return
	}

	// For base elements, return simple result
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			log.Printf("DEBUG: Requested element '%s' is a base element, returning simple result", elementName)
			result := map[string]interface{}{
				"trees": []map[string]interface{}{{
					"name":          elementName,
					"imagePath":     element.ImagePath,
					"ingredients":   []interface{}{},
					"isBaseElement": true,
				}},
				"nodesVisited": 1,
				"timeElapsed":  0,
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	startTime := time.Now()
	g := createElementGraph(h.elements)

	// Use MultiThreadedBFS to find multiple unique paths
	log.Printf("DEBUG: Starting MultiThreadedBFS search for element '%s'", elementName)
	explorationLimit := count * 4 // Request more paths to ensure we get enough unique recipes
	if explorationLimit > 40 {
		explorationLimit = 40 // Limit to prevent excessive resource usage
	}

	paths, visited := alg.MultiThreadedBFS(h.elements, elementName, explorationLimit, false)
	log.Printf("DEBUG: MultiThreadedBFS found %d paths after visiting %d nodes", len(paths), visited)

	// Process the paths into trees
	uniqueTrees := make([]map[string]interface{}, 0)
	uniqueSignatures := make(map[string]bool)
	pathVisitCounts := make([]int, 0)
	totalNodesVisited := visited

	for i, path := range paths {
		if len(uniqueTrees) >= count {
			break
		}

		if len(path) < 2 {
			// Skip paths that are too short
			continue
		}

		// Convert path to tree
		pathVisitCount := 0
		tree := convertPathToCompleteTree(path, h.elements, &pathVisitCount, "bfs")

		// Generate a signature for this tree to check uniqueness
		recipeSignature := generateTreeSignature(tree)
		if !uniqueSignatures[recipeSignature] {
			uniqueSignatures[recipeSignature] = true
			uniqueTrees = append(uniqueTrees, tree)
			pathVisitCounts = append(pathVisitCounts, pathVisitCount)
			log.Printf("DEBUG: Added unique tree #%d from path %d (signature: %s)",
				len(uniqueTrees), i, recipeSignature)
			totalNodesVisited += pathVisitCount
		}
	}

	// If we didn't get enough trees from the paths, try generating more from the recipes directly
	if len(uniqueTrees) < count {
		log.Printf("DEBUG: Not enough unique trees from paths (%d/%d), generating from recipes",
			len(uniqueTrees), count)

		node := g.Nodes[elementName]

		for _, recipe := range node.RecipesToMakeThisElement {
			if len(uniqueTrees) >= count {
				break
			}

			// Skip recipes with no ingredients
			if len(recipe.Ingredients) == 0 {
				continue
			}

			// Generate a tree for this recipe
			recipeVisitCount := 0
			recipeTree := map[string]interface{}{
				"name":        elementName,
				"imagePath":   element.ImagePath,
				"ingredients": make([]interface{}, 0, len(recipe.Ingredients)),
			}

			// Build subtrees for each ingredient
			for _, ingredientName := range recipe.Ingredients {
				ingredientVisited := make(map[string]bool)
				ingredientVisitCount := 0

				// Build tree with BFS
				ingredientTree := buildElementTreeBFS(g, ingredientName, ingredientVisited, &ingredientVisitCount)
				recipeTree["ingredients"] = append(recipeTree["ingredients"].([]interface{}), ingredientTree)
				recipeVisitCount += ingredientVisitCount
			}

			// Check if this tree is unique
			recipeSignature := generateTreeSignature(recipeTree)
			if !uniqueSignatures[recipeSignature] {
				uniqueSignatures[recipeSignature] = true
				uniqueTrees = append(uniqueTrees, recipeTree)
				pathVisitCounts = append(pathVisitCounts, recipeVisitCount)
				totalNodesVisited += recipeVisitCount
				log.Printf("DEBUG: Added unique tree from recipe (signature: %s)", recipeSignature)
			}
		}
	}

	// If we still don't have any trees, build a standard tree
	if len(uniqueTrees) == 0 {
		visited := make(map[string]bool)
		visitCount := 0
		tree := buildElementTreeBFS(g, elementName, visited, &visitCount)

		uniqueTrees = append(uniqueTrees, tree)
		totalNodesVisited += visitCount
		log.Printf("DEBUG: Added fallback element tree using BFS (nodes visited: %d)", visitCount)
	}

	result := map[string]interface{}{
		"trees":        uniqueTrees,
		"nodesVisited": totalNodesVisited,
		"timeElapsed":  time.Since(startTime).Milliseconds(),
		"algorithm":    "multithreaded_bfs",
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}

	log.Printf("DEBUG: Successfully sent response with %d recipe trees", len(uniqueTrees))
}

// Helper function to generate a signature for a tree to check uniqueness
func generateTreeSignature(tree map[string]interface{}) string {
	rootName := tree["name"].(string)

	// Extract ingredient names
	ingredients, ok := tree["ingredients"].([]interface{})
	if !ok || len(ingredients) == 0 {
		return rootName + "|no_ingredients"
	}

	// Collect names of all direct ingredients
	names := make([]string, 0, len(ingredients))
	for _, ing := range ingredients {
		if ingMap, ok := ing.(map[string]interface{}); ok {
			if name, ok := ingMap["name"].(string); ok {
				names = append(names, name)
			}
		}
	}

	// Sort ingredients for consistent signature
	sort.Strings(names)

	// Create signature as "ElementName|ing1,ing2,ing3"
	return rootName + "|" + strings.Join(names, ",")
}
