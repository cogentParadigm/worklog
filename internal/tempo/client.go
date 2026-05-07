package tempo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL   string
	token     string
	accountID string
	client    *http.Client
}

func NewClient(baseURL, token, accountID string) *Client {
	if baseURL == "" {
		baseURL = "https://api.tempo.io/core/3"
	}
	return &Client{
		baseURL:   baseURL,
		token:     token,
		accountID: accountID,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

type Worklog struct {
	IssueKey         string
	TimeSpentSeconds int
	StartDate        string
	StartTime        string
	Description      string
}

func (c *Client) CreateWorklog(wl Worklog) error {
	url := c.baseURL + "/worklogs"
	payload := map[string]interface{}{
		"issueKey":         wl.IssueKey,
		"timeSpentSeconds": wl.TimeSpentSeconds,
		"startDate":        wl.StartDate,
		"startTime":        wl.StartTime,
		"description":      wl.Description,
		"authorAccountId":  c.accountID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var errBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
		return fmt.Errorf("tempo API %s: %v", resp.Status, errBody)
	}
	return fmt.Errorf("tempo API %s", resp.Status)
}
