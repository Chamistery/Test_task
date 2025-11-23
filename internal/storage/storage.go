package storage

import "github.com/Chamistery/Test_task/internal/models"

type Storage interface {
	CreateTeam(team *models.Team) error
	GetTeam(teamName string) (*models.Team, error)
	TeamExists(teamName string) (bool, error)

	UpsertUser(user *models.TeamMember, teamName string) error
	GetUser(userID string) (*models.User, error)
	SetUserIsActive(userID string, isActive bool) error
	GetUserTeam(userID string) (string, error)

	CreatePullRequest(pr *models.PullRequest) error
	GetPullRequest(prID string) (*models.PullRequest, error)
	PRExists(prID string) (bool, error)
	MergePullRequest(prID string) (*models.PullRequest, error)
	GetPRsByReviewer(userID string) ([]models.PullRequestShort, error)

	IsReviewerAssigned(prID string, userID string) (bool, error)
	ReassignReviewer(prID string, oldUserID string, newUserID string) error

	GetActiveCandidates(teamName string, excludeIDs []string) ([]string, error)

	GetStatistics() (*models.Statistics, error)

	BulkDeactivateTeamMembers(teamName string) (int, int, error)

	Close() error
}
