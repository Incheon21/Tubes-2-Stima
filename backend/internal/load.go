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

	//buat convert list ke map biar efisien pas lookup
	elementsMap := make(map[string]model.Element, len(elementsList))
	for _, element := range elementsList {
		elementsMap[element.Name] = element
	}

	//ini buat validasi tier, di spek blg ga boleh kek misal
	elementsMap = utils.ValidateRecipeTiers(elementsMap)

	//buat graf
	elementGraph := graph.NewElementGraph(elementsMap)

	return elementsMap, elementGraph, nil
}
