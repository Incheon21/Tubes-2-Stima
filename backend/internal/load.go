package internal

import (
	"backend/internal/graph"
	"backend/model"
	"backend/utils"
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadElements() (map[string]model.Element, *graph.ElementGraph, error) {
	absPath, err := filepath.Abs("elements.json")
	if err != nil {
		return nil, nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var elementsList []model.Element
	if err := json.NewDecoder(file).Decode(&elementsList); err != nil {
		return nil, nil, err
	}

	// Convert list to map for efficiency
	elementsMap := make(map[string]model.Element, len(elementsList))
	for _, element := range elementsList {
		elementsMap[element.Name] = element
	}

	// Apply tier validation to filter invalid recipes
	elementsMap = utils.ValidateRecipeTiers(elementsMap)

	// Build the graph representation using the validated elements
	elementGraph := graph.NewElementGraph(elementsMap)

	return elementsMap, elementGraph, nil
}
