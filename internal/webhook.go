package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebHookPayload struct {
	Event     string  `json:"event"`
	Timestamp string  `json:"timestamp"`
	RepoName  string  `json:"repo_name,omitempty"`
	Matches   []Match `json:"matches"`
}

func SendWebhook(url string, matches []Match) (err error) {
	if url == "" {
		return nil
	}

	payload := WebHookPayload{
		Event:     "secret_detected",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Matches:   matches,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "git-good-cli")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err != nil {
			err = closeErr
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook server returned bad status code: %d", resp.StatusCode)
	}

	return nil
}
