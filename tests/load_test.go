package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Chamistery/Test_task/internal/models"
)

func BenchmarkPRCreation(b *testing.B) {
	server, _, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	teamName := "bench-team"
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "bench-u1", Username: "User1", IsActive: true},
			{UserID: "bench-u2", Username: "User2", IsActive: true},
			{UserID: "bench-u3", Username: "User3", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		prReq := models.CreatePRRequest{
			PullRequestID:   fmt.Sprintf("bench-pr-%d", i),
			PullRequestName: fmt.Sprintf("PR %d", i),
			AuthorID:        "bench-u1",
		}
		body, _ := json.Marshal(prReq)
		resp, _ := http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
		resp.Body.Close()
	}
}

func TestConcurrentPRCreation(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "concurrent-team"
	team := models.Team{
		TeamName: teamName,
		Members:  []models.TeamMember{},
	}

	for i := 0; i < 10; i++ {
		team.Members = append(team.Members, models.TeamMember{
			UserID:   fmt.Sprintf("concurrent-u%d", i),
			Username: fmt.Sprintf("User%d", i),
			IsActive: true,
		})
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	var wg sync.WaitGroup
	requests := 50
	start := time.Now()

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			prReq := models.CreatePRRequest{
				PullRequestID:   fmt.Sprintf("concurrent-pr-%d", id),
				PullRequestName: fmt.Sprintf("Concurrent PR %d", id),
				AuthorID:        fmt.Sprintf("concurrent-u%d", id%10),
			}
			body, _ := json.Marshal(prReq)
			resp, err := http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
			if err != nil {
				t.Errorf("Failed to create PR: %v", err)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Created %d PRs in %v", requests, duration)
	t.Logf("Average time per request: %v", duration/time.Duration(requests))

	if duration > 10*time.Second {
		t.Errorf("Performance test failed: took %v for %d requests", duration, requests)
	}
}

func TestResponseTimeUnderLoad(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "load-team"
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "load-u1", Username: "LoadUser", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	iterations := 100
	var totalDuration time.Duration
	slowRequests := 0

	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp, err := http.Get(fmt.Sprintf("%s/team/get?team_name=%s", server.URL, teamName))
		duration := time.Since(start)
		totalDuration += duration

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()

		if duration > 300*time.Millisecond {
			slowRequests++
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average response time: %v over %d requests", avgDuration, iterations)
	t.Logf("Slow requests (>300ms): %d/%d", slowRequests, iterations)

	if avgDuration > 300*time.Millisecond {
		t.Errorf("Average response time too slow: %v (expected < 300ms)", avgDuration)
	}
}

func TestBulkDeactivatePerformance(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "bulk-team"
	team := models.Team{
		TeamName: teamName,
		Members:  []models.TeamMember{},
	}

	for i := 0; i < 20; i++ {
		team.Members = append(team.Members, models.TeamMember{
			UserID:   fmt.Sprintf("bulk-u%d", i),
			Username: fmt.Sprintf("BulkUser%d", i),
			IsActive: true,
		})
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	for i := 0; i < 50; i++ {
		prReq := models.CreatePRRequest{
			PullRequestID:   fmt.Sprintf("bulk-pr-%d", i),
			PullRequestName: fmt.Sprintf("Bulk PR %d", i),
			AuthorID:        fmt.Sprintf("bulk-u%d", i%20),
		}
		body, _ := json.Marshal(prReq)
		resp, _ := http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
		resp.Body.Close()
	}

	req := models.BulkDeactivateRequest{TeamName: teamName}
	body, _ = json.Marshal(req)

	start := time.Now()
	resp, err := http.Post(server.URL+"/team/deactivate", "application/json", bytes.NewBuffer(body))
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Bulk deactivate failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.BulkDeactivateResponse
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("Deactivated %d users and reassigned %d PRs in %v",
		result.DeactivatedUsers, result.ReassignedPRs, duration)

	if duration > 100*time.Millisecond {
		t.Logf("WARNING: Bulk deactivate took %v (target: <100ms)", duration)
	}
}
