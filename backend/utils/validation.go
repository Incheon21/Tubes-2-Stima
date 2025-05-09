package utils

import (
	"backend/model"
	"log"
)

// fungsi buat validasi tier, kalo tier 8 + tier 2 <-- X
func ValidateRecipeTiers(elements map[string]model.Element) map[string]model.Element {
	totalRecipes := 0
	invalidRecipes := 0

	validatedElements := make(map[string]model.Element) /* isinya elemen yang boleh masuk */

	for name, element := range elements {
		validRecipes := make([]model.ElementRecipe, 0)
		for _, recipe := range element.Recipes {
			totalRecipes++
			valid := true

			//cek tiernya
			for _, ingredientName := range recipe.Ingredients {
				ingredient, exists := elements[ingredientName]
				if !exists {
					log.Printf("Warning: Ingredient '%s' for '%s' not found in element database",
						ingredientName, name)
					continue
				}

				// tolak kalo tiernya lebih tinggi
				if ingredient.Tier > element.Tier {
					log.Printf("Invalid recipe: %s (tier %d) + others → %s (tier %d)",
						ingredientName, ingredient.Tier, name, element.Tier)
					valid = false
					invalidRecipes++
					break
				}
			}

			//kalo valid appen
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
