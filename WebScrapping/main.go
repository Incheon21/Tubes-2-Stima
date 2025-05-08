package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ElementRecipe represents a single combination of ingredients
type ElementRecipe struct {
	Ingredients []string `json:"ingredients"`
}

// Element represents a full element with its name, tier, image and recipes
type Element struct {
	Name       string
	Tier       int
	ImageURL   string
	LocalImage string
	Recipes    []ElementRecipe
	// Add a visited flag to prevent infinite recursion
	visited   bool
	computing bool
}

// Cleans image URLs but preserves the full path structure
func cleanImageUrl(imageURL string) string {
	// Only remove query parameters
	if strings.Contains(imageURL, "?") {
		imageURL = strings.Split(imageURL, "?")[0]
	}

	// Debug output
	fmt.Printf("Cleaned URL: %s\n", imageURL)

	return imageURL
}

// downloadImage downloads an image from a URL and saves it to a local file
func downloadImage(client *http.Client, imageURL, resultName, localImagePath string, debug bool) (string, error) {
	// Create request with proper headers
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Parse the URL to get the host for the Origin header
	parsedURL, err := url.Parse(imageURL)
	var origin string
	if err == nil {
		origin = parsedURL.Scheme + "://" + parsedURL.Host
	} else {
		origin = "https://static.wikia.nocookie.net"
	}

	// Add headers that mimic a web browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/png,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", "https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	// Try to download the image with retries
	var imgResp *http.Response
	var downloadErr error

	// Try up to 3 times with increasing backoff
	for attempt := 1; attempt <= 3; attempt++ {
		imgResp, downloadErr = client.Do(req)

		if downloadErr == nil && imgResp.StatusCode == http.StatusOK {
			break
		}

		if imgResp != nil {
			imgResp.Body.Close()
		}

		// Exponential backoff
		waitTime := time.Duration(attempt) * time.Second
		if debug {
			fmt.Printf("Retry %d for %s (waiting %v)\n", attempt, resultName, waitTime)
		}
		time.Sleep(waitTime)
	}

	if downloadErr != nil {
		return "", fmt.Errorf("download error after retries: %v", downloadErr)
	}

	if imgResp == nil || imgResp.StatusCode != http.StatusOK {
		statusCode := 0
		if imgResp != nil {
			statusCode = imgResp.StatusCode
			imgResp.Body.Close()
		}
		return "", fmt.Errorf("bad response status: %d", statusCode)
	}

	defer imgResp.Body.Close()

	// Debug headers
	if debug {
		fmt.Println("Response headers:")
		for key, val := range imgResp.Header {
			fmt.Printf("  %s: %s\n", key, val)
		}
		fmt.Printf("Content-Type: %s\n", imgResp.Header.Get("Content-Type"))
	}

	// Create the file
	imgFile, err := os.Create(localImagePath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer imgFile.Close()

	// Copy image data to file
	bytesWritten, err := io.Copy(imgFile, imgResp.Body)
	if err != nil {
		return "", fmt.Errorf("error writing to file: %v", err)
	}

	if debug {
		fmt.Printf("Successfully saved %d bytes to %s\n", bytesWritten, localImagePath)
	}

	// Check if file is empty or too small
	if bytesWritten < 100 {
		return "", fmt.Errorf("downloaded file is too small (%d bytes)", bytesWritten)
	}

	return localImagePath, nil
}

func main() {
	fmt.Println("Starting Little Alchemy 2 scraper...")

	// Create an HTTP client with a timeout and that follows redirects
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects, but copy the original headers to the redirect
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			for key, val := range via[0].Header {
				if _, ok := req.Header[key]; !ok {
					req.Header[key] = val
				}
			}
			return nil
		},
	}

	// Fetch the webpage
	resp, err := client.Get("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)")
	if err != nil {
		log.Fatalf("Failed to fetch webpage: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Bad response status: %s", resp.Status)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	// Create images directory if it doesn't exist
	imagesDir := "images"
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		err = os.Mkdir(imagesDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create images directory: %v", err)
		}
	}

	// Map to store elements by name
	elements := make(map[string]*Element)

	// Debug: Print total table count
	fmt.Printf("Found %d tables total\n", doc.Find("table").Length())
	fmt.Printf("Found %d list-tables\n", doc.Find("table.list-table").Length())

	// Process each table row in list-table class (skip the header row)
	doc.Find("table.list-table tr:not(:first-child)").Each(func(rowIdx int, row *goquery.Selection) {
		cells := row.Find("td")

		// Debug row structure
		if rowIdx < 3 {
			fmt.Printf("Row %d has %d cells\n", rowIdx, cells.Length())
		}

		// Skip if not enough cells
		if cells.Length() < 2 {
			return
		}

		// Get result element from first cell (td -> span -> a)
		firstCell := cells.First()
		var resultName string
		var imageURL string

		// First try to get the image
		firstCell.Find("span.icon-hover span[typeof='mw:File'] img").Each(func(_ int, img *goquery.Selection) {
			// Try data-src first, which often contains the original URL
			if src, exists := img.Attr("data-src"); exists && src != "" {
				imageURL = cleanImageUrl(src)
			} else if src, exists := img.Attr("src"); exists && src != "" {
				imageURL = cleanImageUrl(src)
			}

			// Debug image URLs for first few rows
			if rowIdx < 3 {
				fmt.Printf("Found image URL: %s\n", imageURL)
			}
		})

		// Try to get element name from a link inside span
		firstCell.Find("span.icon-hover a").Each(func(_ int, a *goquery.Selection) {
			if title, exists := a.Attr("title"); exists && title != "" {
				resultName = title
			} else {
				resultName = strings.TrimSpace(a.Text())
			}
		})

		// If we couldn't find the result via the above method, try direct a tags
		if resultName == "" {
			firstCell.Find("a").Each(func(_ int, a *goquery.Selection) {
				if title, exists := a.Attr("title"); exists && title != "" {
					resultName = title
				} else {
					resultName = strings.TrimSpace(a.Text())
				}
			})
		}

		// As a last resort, use cell text
		if resultName == "" {
			resultName = strings.TrimSpace(firstCell.Text())
		}

		// Skip if no result name found
		if resultName == "" {
			return
		}

		// Download the image if we have a URL
		var localImagePath string
		if imageURL != "" {
			// Debug URLs
			if rowIdx < 3 {
				fmt.Printf("Using Image URL: %s\n", imageURL)
			}

			// Extract file extension
			extension := ".png" // Default to .png now
			if strings.Contains(imageURL, ".jpg") || strings.Contains(imageURL, ".jpeg") {
				extension = ".jpg"
			}

			// Generate filename from URL using MD5 hash
			hasher := md5.New()
			hasher.Write([]byte(imageURL))
			hash := hex.EncodeToString(hasher.Sum(nil))

			// Generate local filename - clean up any invalid characters
			safeResultName := strings.Map(func(r rune) rune {
				if strings.ContainsRune(`<>:"/\|?*`, r) {
					return '_'
				}
				return r
			}, resultName)

			filename := fmt.Sprintf("%s_%s%s", strings.ReplaceAll(safeResultName, " ", "_"), hash[:8], extension)
			localImagePath = filepath.Join(imagesDir, filename)

			// Download the image if it doesn't exist
			if _, err := os.Stat(localImagePath); os.IsNotExist(err) {
				fmt.Printf("Downloading image for %s from %s\n", resultName, imageURL)

				// Use our improved download function
				downloadedPath, err := downloadImage(client, imageURL, resultName, localImagePath, rowIdx < 3)
				if err != nil {
					fmt.Printf("Error downloading image for %s: %v\n", resultName, err)
					localImagePath = ""
				} else {
					localImagePath = downloadedPath
				}
			}
		}

		if _, exists := elements[resultName]; !exists {
			elements[resultName] = &Element{
				Name:       resultName,
				Tier:       1,
				ImageURL:   imageURL,
				LocalImage: localImagePath,
				Recipes:    []ElementRecipe{},
			}
		} else if imageURL != "" && elements[resultName].ImageURL == "" {
			elements[resultName].ImageURL = imageURL
			elements[resultName].LocalImage = localImagePath
		}

		secondCell := cells.Eq(1)

		if rowIdx < 3 {
			html, _ := secondCell.Html()
			fmt.Printf("Recipe cell HTML for %s: %s\n", resultName, html)
		}

		recipeCount := 0
		secondCell.Find("ul li").Each(func(liIdx int, li *goquery.Selection) {
			var ingredients []string

			li.Find("a").Each(func(_ int, a *goquery.Selection) {
				if title, exists := a.Attr("title"); exists && title != "" {
					ingredients = append(ingredients, title)
				} else {
					text := strings.TrimSpace(a.Text())
					if text != "" && text != "+" {
						ingredients = append(ingredients, text)
					}
				}
			})

			if len(ingredients) == 0 {
				li.Find("span.icon-hover a").Each(func(_ int, a *goquery.Selection) {
					if title, exists := a.Attr("title"); exists && title != "" {
						ingredients = append(ingredients, title)
					} else {
						text := strings.TrimSpace(a.Text())
						if text != "" && text != "+" {
							ingredients = append(ingredients, text)
						}
					}
				})
			}

			if len(ingredients) == 0 {
				liText := strings.TrimSpace(li.Text())
				parts := strings.Split(liText, "+")
				for _, part := range parts {
					cleaned := strings.TrimSpace(part)
					if cleaned != "" {
						ingredients = append(ingredients, cleaned)
					}
				}
			}

			if len(ingredients) >= 2 {
				elements[resultName].Recipes = append(elements[resultName].Recipes, ElementRecipe{
					Ingredients: ingredients,
				})
				recipeCount++

				if rowIdx < 3 && liIdx < 3 {
					fmt.Printf("Found recipe: %s = %s\n",
						strings.Join(ingredients, " + "), resultName)
				}

				for _, ingredient := range ingredients {
					if _, exists := elements[ingredient]; !exists {
						elements[ingredient] = &Element{
							Name:    ingredient,
							Tier:    1,
							Recipes: []ElementRecipe{},
						}
					}
				}
			}
		})

		if rowIdx < 3 {
			fmt.Printf("Found %d recipes for %s\n", recipeCount, resultName)
		}
	})

	fmt.Println("Calculating element tiers...")

	for name, elem := range elements {
		if len(elem.Recipes) == 0 {
			elem.Tier = 1
			elem.visited = true
			fmt.Printf("Base element: %s (Tier 1)\n", name)
		}
	}

	changed := true
	maxIterations := 100
	iteration := 0

	for changed && iteration < maxIterations {
		changed = false
		iteration++
		fmt.Printf("Tier calculation iteration %d\n", iteration)

		for name, elem := range elements {
			if elem.visited {
				continue
			}

			canCalculate := false
			maxTier := 0

			for _, recipe := range elem.Recipes {
				allIngredientsVisited := true
				recipeTier := 0

				for _, ingredient := range recipe.Ingredients {
					ingredientElem, exists := elements[ingredient]
					if !exists || !ingredientElem.visited {
						allIngredientsVisited = false
						break
					}

					if ingredientElem.Tier > recipeTier {
						recipeTier = ingredientElem.Tier
					}
				}

				if allIngredientsVisited && recipeTier > maxTier {
					maxTier = recipeTier
					canCalculate = true
				}
			}

			if canCalculate {
				elem.Tier = maxTier + 1
				elem.visited = true
				changed = true
				fmt.Printf("Calculated: %s (Tier %d)\n", name, elem.Tier)
			}
		}
	}

	for name, elem := range elements {
		if !elem.visited {
			fmt.Printf("Warning: Element %s has potential circular dependency, setting to Tier 1\n", name)
			elem.Tier = 1
		}
	}

	elementArray := make([]map[string]interface{}, 0)

	for _, elem := range elements {
		recipesArray := make([]map[string]interface{}, 0)
		for _, recipe := range elem.Recipes {
			recipeMap := map[string]interface{}{
				"ingredients": recipe.Ingredients,
			}
			recipesArray = append(recipesArray, recipeMap)
		}

		elementMap := map[string]interface{}{
			"name":    elem.Name,
			"tier":    elem.Tier,
			"recipes": recipesArray,
		}

		if elem.ImageURL != "" {
			elementMap["image"] = elem.ImageURL
		}

		if elem.LocalImage != "" {
			elementMap["localImage"] = elem.LocalImage
		}

		elementArray = append(elementArray, elementMap)
	}

	jsonData, err := json.MarshalIndent(elementArray, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal to JSON: %v", err)
	}

	err = os.WriteFile("elements.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	fmt.Printf("Scraped %d elements and saved to elements.json\n", len(elementArray))
	fmt.Printf("Downloaded images are saved in the '%s' directory\n", imagesDir)

	if len(elementArray) > 0 {
		fmt.Println("\nSample element format:")
		sampleJSON, _ := json.MarshalIndent(elementArray[0], "", "  ")
		fmt.Println(string(sampleJSON))
	}
}
