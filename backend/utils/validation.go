package utils

import (
	"backend/model"
	"log"
)
// ValidateRecipeTiers filters out recipes where any ingredient has a higher tier than the resulting element
// and recipes that contain the "Time" element
func ValidateRecipeTiers(elements map[string]model.Element) map[string]model.Element {
    log.Println("Starting tier validation for all recipes...")

    // Count statistics for logging
    totalRecipes := 0
    invalidRecipes := 0
    timeExcludedRecipes := 0

    // Create a new map with validated recipes
    validatedElements := make(map[string]model.Element)

    // Process each element
    for name, element := range elements {
        validRecipes := make([]model.ElementRecipe, 0)

        // Check each recipe for this element
        for _, recipe := range element.Recipes {
            totalRecipes++
            valid := true

            // Check tier of each ingredient and look for Time element
            for _, ingredientName := range recipe.Ingredients {
                // Check if this recipe contains the Time element
                // Check if this recipe contains the Time element
                if ingredientName == "Time" {
                    log.Printf("Excluding recipe for '%s' because it contains the Time element", name)
                    valid = false
                    break
                }

                ingredient, exists := elements[ingredientName]
                if !exists {
                    // Ingredient not found, log warning but consider valid
                    log.Printf("Warning: Ingredient '%s' for '%s' not found in element database",
                        ingredientName, name)
                    continue
                }

                // If ingredient tier is higher than resulting element, recipe is invalid
                if ingredient.Tier >= element.Tier {
                    log.Printf("Invalid recipe: %s (tier %d) + others → %s (tier %d)",
                        ingredientName, ingredient.Tier, name, element.Tier)
                    valid = false
                    invalidRecipes++
                    break
                }
            }

            // Add valid recipes to the filtered list
            if valid {
                validRecipes = append(validRecipes, recipe)
            }
        }

        // Update element with filtered recipes
        elementCopy := element
        elementCopy.Recipes = validRecipes
        validatedElements[name] = elementCopy
    }

    log.Printf("Tier validation complete: Found %d invalid recipes out of %d total recipes",
        invalidRecipes, totalRecipes)
    log.Printf("Excluded %d recipes containing the Time element", timeExcludedRecipes)

    return validatedElements
}

func isSpecialException(ingredient, result string) bool {
    exceptions := map[string][]string{
        "Water": {"Life"},
    }

    if allowedResults, exists := exceptions[ingredient]; exists {
        for _, allowed := range allowedResults {
            if allowed == result {
                return true
            }
        }
    }

    return false
}

func IsBaseElementName(name string, baseElements []string) bool {
	for _, base := range baseElements {
		if name == base {
			return true
		}
	}
	return false
}
