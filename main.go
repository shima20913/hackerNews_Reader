package main

import (
	"log"

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
}
