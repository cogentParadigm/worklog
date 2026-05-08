package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL  string
	username string
	token    string
	client   *http.Client
}

func NewClient(baseURL, username, token string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		token:    token,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

type issueResponse struct {
	ID string `json:"id"`
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.username+":"+c.token))
}

func (c *Client) GetIssueID(issueKey string) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("jira base URL not configured")
	}
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=id", c.baseURL, issueKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return "", fmt.Errorf("jira API %s: %v", resp.Status, errBody)
		}
		return "", fmt.Errorf("jira API %s", resp.Status)
	}

	var body issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode jira response: %w", err)
	}
	return body.ID, nil
}
