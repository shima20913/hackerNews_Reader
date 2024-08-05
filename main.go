package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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
	storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyIDs)
	storyResponse, err := http.Get(storyURL)
	if err != nil {
		log.Fatalf("Failed to fetch story: %v", err)
	}
	defer storyResponse.Body.Close()

	var story HackerNewsData
	if err := json.NewDecoder(storyResponse.Body).Decode(&story); err != nil {
		log.Fatalf("failed to decode story: %v", err)
	}

	translateTitle, err := translateText(story.Title)
	if err != nil {
		log.Fatalf("Failed to translate title: %v", err)
	}

	message := fmt.Sprintf("**%s**\n%s", translateTitle, story.URL)
	if err := sendToDiscord(message); err != nil {
		log.Fatalf("Failed to post message to Discord: %v", err)
	}

}

func translateText(text string) (string, err) {
	apiKey := os.Getenv("GOOGLE_TRANSLATE_API")
	url := "https://translation.googleapis.com/language/translate/v2"

	body := fmt.Sprintf(`{"q": "%s", "target": "ja"}`, text)
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	translateText := result["data"].(map[string]interface{})["translations"].([]interface{})[0].(map[string]interface{})["translatedText"].(string)
	return translateText, nil
}

func sendToDiscord(message string) error {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	dataToSend := map[string]string{"content": message}
	jsonData, err := json.Marshal(dataToSend)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData)) // DiscordにPOSTリクエストを送信
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}
	return nil

}
