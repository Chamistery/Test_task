package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Chamistery/Test_task/internal/handlers"
	"github.com/Chamistery/Test_task/internal/storage"
)

func main() {
	dbConfig := storage.DBConfig{
		Host:     getEnv("DB_HOST", "postgres"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "reviewer_service"),
	}

	store, err := storage.NewPostgresStorage(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer store.Close()

	h := handlers.NewHandlers(store)

	http.HandleFunc("/team/add", h.HandleTeamAdd)
	http.HandleFunc("/team/get", h.HandleTeamGet)
	http.HandleFunc("/users/setIsActive", h.HandleUserSetIsActive)
	http.HandleFunc("/pullRequest/create", h.HandlePullRequestCreate)
	http.HandleFunc("/pullRequest/merge", h.HandlePullRequestMerge)
	http.HandleFunc("/pullRequest/reassign", h.HandlePullRequestReassign)
	http.HandleFunc("/users/getReview", h.HandleUsersGetReview)

	http.HandleFunc("/statistics", h.HandleStatistics)
	http.HandleFunc("/team/deactivate", h.HandleTeamDeactivate)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
