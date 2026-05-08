package tempo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateWorklogSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/worklogs" {
			t.Errorf("expected path /worklogs, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["issueId"] != "PROJ-123" {
			t.Errorf("issueId: got %v, want PROJ-123", body["issueId"])
		}
		if body["authorAccountId"] != "account-123" {
			t.Errorf("authorAccountId: got %v, want account-123", body["authorAccountId"])
		}
		if body["startTime"] != "09:00:00" {
			t.Errorf("startTime: got %v, want 09:00:00", body["startTime"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", "account-123")
	wl := Worklog{
		IssueId:          "PROJ-123",
		TimeSpentSeconds: 3600,
		StartDate:        "2026-05-06",
		StartTime:        "09:00:00",
		Description:      "Test worklog",
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestCreateWorklogError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid token"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token", "account-123")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	expected := "invalid token"
	if !contains(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
}

func TestCreateWorklogNetworkError(t *testing.T) {
	// Use an invalid URL to force a network error
	client := NewClient("http://localhost:1", "token", "account")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient("", "token", "account")
	if client.baseURL != "https://api.tempo.io/4" {
		t.Errorf("default base URL: got %q, want %q", client.baseURL, "https://api.tempo.io/4")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCreateWorklogPlainTextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Forbidden")
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestCreateWorklogWithRemainingEstimate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["remainingEstimateSeconds"] != float64(7200) {
			t.Errorf("remainingEstimateSeconds: got %v, want 7200", body["remainingEstimateSeconds"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	rem := 7200
	wl := Worklog{
		IssueId:                  "PROJ-123",
		TimeSpentSeconds:         3600,
		StartDate:                "2026-05-06",
		StartTime:                "09:00:00",
		Description:              "Test worklog",
		RemainingEstimateSeconds: &rem,
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestCreateWorklogWithoutRemainingEstimate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["remainingEstimateSeconds"]; ok {
			t.Errorf("remainingEstimateSeconds should not be present, got %v", body["remainingEstimateSeconds"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	wl := Worklog{
		IssueId:          "PROJ-123",
		TimeSpentSeconds: 3600,
		StartDate:        "2026-05-06",
		StartTime:        "09:00:00",
		Description:      "Test worklog",
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestCreateWorklogWithAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		attrs, ok := body["attributes"].([]interface{})
		if !ok {
			t.Fatalf("expected attributes array, got %T", body["attributes"])
		}
		if len(attrs) != 2 {
			t.Fatalf("expected 2 attributes, got %d", len(attrs))
		}
		keys := make(map[string]string)
		for _, a := range attrs {
			m := a.(map[string]interface{})
			keys[m["key"].(string)] = m["value"].(string)
		}
		if keys["_WorkType_"] != "Development" {
			t.Errorf("_WorkType_: got %q, want Development", keys["_WorkType_"])
		}
		if keys["_Billable_"] != "Yes" {
			t.Errorf("_Billable_: got %q, want Yes", keys["_Billable_"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	wl := Worklog{
		IssueId:          "PROJ-123",
		TimeSpentSeconds: 3600,
		StartDate:        "2026-05-06",
		StartTime:        "09:00:00",
		Description:      "Test worklog",
		Attributes:       map[string]string{"_WorkType_": "Development", "_Billable_": "Yes"},
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestCreateWorklogNoAttributesWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["attributes"]; ok {
			t.Errorf("attributes should not be present when empty")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	wl := Worklog{
		IssueId:          "PROJ-123",
		TimeSpentSeconds: 3600,
		StartDate:        "2026-05-06",
		StartTime:        "09:00:00",
		Description:      "Test worklog",
		Attributes:       map[string]string{},
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestGetWorkAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/work-attributes" {
			t.Errorf("expected path /work-attributes, got %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"key":      "_WorkType_",
					"name":     "Work Type",
					"type":     "STATIC_LIST",
					"required": true,
					"values":   []string{"Development", "CodeReview", "Testing"},
					"names": map[string]string{
						"Development": "Development",
						"CodeReview":  "Code Review",
						"Testing":     "Testing",
					},
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	attrs, err := client.GetWorkAttributes()
	if err != nil {
		t.Fatalf("GetWorkAttributes: %v", err)
	}
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(attrs))
	}
	if attrs[0].Key != "_WorkType_" {
		t.Errorf("key: got %q, want _WorkType_", attrs[0].Key)
	}
	if !attrs[0].Required {
		t.Error("expected required=true")
	}
	if len(attrs[0].Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(attrs[0].Values))
	}
	if attrs[0].Names["CodeReview"] != "Code Review" {
		t.Errorf("names['CodeReview']: got %q, want 'Code Review'", attrs[0].Names["CodeReview"])
	}
}

func TestGetWorkAttributesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad token"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	_, err := client.GetWorkAttributes()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
