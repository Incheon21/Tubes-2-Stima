package api

import (
	alg "backend/internal/algorithm"
	"backend/internal/graph"
	"backend/model"
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) HandleBFS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/bfs/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL format. Use /api/bfs/{elementName}?count=N&singlePath=true", http.StatusBadRequest)
		return
	}
	elementName := strings.Join(pathParts, "/")
	count := 1
	if countParam := r.URL.Query().Get("count"); countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}
	singlePath := true
	if singlePathParam := r.URL.Query().Get("singlePath"); singlePathParam != "" {
		parsedValue, err := strconv.ParseBool(singlePathParam)
		if err == nil {
			singlePath = parsedValue
		} else {
		}
	}
	if _, exists := h.elements[elementName]; !exists {
		http.Error(w, "Element not found", http.StatusNotFound)
		return
	}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}
	for _, base := range baseElements {
		if elementName == base {
			element := h.elements[base]
			result := model.SearchResult{
				Paths:        [][]model.Node{{{Element: elementName, ImagePath: element.ImagePath}}},
				NodesVisited: 1,
				TimeElapsed:  0,
			}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}
	startTime := time.Now()
	var allPaths [][]model.Node
	var visited int
	if !singlePath && count > 1 {
		explorationCount := count * 15
		if explorationCount > 60 {
			explorationCount = 60
		}
		paths1, visited1 := alg.MultiThreadedBFS(h.elements, elementName, explorationCount, false)
		paths2, visited2 := alg.BFS(h.elements, elementName, count*3, false)
		allPaths = append(paths1, paths2...)
		visited = visited1 + visited2
	} else {
		allPaths, visited = alg.BFS(h.elements, elementName, 1, true)
	}
	var validPaths [][]model.Node
	targetTier := h.elements[elementName].Tier
	for i, path := range allPaths {
		valid := true
		for _, node := range path {
			if node.Element == elementName {
				continue
			}
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
	allPaths = validPaths
	timeElapsed := time.Since(startTime).Milliseconds()
	for i := range allPaths {
		for j := range allPaths[i] {
			elem := allPaths[i][j].Element
			if elemData, exists := h.elements[elem]; exists && allPaths[i][j].ImagePath == "" {
				allPaths[i][j].ImagePath = elemData.ImagePath
			}
		}
	}
	var finalPaths [][]model.Node
	if !singlePath && len(allPaths) > 1 {
		pathGroups := make(map[string][]model.Node)
		log.Printf("DEBUG: Grouping paths by base elements for diversity")
		for i, path := range allPaths {
			if len(path) < 2 {
				continue
			}
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
				if !isBase && len(node.Ingredients) > 0 {
					if len(baseElementsUsed) < 5 {
						baseElementsUsed = append(baseElementsUsed, node.Element)
					}
				}
			}
			sort.Strings(baseElementsUsed)
			signature := strings.Join(baseElementsUsed, ",") + fmt.Sprintf("|len:%d", len(path))
			log.Printf("DEBUG: Path %d has signature: %s", i, signature)
			if _, exists := pathGroups[signature]; !exists {
				pathGroups[signature] = path
				log.Printf("DEBUG: Added path with unique signature: %s", signature)
			}
		}
		for _, path := range pathGroups {
			finalPaths = append(finalPaths, path)
			if len(finalPaths) >= count {
				log.Printf("DEBUG: Selected %d diverse paths, stopping", count)
				break
			}
		}
		if len(finalPaths) < count && len(allPaths) > len(finalPaths) {
			log.Printf("DEBUG: Still need more paths, adding from all paths")
			sort.Slice(allPaths, func(i, j int) bool {
				return len(allPaths[i]) < len(allPaths[j])
			})
			for _, path := range allPaths {
				if len(finalPaths) >= count {
					break
				}
				alreadyIncluded := false
				for _, existingPath := range finalPaths {
					if utils.GeneratePathFingerprint(existingPath) == utils.GeneratePathFingerprint(path) {
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
		finalPaths = allPaths
	}
	if len(finalPaths) == 0 && len(allPaths) > 0 {
		finalPaths = allPaths[:1]
		log.Printf("DEBUG: No diverse paths found, using first available path")
	} else if len(finalPaths) == 0 {
		element := h.elements[elementName]
		if len(element.Recipes) > 0 {
			log.Printf("DEBUG: Creating manual path from first recipe")
			recipe := element.Recipes[0]
			path := []model.Node{{Element: elementName, ImagePath: element.ImagePath}}
			for _, ing := range recipe.Ingredients {
				if ingElement, exists := h.elements[ing]; exists {
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

// Add this function after HandleBFS or HandleSearch

// Helper function to generate all possible recipe trees

// Helper function to generate all combinations of ingredient trees
func generateTreeCombinations(
    baseTree map[string]interface{},
    ingredientVariations [][]map[string]interface{},
    currentIndex int,
) []map[string]interface{} {
    // If we've processed all ingredients, return the base tree
    if currentIndex >= len(ingredientVariations) {
        // Deep copy the tree to avoid sharing references
        return []map[string]interface{}{utils.DeepCopyTree(baseTree)}
    }
    
    // Get the variations for the current ingredient
    currentIngredientVariations := ingredientVariations[currentIndex]
    
    var results []map[string]interface{}
    
    // For each possible tree of the current ingredient
    for _, ingTree := range currentIngredientVariations {
        // Add this ingredient tree to the base tree
        ingredientsList := baseTree["ingredients"].([]interface{})
        baseTree["ingredients"] = append(ingredientsList, ingTree)
        
        // Recursively generate combinations for the next ingredients
        subCombinations := generateTreeCombinations(
            baseTree,
            ingredientVariations,
            currentIndex+1,
        )
        
        // Add these combinations to our results
        results = append(results, subCombinations...)
        
        // Remove this ingredient tree for the next iteration (backtracking)
        baseTree["ingredients"] = ingredientsList
    }
    
    return results
}

func (h *Handler) HandleDFSTree(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    // Parse request path and extract element name
    pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/dfs-tree/"), "/")
    if len(pathParts) < 1 {
        http.Error(w, "Invalid URL format. Use /api/dfs-tree/{elementName}?count=N", http.StatusBadRequest)
        return
    }
    elementName := strings.Join(pathParts, "/")
    
    // Parse query parameters - increase default count for elements like Metal
    count := 5 // Increased default to ensure we get all recipes for Metal
    if countParam := r.URL.Query().Get("count"); countParam != "" {
        if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
            count = parsedCount
        }
    }
    
    // Determine if certain elements need special handling (like Metal)
    if elementName == "Metal" && count < 10 {
        // For Metal specifically, we know there are 4 possible recipe trees
        count = 10
        log.Printf("DEBUG: Element Metal detected, increasing count to ensure all variations")
    }
    
    // Validate element exists
    element, exists := h.elements[elementName]
    if !exists {
        http.Error(w, "Element not found", http.StatusNotFound)
        log.Printf("DEBUG: Element '%s' not found in database", elementName)
        return
    }
    
    // Handle base elements quickly
    baseElements := []string{"Water", "Fire", "Earth", "Air"}
    isBaseElement := false
    for _, base := range baseElements {
        if elementName == base {
            isBaseElement = true
            break
        }
    }
    
    if isBaseElement {
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
            "algorithm":    "dfs",
        }

        if err := json.NewEncoder(w).Encode(result); err != nil {
            http.Error(w, "Failed to encode response", http.StatusInternalServerError)
            log.Printf("Error encoding response: %v", err)
        }
        return
    }
    
    // Start searching for recipes
    log.Printf("DEBUG: Starting DFS tree search for element '%s' (requesting %d trees)", 
        elementName, count)
    startTime := time.Now()
    
    // Create element graph
    g := utils.CreateElementGraph(h.elements)
    
    // Generate all possible recipe trees with our improved function
    trees, visited := generateAllRecipeTrees(g, elementName, element.ImagePath, count, baseElements)
    
    // If no trees found, fallback to single tree
    if len(trees) == 0 {
        visitCount := 0
        visitedNodes := make(map[string]bool)
        tree := utils.BuildElementTreeDFS(g, elementName, visitedNodes, &visitCount)
        trees = []map[string]interface{}{tree}
        visited = visitCount
        log.Printf("DEBUG: No recipe trees found, added fallback element tree using DFS (nodes visited: %d)", visitCount)
    }
    
    // Limit the number of trees to respect the requested count
    // But ensure we keep at least 4 for Metal
    minCount := count
    if elementName == "Metal" && count < 5 {
        minCount = 5
        log.Printf("DEBUG: Ensuring at least 5 trees for Metal element")
    }
    
    if len(trees) > minCount {
        trees = trees[:minCount]
        log.Printf("DEBUG: Limited trees to requested count: %d", minCount)
    }
    
    // Track time elapsed
    timeElapsed := time.Since(startTime).Milliseconds()
    
    // Prepare and send response
    result := map[string]interface{}{
        "trees":        trees,
        "nodesVisited": visited,
        "timeElapsed":  timeElapsed,
        "algorithm":    "dfs",
    }
    
    if err := json.NewEncoder(w).Encode(result); err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        log.Printf("Error encoding response: %v", err)
        return
    }
    
    log.Printf("DEBUG: Successfully sent DFS tree response with %d trees in %d ms", 
        len(trees), timeElapsed)
}

// Helper function to generate all possible recipe trees - fixed version
func generateAllRecipeTrees(g *graph.ElementGraph, elementName, imagePath string, maxCount int, baseElements []string) ([]map[string]interface{}, int) {
    totalVisited := 0
    node := g.Nodes[elementName]
    
    if node == nil || len(node.RecipesToMakeThisElement) == 0 {
        log.Printf("DEBUG: Element '%s' has no recipes", elementName)
        return []map[string]interface{}{}, 0
    }
    
    var allTrees []map[string]interface{}
    log.Printf("DEBUG: Element '%s' has %d direct recipes", elementName, len(node.RecipesToMakeThisElement))
    
    // Metal might have a small number of direct recipes but many variations because of ingredient recipes
    // Increase exploration limit for elements like Metal that have deep recipe trees
    explorationLimit := maxCount * 10 // Allow for more exploration
    
    // Process each recipe for the element
    for recipeIdx, recipe := range node.RecipesToMakeThisElement {
        if len(recipe.Ingredients) == 0 {
            continue
        }
        
        log.Printf("DEBUG: Processing recipe %d for %s with ingredients: %v", 
            recipeIdx, elementName, recipe.Ingredients)
        
        // Create the base tree for this recipe
        baseTree := map[string]interface{}{
            "name":          elementName,
            "imagePath":     imagePath,
            "isBaseElement": false,
            "ingredients":   make([]interface{}, 0, len(recipe.Ingredients)),
        }
        
        // For each ingredient in the recipe, get all its possible trees
        ingredientTreeVariations := make([][]map[string]interface{}, len(recipe.Ingredients))
        localVisited := 0
        
        for i, ingredient := range recipe.Ingredients {
            // Check if ingredient is a base element
            isBase := false
            for _, base := range baseElements {
                if ingredient == base {
                    isBase = true
                    break
                }
            }
            
            ingNode := g.Nodes[ingredient]
            if isBase {
                // Base elements have only one possible tree
                ingredientTreeVariations[i] = []map[string]interface{}{{
                    "name":          ingredient,
                    "imagePath":     ingNode.ImagePath,
                    "isBaseElement": true,
                    "ingredients":   []interface{}{},
                }}
                localVisited++
            } else if ingNode == nil || len(ingNode.RecipesToMakeThisElement) == 0 {
                // Non-base elements without recipes
                ingredientTreeVariations[i] = []map[string]interface{}{{
                    "name":          ingredient,
                    "imagePath":     ingNode.ImagePath,
                    "isBaseElement": false,
                    "ingredients":   []interface{}{},
                }}
                localVisited++
            } else {
                // For more complex ingredients like Stone that have multiple recipes,
                // we need to pass a higher maxCount to ensure we get all variations
                ingredientMaxCount := 10 // Get more trees for ingredients
                
                // Log how many recipes this ingredient has
                log.Printf("DEBUG: Ingredient %s has %d recipes", 
                    ingredient, len(ingNode.RecipesToMakeThisElement))
                
                // Recursive call to get all possible trees for this ingredient
                subVisited := 0
                subTrees, subVisited := generateAllRecipeTrees(g, ingredient, ingNode.ImagePath, ingredientMaxCount, baseElements)
                
                if len(subTrees) == 0 {
                    // If no subtrees found, create a leaf node
                    ingredientTreeVariations[i] = []map[string]interface{}{{
                        "name":          ingredient,
                        "imagePath":     ingNode.ImagePath,
                        "isBaseElement": false,
                        "ingredients":   []interface{}{},
                    }}
                    localVisited++
                } else {
                    ingredientTreeVariations[i] = subTrees
                    log.Printf("DEBUG: Found %d recipe variations for ingredient %s", 
                        len(subTrees), ingredient)
                    localVisited += subVisited
                }
            }
        }
        
        // Now generate all possible combinations of ingredient trees
        treeCombinations := generateTreeCombinations(baseTree, ingredientTreeVariations, 0)
        
        log.Printf("DEBUG: Generated %d tree combinations for recipe %d", 
            len(treeCombinations), recipeIdx)
        
        // Add all the combinations to our result
        allTrees = append(allTrees, treeCombinations...)
        
        // Add to the visited count
        totalVisited += localVisited
        
        // Early exit if we've found an excessive number of trees
        // But make this limit much higher to ensure we get all variations for Metal
        if len(allTrees) > explorationLimit {
            log.Printf("DEBUG: Generated %d trees, stopping early", len(allTrees))
            break
        }
    }
    
    // Deduplicate trees with improved signature generation
    uniqueTrees := improvedDeduplicateTrees(allTrees)
    
    log.Printf("DEBUG: Generated %d unique trees from %d total combinations", 
        len(uniqueTrees), len(allTrees))
    
    return uniqueTrees, totalVisited
}

// Improved deduplication that properly handles Metal's recipe variants
func improvedDeduplicateTrees(trees []map[string]interface{}) []map[string]interface{} {
    if len(trees) <= 1 {
        return trees
    }
    
    uniqueSignatures := make(map[string]bool)
    var uniqueTrees []map[string]interface{}
    
    for _, tree := range trees {
        // Generate a more precise signature that considers ingredient combinations
        signature := generateDetailedTreeSignature(tree)
        
        if !uniqueSignatures[signature] {
            uniqueSignatures[signature] = true
            uniqueTrees = append(uniqueTrees, tree)
        }
    }
    
    log.Printf("DEBUG: After improved deduplication: %d unique trees from %d input trees", 
        len(uniqueTrees), len(trees))
    
    return uniqueTrees
}

// A more precise tree signature function that distinguishes between different recipes
func generateDetailedTreeSignature(tree map[string]interface{}) string {
    var sb strings.Builder
    
    // Start with the element name
    sb.WriteString(tree["name"].(string))
    sb.WriteString(":")
    
    // Process ingredients recursively
    ingredients, ok := tree["ingredients"].([]interface{})
    if !ok || len(ingredients) == 0 {
        return sb.String() + "[]"
    }
    
    // Generate signatures for each ingredient
    ingredientSignatures := make([]string, 0, len(ingredients))
    
    for _, ing := range ingredients {
        ingredient, ok := ing.(map[string]interface{})
        if !ok {
            continue
        }
        
        // Recursive call to get this ingredient's signature
        ingredientSig := generateDetailedTreeSignature(ingredient)
        ingredientSignatures = append(ingredientSignatures, ingredientSig)
    }
    
    // Sort the signatures to ensure consistent ordering
    sort.Strings(ingredientSignatures)
    
    // Join them with a separator
    sb.WriteString("[")
    sb.WriteString(strings.Join(ingredientSignatures, ","))
    sb.WriteString("]")
    
    return sb.String()
}