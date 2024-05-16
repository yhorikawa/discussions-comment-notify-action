package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Event struct {
	Action     string     `json:"action"`
	Comment    Comment    `json:"comment"`
	Discussion Discussion `json:"discussion"`
}

type Comment struct {
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	User      User   `json:"user"`
}

type User struct {
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type Discussion struct {
	Title     string   `json:"title"`
	HTMLURL   string   `json:"html_url"`
	CreatedAt string   `json:"created_at"`
	Category  Category `json:"category"`
}

type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SlackMentionMapList map[string]string

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func main() {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		fmt.Println("GITHUB_EVENT_PATH not found")
		os.Exit(1)
	}

	event, err := getEvent(eventPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	message := createSlackMessage(event)

	mentionMapPath := getGithubInput("SLACK_MENTION_MAP_PATH")
	if mentionMapPath == "" {
		fmt.Println("SLACK_MENTION_MAP_PATH not found")
		os.Exit(1)
	}

	mentionMap, err := readSlackMentionMap(mentionMapPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	message = convertToSlackMention(message, mentionMap)

	slackToken := getGithubInput("SLACK_API_TOKEN")
	if slackToken == "" {
		fmt.Println("SLACK_API_TOKEN not found")
		os.Exit(1)
	}

	slackChannel := getGithubInput("SLACK_CHANNEL")
	if slackChannel == "" {
		fmt.Println("SLACK_CHANNEL not found")
		os.Exit(1)
	}

	if err := SendMessage(slackToken, slackChannel, message); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Message sent successfully")
}

func getGithubInput(name string) string {
	return os.Getenv("INPUT_" + name)
}

func getEvent(eventPath string) (Event, error) {
	var event Event

	eventData, err := os.ReadFile(eventPath)
	if err != nil && !os.IsNotExist(err) {
		return Event{}, fmt.Errorf("could not read event file: %w", err)
	}

	if eventData != nil {
		if err := json.Unmarshal(eventData, &event); err != nil {
			return Event{}, fmt.Errorf("failed to unmarshal event payload: %w", err)
		}
	}

	return event, nil
}

func createSlackMessage(event Event) string {
	message := fmt.Sprintf("New comment by [%s](%s) on [%s](%s) in category %s: %s",
		event.Comment.User.Login,
		event.Comment.User.HTMLURL,
		event.Discussion.Title,
		event.Discussion.HTMLURL,
		event.Discussion.Category.Name,
		event.Comment.Body,
	)

	return message
}

func fetchGitHubFileContent(filePath string) ([]byte, error) {
	apiURL := "https://api.github.com/repos/" + os.Getenv("GITHUB_REPOSITORY") + "/contents/" + filePath

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "token "+getGithubInput("GITHUB_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching file: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON response: %v", err)
	}

	content, ok := result["content"].(string)
	if !ok {
		return nil, fmt.Errorf("error retrieving content from JSON response")
	}

	decodedContent, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("error decoding content: %v", err)
	}

	return decodedContent, nil
}

func readSlackMentionMap(filePath string) (SlackMentionMapList, error) {
	fileContent, err := fetchGitHubFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("error fetching file content: %v", err)
	}

	var mentionMap SlackMentionMapList
	if err := json.Unmarshal(fileContent, &mentionMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling mention map: %v", err)
	}

	return mentionMap, nil
}

func convertToSlackMention(text string, mentionMap SlackMentionMapList) string {
	for githubUsername, slackUserID := range mentionMap {
		text = strings.ReplaceAll(text, "@"+githubUsername, "<@"+slackUserID+">")
	}

	return text
}

func SendMessage(token, channel, message string) error {
	apiURL := "https://slack.com/api/chat.postMessage"

	payload := SlackMessage{
		Channel: channel,
		Text:    message,
	}

	messageBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error encoding message: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(messageBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error status not OK: %v", resp.Status)
	}

	return nil
}
