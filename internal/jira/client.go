package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type issueResponse struct {
	ID string `json:"id"`
}

func (c *Client) authHeader() string {
	if strings.Contains(c.token, ":") {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.token))
	}
	return "Bearer " + c.token
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
