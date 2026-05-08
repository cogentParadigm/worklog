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
		baseURL = "https://api.tempo.io/4"
	}
	return &Client{
		baseURL:   baseURL,
		token:     token,
		accountID: accountID,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

type Worklog struct {
	IssueId                    string
	TimeSpentSeconds           int
	StartDate                  string
	StartTime                  string
	Description                string
	RemainingEstimateSeconds   *int
}

func (c *Client) CreateWorklog(wl Worklog) error {
	url := c.baseURL + "/worklogs"
	payload := map[string]interface{}{
		"issueId":          wl.IssueId,
		"timeSpentSeconds": wl.TimeSpentSeconds,
		"startDate":        wl.StartDate,
		"startTime":        wl.StartTime,
		"description":      wl.Description,
		"authorAccountId":  c.accountID,
	}
	if wl.RemainingEstimateSeconds != nil {
		payload["remainingEstimateSeconds"] = *wl.RemainingEstimateSeconds
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
