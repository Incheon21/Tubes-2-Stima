package utils

import (
	"backend/model"
	"log"
)

func ValidateRecipeTiers(elements map[string]model.Element) map[string]model.Element {
	log.Println("Starting tier validation for all recipes...")

	totalRecipes := 0
	invalidRecipes := 0
	timeExcludedRecipes := 0

	validatedElements := make(map[string]model.Element)

	for name, element := range elements {
		validRecipes := make([]model.ElementRecipe, 0)

		for _, recipe := range element.Recipes {
			totalRecipes++
			valid := true

			for _, ingredientName := range recipe.Ingredients {
				if ingredientName == "Time" {
					log.Printf("Excluding recipe for '%s' because it contains the Time element", name)
					valid = false
					break
				}

				ingredient, exists := elements[ingredientName]
				if !exists {
					log.Printf("Warning: Ingredient '%s' for '%s' not found in element database",
						ingredientName, name)
					continue
				}

				if ingredient.Tier >= element.Tier {
					log.Printf("Invalid recipe: %s (tier %d) + others → %s (tier %d)",
						ingredientName, ingredient.Tier, name, element.Tier)
					valid = false
					invalidRecipes++
					break
				}
			}

			if valid {
				validRecipes = append(validRecipes, recipe)
			}
		}

		elementCopy := element
		elementCopy.Recipes = validRecipes
		validatedElements[name] = elementCopy
	}

	log.Printf("Tier validation complete: Found %d invalid recipes out of %d total recipes",
		invalidRecipes, totalRecipes)
	log.Printf("Excluded %d recipes containing the Time element", timeExcludedRecipes)

	return validatedElements
}

func IsBaseElementName(name string, baseElements []string) bool {
	for _, base := range baseElements {
		if name == base {
			return true
		}
	}
	return false
}
