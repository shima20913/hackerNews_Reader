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
	"time"

	"github.com/joho/godotenv"
)

const (
	sentStoriesFile  = "sent_stories.json"
	lastSentTimeFile = "last_sent_time.json"
	checkInterval    = 15 * time.Minute // ストーリーのチェック間隔
	timeToPost       = 30 * time.Minute // 投稿頻度
)

type HackerNewsData struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"` //hackerNewsストーリーを格納
}

type LastSentTime struct {
	LastSent time.Time `json:"last_sent"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error load .env file") //.envの読み込み
	}

	for {
		if err := checkNewStory(); err != nil {
			log.Printf("error: %v", err)
		}
		time.Sleep(checkInterval)
	}

}

func loadSentStories() (map[int]bool, error) {
	file, err := os.Open(sentStoriesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int]bool), nil // ファイルが存在しない場合新規作成
		}
		return nil, err
	}
	defer file.Close()

	var sentStories map[int]bool
	if err := json.NewDecoder(file).Decode(&sentStories); err != nil {
		return nil, err
	}
	return sentStories, nil
}

func saveSentStories(sentStories map[int]bool) error {
	file, err := os.Create(sentStoriesFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(sentStories)
}

func checkLastSentTime() (*LastSentTime, error) {
	file, err := os.Open(lastSentTimeFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &LastSentTime{LastSent: time.Now().Add(-timeToPost)}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lastSentTime LastSentTime
	if err := json.NewDecoder(file).Decode(&lastSentTime); err != nil {
		return nil, err
	}
	return &lastSentTime, nil

}

func saveLastSentTime(lastSentTime *LastSentTime) error {
	file, err := os.Create(lastSentTimeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(lastSentTime)
}

func checkNewStory() error {
	lastSentTime, err := checkLastSentTime()
	if err != nil {
		return fmt.Errorf("failed to load last sent time: %v", err)
	}

	if time.Since(lastSentTime.LastSent) < timeToPost {

		return nil
	}

	hackerNewsAPI := "https://hacker-news.firebaseio.com/v0/topstories.json"
	response, err := http.Get(hackerNewsAPI)
	if err != nil {
		log.Fatalf("failed to get hackerNews stories:%v", err) // Hacker NewsストーリーのIDを取得
	}
	defer response.Body.Close()

	var storyID []int
	if err := json.NewDecoder(response.Body).Decode(&storyID); err != nil {
		log.Fatalf("failed to decode response: %v", err) //ストーリーIDのリストをデコード
	}

	sentStories, err := loadSentStories()
	if err != nil {
		return fmt.Errorf("failed to load sent stories: %v", err)
	}

	//送信済み記事を除外
	for _, id := range storyID {
		if _, exists := sentStories[id]; exists {
			continue
		}

		storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
		storyResponse, err := http.Get(storyURL)
		if err != nil {
			log.Fatalf("Failed to fetch story: %v", err)
		}
		defer storyResponse.Body.Close() //ストーリーの詳細を取得

		var story HackerNewsData
		if err := json.NewDecoder(storyResponse.Body).Decode(&story); err != nil {
			log.Fatalf("failed to decode story: %v", err)
		}

		translateTitle, err := translateText(story.Title)
		if err != nil {
			log.Fatalf("Failed to translate title: %v", err) //記事タイトルの翻訳
		}

		translatedContent, err := translateText(story.Text)
		if err != nil {
			log.Fatalf("failed to translate content: %v", err) //記事内容の翻訳
		}

		message := fmt.Sprintf("**%s**\n%s\n\n%s", translateTitle, story.URL, translatedContent)

		if err := sendToDiscord(message); err != nil {
			log.Fatalf("Failed to post message to Discord: %v", err) //discordに送信するメッセージ
		}

		sentStories[id] = true
		if err := saveSentStories(sentStories); err != nil {
			return fmt.Errorf("failed to save sent stories: %v", err) //送信済みストーリーIDをファイルに格納
		}

		lastSentTime.LastSent = time.Now()
		if err := saveLastSentTime(lastSentTime); err != nil {
			return fmt.Errorf("failed to save last sent time: %v", err)
		}
	}
	return nil

}

// google翻訳apiを用いて記事を翻訳
func translateText(text string) (string, error) {
	apiKey := os.Getenv("GOOGLE_TRANSLATE_API")
	if apiKey == "" {
		return "", fmt.Errorf("can not set GOOGLE_TRANSLATE_API")
	}
	url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2?key=%s", apiKey)
	body := fmt.Sprintf(`{"q": "%s", "source": "en", "target": "ja", "format": "text"}`, escapeJSON(text))
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Response status: %s", resp.Status)
		log.Printf("Response body: %s", string(respBody))
		return "", fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	log.Printf("API response: %v", result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response structure: %v", result)
	}

	translations, ok := data["translations"].([]interface{})
	if !ok || len(translations) == 0 {
		return "", fmt.Errorf("no translations found in response")
	}

	translation, ok := translations[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected translation structure: %v", translations[0])
	}

	translateText, ok := translation["translatedText"].(string)
	if !ok {
		return "", fmt.Errorf("translatedText not found or is not a string")
	}
	return translateText, nil
}

func escapeJSON(text string) string {
	return strings.ReplaceAll(text, `"`, `\"`)
}

// Discordにメッセージを送信
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}
	return nil

}
