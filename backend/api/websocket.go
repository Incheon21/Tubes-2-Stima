package api

import (
	"backend/internal/algorithm"
	"backend/model"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type AnimationStep struct {
	StepIndex   int             `json:"stepIndex"`
	TotalSteps  int             `json:"totalSteps"`
	Node        json.RawMessage `json:"node,omitempty"`
	Link        json.RawMessage `json:"link,omitempty"`
	IsBaseNode  bool            `json:"isBaseNode"`
	IsCompleted bool            `json:"isCompleted"`
	Type        string          `json:"type"`
}

func (h *Handler) HandleAnimationWebSocket(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 4 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	targetElement := urlParts[len(urlParts)-1]
	targetElement, _ = url.QueryUnescape(targetElement)

	algorithmType := r.URL.Query().Get("algorithm")
	if algorithmType == "" {
		algorithmType = "bfs" 
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true 
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ERROR: Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("DEBUG: WebSocket connection established for %s using %s algorithm", targetElement, algorithmType)

	conn.WriteJSON(map[string]interface{}{
		"type":      "metadata",
		"algorithm": algorithmType,
		"element":   targetElement,
	})

	var animationSteps []map[string]interface{}
	var visitedCount int

	switch algorithmType {
	case "bfs":
		animationSteps, visitedCount = h.generateBFSAnimationSteps(targetElement)
	case "dfs":
		animationSteps, visitedCount = h.generateDFSAnimationSteps(targetElement)
	case "bidirectional":
		animationSteps, visitedCount = h.generateBidirectionalAnimationSteps(targetElement)
	default:
		log.Printf("WARNING: Unknown algorithm %s, falling back to BFS", algorithmType)
		animationSteps, visitedCount = h.generateBFSAnimationSteps(targetElement)
	}

	conn.WriteJSON(map[string]interface{}{
		"type":       "steps",
		"totalSteps": len(animationSteps),
	})

	for i, step := range animationSteps {
		step["stepIndex"] = i + 1
		step["totalSteps"] = len(animationSteps)

		err := conn.WriteJSON(step)
		if err != nil {
			log.Printf("ERROR: Failed to send animation step: %v", err)
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	conn.WriteJSON(map[string]interface{}{
		"type":         "complete",
		"nodesVisited": visitedCount,
	})

	log.Printf("DEBUG: Animation complete for %s, sent %d steps", targetElement, len(animationSteps))
}

func (h *Handler) generateBFSAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	paths, visitedCount := algorithm.BFS(h.elements, targetElement, 1, true)

	return h.convertBFSToAnimationSteps(paths), visitedCount
}

func (h *Handler) generateDFSAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	paths, visitedCount := algorithm.DFS(h.elements, targetElement, 1, true)

	return h.convertDFSToAnimationSteps(paths), visitedCount
}

func (h *Handler) generateBidirectionalAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	paths, visitedCount := algorithm.BidirectionalBFS(h.elements, targetElement, 1, true)

	return h.convertBidirectionalToAnimationSteps(paths), visitedCount
}

func (h *Handler) convertBFSToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	nodesByLevel := make(map[int][]model.Node)
	maxLevel := 0

	for _, node := range path {
		isBase := false
		for _, base := range baseElements {
			if node.Element == base {
				isBase = true
				break
			}
		}

		if isBase {
			nodesByLevel[0] = append(nodesByLevel[0], node)
		}
	}

	for level := 0; level < len(path); level++ {
		if _, exists := nodesByLevel[level]; !exists {
			continue
		}

		for _, currentNode := range nodesByLevel[level] {
			for _, node := range path {
				if node.Ingredients != nil && containsElement(node.Ingredients, currentNode.Element) {
					alreadyAssigned := false
					for l := 0; l <= level; l++ {
						for _, assignedNode := range nodesByLevel[l] {
							if assignedNode.Element == node.Element {
								alreadyAssigned = true
								break
							}
						}
						if alreadyAssigned {
							break
						}
					}

					if !alreadyAssigned {
						nodesByLevel[level+1] = append(nodesByLevel[level+1], node)
						if level+1 > maxLevel {
							maxLevel = level + 1
						}
					}
				}
			}
		}
	}

	for level := 0; level <= maxLevel; level++ {
		for _, node := range nodesByLevel[level] {
			isBase := false
			for _, base := range baseElements {
				if node.Element == base {
					isBase = true
					break
				}
			}

			steps = append(steps, map[string]interface{}{
				"type": "node",
				"node": map[string]interface{}{
					"name":      node.Element,
					"imagePath": node.ImagePath,
				},
				"isBaseNode":  isBase,
				"isCompleted": false,
			})
		}
	}

	for i := 0; i < len(path)-1; i++ {
		for j := i + 1; j < len(path); j++ {
			if j == i+1 || (path[j].Ingredients != nil && containsElement(path[j].Ingredients, path[i].Element)) {
				steps = append(steps, map[string]interface{}{
					"type": "link",
					"link": map[string]interface{}{
						"source": path[i].Element,
						"target": path[j].Element,
					},
					"isBaseNode":  false,
					"isCompleted": false,
				})
			}
		}
	}

	return steps
}

