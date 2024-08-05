package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

type HackerNewsData struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error load .env file")
	}

	hackerNewsAPI := "https://hacker-news.firebaseio.com/v0/topstories.json"
	response, err := http.Get(hackerNewsAPI)
	if err != nil {
		log.Fatalf("failed to get hackerNews stories:%v", err)
	}
	defer response.Body.Close()

	var storyID []int
	if err := json.NewDecoder(response.Body).Decode(&storyID); err != nil {
		log.Fatalf("failed to decode response: %v", err)
	}

	storyIDs := storyID[0]
	storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyID)
	storyResponse, err := http.Get(storyURL)
	if err != nil {
		log.Fatalf("Failed to fetch story: %v", err)
	}
	defer storyResponse.Body.Close()

	var story HackerNewsData
	if err := json.NewDecoder(storyResponse.Body).Decode(&story); err != nil {
		log.Fatalf("failed to decode story: %v", err)
	}

}
