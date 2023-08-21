package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/lib/pq"
)

// Function to create missing postgres tables
func createPostgresTables(pgDB *sql.DB) error {
	createTablesStatements := []string{
		`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS projects (
			id SERIAL PRIMARY KEY,
			name VARCHAR,
			slug VARCHAR,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS hashtags (
			id SERIAL PRIMARY KEY,
			name VARCHAR,
			created_at TIMESTAMP DEFAULT NOW()
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS users_projects (
			project_id INTEGER REFERENCES projects(id),
			user_id INTEGER REFERENCES users(id),
			PRIMARY KEY (project_id, user_id)
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS project_hashtags (
			hashtag_id INTEGER REFERENCES hashtags(id),
			project_id INTEGER REFERENCES projects(id),
			PRIMARY KEY (hashtag_id, project_id)
		);
		`,
	}

	for _, statement := range createTablesStatements {
		_, err := pgDB.Exec(statement)
		if err != nil {
			return err
		}
	}
	return nil
}

func createTriggers(pgDB *sql.DB) error {
	triggersToCreate := []struct {
		Name      string
		Statement string
	}{
		{
			Name: "projects_data_changes",
			Statement: `
				CREATE OR REPLACE FUNCTION projects_data_changes()
				RETURNS TRIGGER AS $$
				BEGIN
					PERFORM pg_notify('data_changes', json_build_object(
						'trigger_name', 'projects_data_changes',
						'table_name', TG_TABLE_NAME,
						'entry', row_to_json(NEW)
					)::text);
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
		{
			Name: "project_hashtags_data_changes",
			Statement: `
				CREATE OR REPLACE FUNCTION project_hashtags_data_changes()
				RETURNS TRIGGER AS $$
				DECLARE
					hashtag_name TEXT;
				BEGIN
					SELECT h.name INTO hashtag_name FROM hashtags h WHERE h.id = NEW.hashtag_id;
					PERFORM pg_notify('data_changes', json_build_object(
						'trigger_name', 'project_hashtags_data_changes',
						'table_name', TG_TABLE_NAME,
						'entry', json_build_object(
							'project_id', NEW.project_id,
							'hashtag_name', hashtag_name
						)
					)::text);
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
		{
			Name: "users_projects_data_changes",
			Statement: `
				CREATE OR REPLACE FUNCTION users_projects_data_changes()
				RETURNS TRIGGER AS $$
				DECLARE
					user_info JSONB;
				BEGIN
					SELECT row_to_json(u) INTO user_info FROM users u WHERE u.id = NEW.user_id;
					PERFORM pg_notify('data_changes', json_build_object(
						'trigger_name', 'users_projects_data_changes',
						'table_name', TG_TABLE_NAME,
						'entry', json_build_object(
							'project_id', NEW.project_id,
							'user', user_info
						)
					)::text);
					RETURN NEW;
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
	}

	for _, trigger := range triggersToCreate {
		// Check if trigger already exists
		existsQuery := fmt.Sprintf("SELECT 1 FROM pg_trigger WHERE tgname = '%s'", trigger.Name)
		var exists bool
		err := pgDB.QueryRow(existsQuery).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if !exists {
			// Create the trigger if it doesn't exist
			_, err := pgDB.Exec(trigger.Statement)
			if err != nil {
				return err
			}

			// Attach the trigger to the appropriate table
			triggerAttachStatement := fmt.Sprintf("CREATE TRIGGER %s AFTER INSERT OR UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s();",
				trigger.Name, getTableNameFromTriggerName(trigger.Name), trigger.Name)
			_, err = pgDB.Exec(triggerAttachStatement)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getTableNameFromTriggerName(triggerName string) string {
	// Map trigger names to their corresponding table names
	triggerToTableMap := map[string]string{
		"projects_data_changes":         "projects",
		"project_hashtags_data_changes": "project_hashtags",
		"users_projects_data_changes":   "users_projects",
	}

	return triggerToTableMap[triggerName]
}

func startNotificationListener(pgDB *sql.DB, esClient *elasticsearch.Client, pgConnStr string) {
	// Set up PostgreSQL listener
	listener := pq.NewListener(pgConnStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("Listener error: %v", err)
		}
	})
	defer listener.Close()

	// Add PostgreSQL notifications to listener
	err := listener.Listen("data_changes")
	if err != nil {
		log.Fatalf("Error setting up LISTEN channel: %v", err)
	}

	// Start listening for notifications
	for {
		notification, ok := <-listener.Notify
		if !ok {
			return
		}

		// Process the notification payload
		var payload map[string]interface{}
		err := json.Unmarshal([]byte(notification.Extra), &payload)
		if err != nil {
			log.Printf("Error decoding notification payload: %v", err)
			continue
		}

		triggerName := payload["trigger_name"].(string)
		tableName := payload["table_name"].(string)
		entry := payload["entry"].(map[string]interface{})

		err = syncDataToElasticsearch(pgDB, esClient, triggerName, tableName, entry)
		if err != nil {
			fmt.Printf("Error syncing data to Elasticsearch: %v", err)
		}
	}
}

// Function to clear tables
func clearTables(pgDB *sql.DB) {
	tablesToDelete := []string{
		"users_projects",
		"project_hashtags",
		"users",
		"hashtags",
		"projects",
	}

	removeTriggers(pgDB)

	for _, table := range tablesToDelete {
		_, err := pgDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Error clearing table %s: %v", table, err)
		}
	}

	log.Println("Tables cleared successfully.")
}

func removeTriggers(pgDB *sql.DB) {
	triggersToRemove := []string{"project_hashtags_data_changes", "users_projects_data_changes", "projects_data_changes"}
	var triggerTableMap = map[string]string{
		"projects_data_changes":         "projects",
		"project_hashtags_data_changes": "project_hashtags",
		"users_projects_data_changes":   "users_projects",
	}

	for _, trigger := range triggersToRemove {
		_, err := pgDB.Exec(fmt.Sprintf("DROP TRIGGER %s ON %s", trigger, triggerTableMap[trigger]))
		if err != nil {
			log.Printf("Error removing trigger %s: %v", trigger, err)
		}
	}
}
