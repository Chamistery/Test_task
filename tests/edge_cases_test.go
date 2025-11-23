package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Chamistery/Test_task/internal/models"
)

func TestNoActiveReviewers(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "no-active-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "na-u1", Username: "Author", IsActive: true},
			{UserID: "na-u2", Username: "Inactive1", IsActive: false},
			{UserID: "na-u3", Username: "Inactive2", IsActive: false},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-no-active-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test No Active",
		AuthorID:        "na-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	var result map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&result)

	pr := result["pr"]

	if len(pr.AssignedReviewers) != 0 {
		t.Errorf("Expected 0 reviewers when all are inactive, got %d", len(pr.AssignedReviewers))
	}
}

func TestOneActiveReviewer(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "one-active-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "oa-u1", Username: "Author", IsActive: true},
			{UserID: "oa-u2", Username: "Reviewer", IsActive: true},
			{UserID: "oa-u3", Username: "Inactive", IsActive: false},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-one-active-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test One Active",
		AuthorID:        "oa-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	var result map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&result)

	pr := result["pr"]

	if len(pr.AssignedReviewers) != 1 {
		t.Errorf("Expected 1 reviewer, got %d", len(pr.AssignedReviewers))
	}

	if len(pr.AssignedReviewers) > 0 && pr.AssignedReviewers[0] != "oa-u2" {
		t.Errorf("Expected reviewer oa-u2, got %s", pr.AssignedReviewers[0])
	}
}

func TestReassignAfterMerge(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "reassign-merge-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "rm-u1", Username: "Author", IsActive: true},
			{UserID: "rm-u2", Username: "Rev1", IsActive: true},
			{UserID: "rm-u3", Username: "Rev2", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-reassign-merge-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test Reassign After Merge",
		AuthorID:        "rm-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	var prResult map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&prResult)
	resp.Body.Close()

	mergeReq := models.MergePRRequest{PullRequestID: prReq.PullRequestID}
	body, _ = json.Marshal(mergeReq)
	resp, _ = http.Post(server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	pr := prResult["pr"]
	if len(pr.AssignedReviewers) == 0 {
		t.Fatal("No reviewers assigned")
	}

	reassignReq := models.ReassignRequest{
		PullRequestID: prReq.PullRequestID,
		OldUserID:     pr.AssignedReviewers[0],
	}
	body, _ = json.Marshal(reassignReq)
	resp, _ = http.Post(server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d", resp.StatusCode)
	}

	var errResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errResult)

	if errObj, ok := errResult["error"].(map[string]interface{}); ok {
		if code, ok := errObj["code"].(string); ok && code != "PR_MERGED" {
			t.Errorf("Expected error code PR_MERGED, got %s", code)
		}
	}
}

func TestReassignNonAssignedReviewer(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "reassign-non-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "rn-u1", Username: "Author", IsActive: true},
			{UserID: "rn-u2", Username: "Rev1", IsActive: true},
			{UserID: "rn-u3", Username: "Rev2", IsActive: true},
			{UserID: "rn-u4", Username: "NotAssigned", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-reassign-non-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test Reassign Non-Assigned",
		AuthorID:        "rn-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	var prResult map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&prResult)
	resp.Body.Close()

	pr := prResult["pr"]

	var nonAssignedUser string
	for _, userID := range []string{"rn-u2", "rn-u3", "rn-u4"} {
		isAssigned := false
		for _, revID := range pr.AssignedReviewers {
			if revID == userID {
				isAssigned = true
				break
			}
		}
		if !isAssigned {
			nonAssignedUser = userID
			break
		}
	}

	if nonAssignedUser == "" {
		t.Skip("All users were assigned as reviewers")
	}

	reassignReq := models.ReassignRequest{
		PullRequestID: prReq.PullRequestID,
		OldUserID:     nonAssignedUser,
	}
	body, _ = json.Marshal(reassignReq)
	resp, _ = http.Post(server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d", resp.StatusCode)
	}

	var errResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errResult)

	if errObj, ok := errResult["error"].(map[string]interface{}); ok {
		if code, ok := errObj["code"].(string); ok && code != "NOT_ASSIGNED" {
			t.Errorf("Expected error code NOT_ASSIGNED, got %s", code)
		}
	}
}

func TestInactiveUsersNotAssigned(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "inactive-check-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "ic-u1", Username: "Author", IsActive: true},
			{UserID: "ic-u2", Username: "ActiveRev1", IsActive: true},
			{UserID: "ic-u3", Username: "ActiveRev2", IsActive: true},
			{UserID: "ic-u4", Username: "Inactive1", IsActive: false},
			{UserID: "ic-u5", Username: "Inactive2", IsActive: false},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-inactive-check-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test Inactive Not Assigned",
		AuthorID:        "ic-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	var result map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&result)

	pr := result["pr"]

	if len(pr.AssignedReviewers) != 2 {
		t.Errorf("Expected 2 active reviewers, got %d", len(pr.AssignedReviewers))
	}

	for _, revID := range pr.AssignedReviewers {
		if revID == "ic-u4" || revID == "ic-u5" {
			t.Errorf("Inactive user %s should not be assigned as reviewer", revID)
		}
	}
}

func TestReassignFromSameTeam(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	teamName := "reassign-same-" + fmt.Sprint(time.Now().Unix())
	team := models.Team{
		TeamName: teamName,
		Members: []models.TeamMember{
			{UserID: "rs-u1", Username: "Author", IsActive: true},
			{UserID: "rs-u2", Username: "Rev1", IsActive: true},
			{UserID: "rs-u3", Username: "Rev2", IsActive: true},
			{UserID: "rs-u4", Username: "Rev3", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := models.CreatePRRequest{
		PullRequestID:   "pr-reassign-same-" + fmt.Sprint(time.Now().Unix()),
		PullRequestName: "Test Reassign Same Team",
		AuthorID:        "rs-u1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	var prResult map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&prResult)
	resp.Body.Close()

	pr := prResult["pr"]
	if len(pr.AssignedReviewers) == 0 {
		t.Fatal("No reviewers assigned")
	}

	oldReviewer := pr.AssignedReviewers[0]

	reassignReq := models.ReassignRequest{
		PullRequestID: prReq.PullRequestID,
		OldUserID:     oldReviewer,
	}
	body, _ = json.Marshal(reassignReq)
	resp, _ = http.Post(server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var reassignResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&reassignResult)

	if replacedBy, ok := reassignResult["replaced_by"].(string); ok {
		validReviewers := []string{"rs-u2", "rs-u3", "rs-u4"}
		isValid := false
		for _, valid := range validReviewers {
			if replacedBy == valid && replacedBy != "rs-u1" {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("New reviewer %s is not from the same team", replacedBy)
		}
	}
}