func (h *Handler) convertDFSToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	for i := 0; i < len(path); i++ {
		isBase := false
		for _, base := range baseElements {
			if path[i].Element == base {
				isBase = true
				break
			}
		}

		steps = append(steps, map[string]interface{}{
			"type": "node",
			"node": map[string]interface{}{
				"name":      path[i].Element,
				"imagePath": path[i].ImagePath,
			},
			"isBaseNode":  isBase,
			"isCompleted": false,
		})

		if i < len(path)-1 && path[i+1].Ingredients != nil &&
			containsElement(path[i+1].Ingredients, path[i].Element) {
			steps = append(steps, map[string]interface{}{
				"type": "link",
				"link": map[string]interface{}{
					"source": path[i].Element,
					"target": path[i+1].Element,
				},
				"isBaseNode":  false,
				"isCompleted": false,
			})
		}
	}

	for i := 0; i < len(path)-1; i++ {
		for j := i + 1; j < len(path); j++ {
			if j == i+1 {
				continue
			}

			if path[j].Ingredients != nil && containsElement(path[j].Ingredients, path[i].Element) {
				steps = append(steps, map[string]interface{}{
					"type": "link",
					"link": map[string]interface{}{
						"source": path[i].Element,
						"target": path[j].Element,
					},
					"isBaseNode":  false,
					"isCompleted": false,
				})
			}
		}
	}

	return steps
}

func (h *Handler) convertBidirectionalToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	targetElement := ""
	if len(path) > 0 {
		targetElement = path[len(path)-1].Element
	}

	for i := len(path) - 1; i >= 0; i-- {
		if path[i].Element == targetElement {
			steps = append(steps, map[string]interface{}{
				"type": "node",
				"node": map[string]interface{}{
					"name":      path[i].Element,
					"imagePath": path[i].ImagePath,
				},
				"isBaseNode":  false,
				"isCompleted": false,
			})
			break
		}
	}

	for _, node := range path {
		isBase := false
		for _, base := range baseElements {
			if node.Element == base {
				isBase = true
				break
			}
		}

		if isBase {
			steps = append(steps, map[string]interface{}{
				"type": "node",
				"node": map[string]interface{}{
					"name":      node.Element,
					"imagePath": node.ImagePath,
				},
				"isBaseNode":  true,
				"isCompleted": false,
			})
		}
	}

	addedNodes := make(map[string]bool)
	for _, step := range steps {
		if step["type"] == "node" {
			nodeName := step["node"].(map[string]interface{})["name"].(string)
			addedNodes[nodeName] = true
		}
	}
	frontIndex := 0
	backIndex := len(path) - 1
	addFromFront := true

	for len(addedNodes) < len(path) {
		var node model.Node

		if addFromFront {
			for frontIndex < len(path) {
				if !addedNodes[path[frontIndex].Element] {
					node = path[frontIndex]
					addedNodes[node.Element] = true
					frontIndex++
					break
				}
				frontIndex++
			}
		} else {
			for backIndex >= 0 {
				if !addedNodes[path[backIndex].Element] {
					node = path[backIndex]
					addedNodes[node.Element] = true
					backIndex--
					break
				}
				backIndex--
			}
		}

		addFromFront = !addFromFront
		if node.Element != "" {
			isBase := false
			for _, base := range baseElements {
				if node.Element == base {
					isBase = true
					break
				}
			}

			steps = append(steps, map[string]interface{}{
				"type": "node",
				"node": map[string]interface{}{
					"name":      node.Element,
					"imagePath": node.ImagePath,
				},
				"isBaseNode":  isBase,
				"isCompleted": false,
			})
		}
	}

	for i := 0; i < len(path)-1; i++ {
		for j := i + 1; j < len(path); j++ {
			if j == i+1 || (path[j].Ingredients != nil && containsElement(path[j].Ingredients, path[i].Element)) {
				steps = append(steps, map[string]interface{}{
					"type": "link",
					"link": map[string]interface{}{
						"source": path[i].Element,
						"target": path[j].Element,
					},
					"isBaseNode":  false,
					"isCompleted": false,
				})
			}
		}
	}

	return steps
}

func containsElement(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
