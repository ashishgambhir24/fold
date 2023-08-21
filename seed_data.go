package main

import (
	"database/sql"
	"time"
)

// Function to add seed data to db
func seedData(pgDB *sql.DB) error {
	tableSeedFunctions := []func(*sql.DB){
		seedUsers,
		seedProjects,
		seedHashtags,
		seedUsersProjects,
		seedProjectHashtags,
	}

	for _, seedFunc := range tableSeedFunctions {
		seedFunc(pgDB)
	}

	// Commit the transaction
	_, _ = pgDB.Exec("COMMIT")

	// Introduce a delay before querying for notifications
	time.Sleep(1 * time.Millisecond)

	return nil
}

func seedUsers(pgDB *sql.DB) {
	seedData := []struct {
		ID        int
		Name      string
		CreatedAt time.Time
	}{
		{ID: 1, Name: "hawking", CreatedAt: time.Now()},
		{ID: 2, Name: "newton", CreatedAt: time.Now()},
		{ID: 3, Name: "tesla", CreatedAt: time.Now()},
		{ID: 4, Name: "curie", CreatedAt: time.Now()},
		{ID: 5, Name: "musk", CreatedAt: time.Now()},
	}

	for _, data := range seedData {
		_, _ = pgDB.Exec("INSERT INTO users (id, name, created_at) VALUES ($1, $2, $3)", data.ID, data.Name, data.CreatedAt)
	}
}

func seedProjects(pgDB *sql.DB) {
	seedData := []struct {
		ID          int
		Name        string
		Slug        string
		Description string
		CreatedAt   time.Time
	}{
		{ID: 1, Name: "Project alpha", Slug: "project-alpha", Description: "Project Alpha's small description", CreatedAt: time.Now()},
		{ID: 2, Name: "Project beta", Slug: "project-beta", Description: "Project Beta's small description", CreatedAt: time.Now()},
		{ID: 3, Name: "Project gamma", Slug: "project-gamma", Description: "Project Gamma's small description", CreatedAt: time.Now()},
		{ID: 4, Name: "Project delta", Slug: "project-delta", Description: "Project Delta's small description", CreatedAt: time.Now()},
	}

	for _, data := range seedData {
		_, _ = pgDB.Exec("INSERT INTO projects (id, name, slug, description, created_at) VALUES ($1, $2, $3, $4, $5)",
			data.ID, data.Name, data.Slug, data.Description, data.CreatedAt)
	}
}

func seedHashtags(pgDB *sql.DB) {
	seedData := []struct {
		ID        int
		Name      string
		CreatedAt time.Time
	}{
		{ID: 1, Name: "world_cup", CreatedAt: time.Now()},
		{ID: 2, Name: "ipl", CreatedAt: time.Now()},
		{ID: 3, Name: "champions_league", CreatedAt: time.Now()},
		{ID: 4, Name: "premier_league", CreatedAt: time.Now()},
		{ID: 5, Name: "laliga", CreatedAt: time.Now()},
	}

	for _, data := range seedData {
		_, _ = pgDB.Exec("INSERT INTO hashtags (id, name, created_at) VALUES ($1, $2, $3)", data.ID, data.Name, data.CreatedAt)
	}
}

func seedUsersProjects(pgDB *sql.DB) {
	seedData := []struct {
		UserID    int
		ProjectID int
	}{
		{UserID: 1, ProjectID: 1},
		{UserID: 1, ProjectID: 2},
		{UserID: 2, ProjectID: 2},
		{UserID: 2, ProjectID: 4},
		{UserID: 3, ProjectID: 3},
		{UserID: 4, ProjectID: 4},
		{UserID: 4, ProjectID: 2},
		{UserID: 4, ProjectID: 1},
		{UserID: 5, ProjectID: 4},
	}

	for _, data := range seedData {
		_, _ = pgDB.Exec("INSERT INTO users_projects (user_id, project_id) VALUES ($1, $2)", data.UserID, data.ProjectID)
	}
}

func seedProjectHashtags(pgDB *sql.DB) {
	seedData := []struct {
		HashtagID int
		ProjectID int
	}{
		{HashtagID: 1, ProjectID: 3},
		{HashtagID: 1, ProjectID: 1},
		{HashtagID: 2, ProjectID: 4},
		{HashtagID: 3, ProjectID: 1},
		{HashtagID: 3, ProjectID: 3},
		{HashtagID: 3, ProjectID: 2},
		{HashtagID: 4, ProjectID: 1},
		{HashtagID: 5, ProjectID: 2},
		{HashtagID: 5, ProjectID: 3},
	}

	for _, data := range seedData {
		_, _ = pgDB.Exec("INSERT INTO project_hashtags (project_id, hashtag_id) VALUES ($1, $2)", data.ProjectID, data.HashtagID)
	}
}
