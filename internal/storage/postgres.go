package storage

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Chamistery/Test_task/internal/models"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type PostgresStorage struct {
	db   *sql.DB
	rand *rand.Rand
}

func NewPostgresStorage(config DBConfig) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName)

	var db *sql.DB
	var err error

	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStorage{
		db:   db,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *PostgresStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS teams (
		team_name TEXT PRIMARY KEY
	);
	
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		team_name TEXT REFERENCES teams(team_name) ON DELETE CASCADE,
		is_active BOOLEAN DEFAULT true
	);
	
	CREATE INDEX IF NOT EXISTS idx_users_team ON users(team_name);
	CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);
	
	CREATE TABLE IF NOT EXISTS pull_requests (
		pull_request_id TEXT PRIMARY KEY,
		pull_request_name TEXT NOT NULL,
		author_id TEXT REFERENCES users(user_id),
		status TEXT DEFAULT 'OPEN',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		merged_at TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
	CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);
	
	CREATE TABLE IF NOT EXISTS pr_reviewers (
		pull_request_id TEXT REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
		user_id TEXT REFERENCES users(user_id),
		PRIMARY KEY (pull_request_id, user_id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_reviewers_user ON pr_reviewers(user_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

func (s *PostgresStorage) CreateTeam(team *models.Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		_, err := tx.Exec(`
			INSERT INTO users (user_id, username, team_name, is_active) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE 
			SET username = EXCLUDED.username, 
			    team_name = EXCLUDED.team_name,
			    is_active = EXCLUDED.is_active
		`, member.UserID, member.Username, team.TeamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStorage) GetTeam(teamName string) (*models.Team, error) {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT user_id, username, is_active 
		FROM users 
		WHERE team_name = $1
		ORDER BY user_id
	`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	team := &models.Team{
		TeamName: teamName,
		Members:  []models.TeamMember{},
	}

	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		team.Members = append(team.Members, member)
	}

	return team, nil
}

func (s *PostgresStorage) TeamExists(teamName string) (bool, error) {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	return exists, err
}

func (s *PostgresStorage) UpsertUser(user *models.TeamMember, teamName string) error {
	_, err := s.db.Exec(`
		INSERT INTO users (user_id, username, team_name, is_active) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE 
		SET username = EXCLUDED.username, 
		    team_name = EXCLUDED.team_name,
		    is_active = EXCLUDED.is_active
	`, user.UserID, user.Username, teamName, user.IsActive)
	return err
}

func (s *PostgresStorage) GetUser(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStorage) SetUserIsActive(userID string, isActive bool) error {
	result, err := s.db.Exec("UPDATE users SET is_active = $1 WHERE user_id = $2", isActive, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *PostgresStorage) GetUserTeam(userID string) (string, error) {
	var teamName string
	err := s.db.QueryRow("SELECT team_name FROM users WHERE user_id = $1", userID).Scan(&teamName)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return teamName, err
}

func (s *PostgresStorage) CreatePullRequest(pr *models.PullRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) 
		VALUES ($1, $2, $3, $4, $5)
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, "OPEN", now)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		_, err := tx.Exec(`
			INSERT INTO pr_reviewers (pull_request_id, user_id) 
			VALUES ($1, $2)
		`, pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	pr.CreatedAt = &now
	pr.Status = "OPEN"

	return tx.Commit()
}

func (s *PostgresStorage) GetPullRequest(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt time.Time
	var mergedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pr.CreatedAt = &createdAt
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := s.db.Query("SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pr.AssignedReviewers = []string{}
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, nil
}

func (s *PostgresStorage) PRExists(prID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", prID).Scan(&exists)
	return exists, err
}

func (s *PostgresStorage) MergePullRequest(prID string) (*models.PullRequest, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if status != "MERGED" {
		now := time.Now()
		_, err = tx.Exec(`
			UPDATE pull_requests 
			SET status = 'MERGED', merged_at = $1 
			WHERE pull_request_id = $2
		`, now, prID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetPullRequest(prID)
}

func (s *PostgresStorage) GetPRsByReviewer(userID string) ([]models.PullRequestShort, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_requests p
		JOIN pr_reviewers r ON p.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1
		ORDER BY p.pull_request_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prs := []models.PullRequestShort{}
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

func (s *PostgresStorage) IsReviewerAssigned(prID string, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM pr_reviewers 
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`, prID, userID).Scan(&exists)
	return exists, err
}

func (s *PostgresStorage) ReassignReviewer(prID string, oldUserID string, newUserID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2", prID, oldUserID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)", prID, newUserID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStorage) GetActiveCandidates(teamName string, excludeIDs []string) ([]string, error) {
	query := `
		SELECT user_id FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != ALL($2)
		ORDER BY user_id
	`

	rows, err := s.db.Query(query, teamName, excludeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := []string{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		candidates = append(candidates, userID)
	}

	return candidates, nil
}

func (s *PostgresStorage) GetStatistics() (*models.Statistics, error) {
	stats := &models.Statistics{
		ReviewerAssignments: make(map[string]int),
		PRsByAuthor:         make(map[string]int),
	}

	err := s.db.QueryRow("SELECT COUNT(*) FROM pull_requests").Scan(&stats.TotalPRs)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'").Scan(&stats.OpenPRs)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'").Scan(&stats.MergedPRs)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT u.username, COUNT(*) as count
		FROM pr_reviewers prr
		JOIN users u ON prr.user_id = u.user_id
		GROUP BY u.user_id, u.username
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		stats.ReviewerAssignments[name] = count
	}

	rows, err = s.db.Query(`
		SELECT u.username, COUNT(*) as count
		FROM pull_requests pr
		JOIN users u ON pr.author_id = u.user_id
		GROUP BY u.user_id, u.username
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		stats.PRsByAuthor[name] = count
	}

	var totalReviewers int
	err = s.db.QueryRow("SELECT COUNT(*) FROM pr_reviewers").Scan(&totalReviewers)
	if err != nil {
		return nil, err
	}

	if stats.TotalPRs > 0 {
		stats.AverageReviewersPerPR = float64(totalReviewers) / float64(stats.TotalPRs)
	}

	return stats, nil
}

func (s *PostgresStorage) BulkDeactivateTeamMembers(teamName string) (int, int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT user_id FROM users
		WHERE team_name = $1 AND is_active = true
	`, teamName)
	if err != nil {
		return 0, 0, err
	}

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return 0, 0, err
		}
		userIDs = append(userIDs, userID)
	}
	rows.Close()

	if len(userIDs) == 0 {
		return 0, 0, nil
	}

	_, err = tx.Exec(`UPDATE users SET is_active = false WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		return 0, 0, err
	}

	prRows, err := tx.Query(`
		SELECT DISTINCT pr.pull_request_id, pr.author_id
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE pr.status = 'OPEN' AND prr.user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return 0, 0, err
	}

	type prInfo struct {
		id       string
		authorID string
	}
	var prs []prInfo
	for prRows.Next() {
		var pr prInfo
		if err := prRows.Scan(&pr.id, &pr.authorID); err != nil {
			prRows.Close()
			return 0, 0, err
		}
		prs = append(prs, pr)
	}
	prRows.Close()

	reassignedCount := 0

	for _, pr := range prs {
		var currentReviewers []string
		reviewerRows, err := tx.Query("SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1", pr.id)
		if err != nil {
			return 0, 0, err
		}

		for reviewerRows.Next() {
			var revID string
			if err := reviewerRows.Scan(&revID); err != nil {
				reviewerRows.Close()
				return 0, 0, err
			}
			currentReviewers = append(currentReviewers, revID)
		}
		reviewerRows.Close()

		var toReplace []string
		for _, revID := range currentReviewers {
			for _, deactivatedID := range userIDs {
				if revID == deactivatedID {
					toReplace = append(toReplace, revID)
					break
				}
			}
		}

		if len(toReplace) == 0 {
			continue
		}

		var authorTeamName string
		err = tx.QueryRow("SELECT team_name FROM users WHERE user_id = $1", pr.authorID).Scan(&authorTeamName)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, err
		}
		if err == sql.ErrNoRows {
			continue
		}

		excludeIDs := append(currentReviewers, pr.authorID)

		candidates, err := s.getActiveCandidatesInTx(tx, authorTeamName, excludeIDs)
		if err != nil {
			return 0, 0, err
		}

		replaced := 0
		for _, oldRevID := range toReplace {
			if len(candidates) == 0 {
				break
			}

			newRevID := candidates[s.rand.Intn(len(candidates))]

			_, err = tx.Exec("DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2", pr.id, oldRevID)
			if err != nil {
				return 0, 0, err
			}

			_, err = tx.Exec("INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)", pr.id, newRevID)
			if err != nil {
				return 0, 0, err
			}

			for i, c := range candidates {
				if c == newRevID {
					candidates = append(candidates[:i], candidates[i+1:]...)
					break
				}
			}

			replaced++
		}

		if replaced > 0 {
			reassignedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return len(userIDs), reassignedCount, nil
}

func (s *PostgresStorage) getActiveCandidatesInTx(tx *sql.Tx, teamName string, excludeIDs []string) ([]string, error) {
	query := `
		SELECT user_id FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != ALL($2)
		ORDER BY user_id
	`

	rows, err := tx.Query(query, teamName, excludeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := []string{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		candidates = append(candidates, userID)
	}

	return candidates, nil
}
