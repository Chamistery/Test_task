package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chamistery/Test_task/internal/handlers"
	"github.com/Chamistery/Test_task/internal/models"
	"github.com/Chamistery/Test_task/internal/storage"
)

func setupTestServer(t *testing.T) (*httptest.Server, *handlers.Handlers, func()) {
	dbConfig := storage.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "reviewer_service",
	}

	store, err := storage.NewPostgresStorage(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	h := handlers.NewHandlers(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/team/add", h.HandleTeamAdd)
	mux.HandleFunc("/team/get", h.HandleTeamGet)
	mux.HandleFunc("/users/setIsActive", h.HandleUserSetIsActive)
	mux.HandleFunc("/pullRequest/create", h.HandlePullRequestCreate)
	mux.HandleFunc("/pullRequest/merge", h.HandlePullRequestMerge)
	mux.HandleFunc("/pullRequest/reassign", h.HandlePullRequestReassign)
	mux.HandleFunc("/users/getReview", h.HandleUsersGetReview)
	mux.HandleFunc("/statistics", h.HandleStatistics)
	mux.HandleFunc("/team/deactivate", h.HandleTeamDeactivate)

	server := httptest.NewServer(mux)

	cleanup := func() {
		server.Close()
		store.Close()
	}

	return server, h, cleanup
}

func TestTeamCreationAndRetrieval(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	team := models.Team{
		TeamName: "test-team-" + fmt.Sprint(time.Now().Unix()),
		Members: []models.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, err := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	resp, err = http.Get(fmt.Sprintf("%s/team/get?team_name=%s", server.URL, team.TeamName))
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPRCreationAndReviewerAssignment(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "pr-test-team-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "pr-u1", Username: "Author", IsActive: true},
			{UserID: "pr-u2", Username: "Rev1", IsActive: true},
			{UserID: "pr-u3", Username: "Rev2", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-test-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test PR",
		AuthorID:        "pr-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	var result map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&result)

	pr := result["pr"]

	if len(pr.AssignedReviewers) == 0 {
		t.Error("Expected reviewers to be assigned")
	}

	if len(pr.AssignedReviewers) > 2 {
		t.Error("Expected max 2 reviewers")
	}

	for _, revID := range pr.AssignedReviewers {
		if revID == "pr-u1" {
			t.Error("Author should not be assigned as reviewer")
		}
	}
}

func TestMergeIdempotency(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "merge-test-team-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "merge-u1", Username: "Author", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-merge-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Merge Test",
		AuthorID:        "merge-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	var result map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	mergeReq := models.MergePRRequest{PullRequestID: prReq.PullRequestID}
	body, _ = json.Marshal(mergeReq)

	resp, _ = http.Post(server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("First merge failed with status %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, _ = http.Post(server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Second merge failed with status %d", resp.StatusCode)
	}

	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result["pr"].Status != "MERGED" {
		t.Errorf("Expected status MERGED, got %s", result["pr"].Status)
	}
}
