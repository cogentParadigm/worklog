package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

type timeTracking struct {
	RemainingEstimateSeconds int `json:"remainingEstimateSeconds"`
}

type issueTimeTrackingResponse struct {
	Fields struct {
		TimeTracking *timeTracking `json:"timetracking"`
	} `json:"fields"`
}

type SearchIssueResult struct {
	Key     string
	Summary string
}

type issuePickerIssue struct {
	Key         string `json:"key"`
	Summary     string `json:"summary"`
	SummaryText string `json:"summaryText"`
}

type issuePickerSection struct {
	ID     string             `json:"id"`
	Label  string             `json:"label"`
	Issues []issuePickerIssue `json:"issues"`
}

type issuePickerResponse struct {
	Sections []issuePickerSection `json:"sections"`
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

func (c *Client) SearchIssues(query string) ([]SearchIssueResult, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("jira base URL not configured")
	}
	url := fmt.Sprintf("%s/rest/api/3/issue/picker?query=%s", c.baseURL, url.QueryEscape(query))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return nil, fmt.Errorf("jira API %s: %v", resp.Status, errBody)
		}
		return nil, fmt.Errorf("jira API %s", resp.Status)
	}

	var body issuePickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode jira response: %w", err)
	}

	var results []SearchIssueResult
	for _, section := range body.Sections {
		for _, issue := range section.Issues {
			summary := issue.SummaryText
			if summary == "" {
				summary = issue.Summary
			}
			results = append(results, SearchIssueResult{
				Key:     issue.Key,
				Summary: summary,
			})
		}
	}
	return results, nil
}

func (c *Client) GetRemainingEstimate(issueKey string) (int, error) {
	if c.baseURL == "" {
		return 0, fmt.Errorf("jira base URL not configured")
	}
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=timetracking", c.baseURL, issueKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			return 0, fmt.Errorf("jira API %s: %v", resp.Status, errBody)
		}
		return 0, fmt.Errorf("jira API %s", resp.Status)
	}

	var body issueTimeTrackingResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode jira response: %w", err)
	}
	if body.Fields.TimeTracking == nil {
		return 0, nil
	}
	return body.Fields.TimeTracking.RemainingEstimateSeconds, nil
}
