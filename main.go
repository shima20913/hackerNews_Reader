package main

import (
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

}
