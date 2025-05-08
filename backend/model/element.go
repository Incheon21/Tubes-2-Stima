package model

//kombinasi ingredient untuk elemen
type ElementRecipe struct {
	Ingredients []string `json:"ingredients"`
}

//struktur untuk elemen
type Element struct {
	Name       string          `json:"name"`
	Image      string          `json:"image,omitempty"`
	LocalImage string          `json:"localImage,omitempty"`
	Recipes    []ElementRecipe `json:"recipes,omitempty"`
	Tier       int             `json:"tier"`
}

//searchconfig untuk pencarian elemen
type SearchConfig struct {
	TargetElement string `json:"targetElement"`
	Algorithm     string `json:"algorithm"` // "bfs", "dfs", ato "bidirectional"
	MaxResults    int    `json:"maxResults"`
	SinglePath    bool   `json:"singlePath"`
}

//hasil dari pencarian elemen
type SearchResult struct {
	Paths        [][]Node `json:"paths"`
	TimeElapsed  int64    `json:"timeElapsed"` //ms
	NodesVisited int      `json:"nodesVisited"`
}

//struct node untuk menyimpan elemen dan ingredientnya
type Node struct {
	Element     string   `json:"element"`
	Ingredients []string `json:"ingredients,omitempty"`
}
