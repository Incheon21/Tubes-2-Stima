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

// AnimationStep represents a single step in the tree formation animation
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
	// Extract target element from URL path
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 4 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	targetElement := urlParts[len(urlParts)-1]
	targetElement, _ = url.QueryUnescape(targetElement)

	// Get algorithm type from query parameter
	algorithmType := r.URL.Query().Get("algorithm")
	if algorithmType == "" {
		algorithmType = "bfs" // Default to BFS if not specified
	}

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ERROR: Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("DEBUG: WebSocket connection established for %s using %s algorithm", targetElement, algorithmType)

	// Send metadata about the animation
	conn.WriteJSON(map[string]interface{}{
		"type":      "metadata",
		"algorithm": algorithmType,
		"element":   targetElement,
	})

	// Process algorithm based on type
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

	// Send information about total steps
	conn.WriteJSON(map[string]interface{}{
		"type":       "steps",
		"totalSteps": len(animationSteps),
	})

	// Send each animation step with a slight delay
	for i, step := range animationSteps {
		step["stepIndex"] = i + 1
		step["totalSteps"] = len(animationSteps)

		err := conn.WriteJSON(step)
		if err != nil {
			log.Printf("ERROR: Failed to send animation step: %v", err)
			break
		}

		// Add small delay between steps for visualization
		time.Sleep(50 * time.Millisecond)
	}

	// Send completion message
	conn.WriteJSON(map[string]interface{}{
		"type":         "complete",
		"nodesVisited": visitedCount,
	})

	log.Printf("DEBUG: Animation complete for %s, sent %d steps", targetElement, len(animationSteps))
}

// Generate animation steps for BFS
func (h *Handler) generateBFSAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	// Get recipe paths using BFS
	paths, visitedCount := algorithm.BFS(h.elements, targetElement, 1, true)

	// Convert paths to animation steps - breadth-first order (level by level)
	return h.convertBFSToAnimationSteps(paths), visitedCount
}

// Generate animation steps for DFS
func (h *Handler) generateDFSAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	// Get recipe paths using DFS
	paths, visitedCount := algorithm.DFS(h.elements, targetElement, 1, true)

	// Convert paths to animation steps - depth-first order
	return h.convertDFSToAnimationSteps(paths), visitedCount
}

// Generate animation steps for Bidirectional search
func (h *Handler) generateBidirectionalAnimationSteps(targetElement string) ([]map[string]interface{}, int) {
	// Get recipe paths using Bidirectional search
	paths, visitedCount := algorithm.BidirectionalBFS(h.elements, targetElement, 1, true)

	// Convert paths to animation steps - converging from both ends
	return h.convertBidirectionalToAnimationSteps(paths), visitedCount
}

// BFS animation focuses on level-by-level expansion
func (h *Handler) convertBFSToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// Group nodes by their depth from base elements
	nodesByLevel := make(map[int][]model.Node)
	maxLevel := 0

	// First, add base elements (level 0)
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

	// Then determine levels for other nodes based on recipe dependencies
	// This is a simplified way to simulate BFS levels
	for level := 0; level < len(path); level++ {
		if _, exists := nodesByLevel[level]; !exists {
			continue
		}

		// For each node at current level
		for _, currentNode := range nodesByLevel[level] {
			// Find nodes that use this as an ingredient
			for _, node := range path {
				if node.Ingredients != nil && containsElement(node.Ingredients, currentNode.Element) {
					// If this node isn't assigned a level yet, put it at next level
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

	// Add nodes in BFS order (level by level)
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

	// Then add links between nodes
	for i := 0; i < len(path)-1; i++ {
		for j := i + 1; j < len(path); j++ {
			// Check if these nodes should be connected based on recipe
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

// DFS animation focuses on complete paths one at a time
func (h *Handler) convertDFSToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	// For DFS, we want to show the deepest path first, then backtracking
	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// In a DFS path, elements are typically organized from base elements to target
	// We'll simulate DFS behavior by first showing a full path from base to target
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

		// If this node has a direct link to the next node, add it immediately
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

	// Then add any remaining links between nodes to complete the graph
	for i := 0; i < len(path)-1; i++ {
		for j := i + 1; j < len(path); j++ {
			// Skip links we already added
			if j == i+1 {
				continue
			}

			// Add links based on recipe dependencies
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

// Bidirectional animation shows exploration from both ends
func (h *Handler) convertBidirectionalToAnimationSteps(paths [][]model.Node) []map[string]interface{} {
	if len(paths) == 0 {
		return []map[string]interface{}{}
	}

	path := paths[0]
	steps := []map[string]interface{}{}
	baseElements := []string{"Water", "Fire", "Earth", "Air"}

	// For bidirectional search, we want to show nodes from both ends simultaneously
	// Find the target node (usually last) and base elements (usually first)
	targetElement := ""
	if len(path) > 0 {
		targetElement = path[len(path)-1].Element
	}

	// First add the target node
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

	// Then add base elements that exist in the path
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

	// Then alternately add more nodes from each end
	addedNodes := make(map[string]bool)
	for _, step := range steps {
		if step["type"] == "node" {
			nodeName := step["node"].(map[string]interface{})["name"].(string)
			addedNodes[nodeName] = true
		}
	}

	// Add remaining nodes in a bidirectional pattern
	frontIndex := 0
	backIndex := len(path) - 1
	addFromFront := true

	for len(addedNodes) < len(path) {
		var node model.Node

		if addFromFront {
			// Add from front (base elements toward middle)
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
			// Add from back (target toward middle)
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

		// Switch directions for next iteration
		addFromFront = !addFromFront

		// Add the node to steps if we found one
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

	// Then add links (mostly the same as before)
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

// Helper function to check if a string is in a slice
func containsElement(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
