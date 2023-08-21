package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
)

func syncDataToElasticsearch(pgDB *sql.DB, esClient *elasticsearch.Client, triggerName string, tableName string, entry map[string]interface{}) error {
	// Handle the change based on trigger and table
	switch triggerName {
	case "projects_data_changes":
		// Call a function to handle Elasticsearch indexing or updating
		entry["hashtags"] = []string{}
		entry["users"] = []interface{}{}
		syncDataToElasticsearchForProjects(esClient, entry)
	case "project_hashtags_data_changes":
		// Call a function to handle Elasticsearch indexing or updating
		err := syncDataToElasticsearchForProjectHashtags(esClient, entry)
		if err != nil {
			fmt.Println("hashtags update failed: %v", err)
		}
	case "users_projects_data_changes":
		// Call a function to handle Elasticsearch indexing or updating
		err := syncDataToElasticsearchForUsersProjects(esClient, entry)
		if err != nil {
			fmt.Println("hashtags update failed: %v", err)
		}
	default:
		fmt.Println("Unknown trigger:", triggerName)
	}
	return nil
}

// Function to sync projects table updates to elastic search
func syncDataToElasticsearchForProjects(esClient *elasticsearch.Client, project map[string]interface{}) error {
	ctx := context.Background()

	// Convert project data to JSON
	projectJSON, err := json.Marshal(project)
	if err != nil {
		return err
	}

	// Index the project data in Elasticsearch
	req := esapi.IndexRequest{
		Index:      projects_mapping_index,
		DocumentID: strconv.Itoa(int(project["id"].(float64))), // Assuming "id" is an integer
		Body:       strings.NewReader(string(projectJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.Status())
	}

	return nil
}

// Function to sync project_hashtags table updates to elastic search
func syncDataToElasticsearchForProjectHashtags(esClient *elasticsearch.Client, projectHashtag map[string]interface{}) error {
	ctx := context.Background()

	projectID := int(projectHashtag["project_id"].(float64))
	hashtagName := projectHashtag["hashtag_name"].(string)

	script := fmt.Sprintf(`{
        "script": {
            "source": "ctx._source.hashtags.add('%s')",
            "lang": "painless"
        }
    }`, hashtagName)

	req := esapi.UpdateRequest{
		Index:      projects_mapping_index,
		DocumentID: strconv.Itoa(projectID),
		Body:       strings.NewReader(script),
	}

	res, err := req.Do(ctx, esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("update document failed: %v", res.String())
	}

	return nil
}

// Function to sync users_projects table updates to elastic search
func syncDataToElasticsearchForUsersProjects(esClient *elasticsearch.Client, userProject map[string]interface{}) error {
	ctx := context.Background()

	projectID := int(userProject["project_id"].(float64))
	userInfo := userProject["user"].(map[string]interface{})
	userID := int(userInfo["id"].(float64))
	userName := userInfo["name"].(string)
	userCreatedAt := userInfo["created_at"].(string)

	script := fmt.Sprintf(`{
        "script": {
            "source": "ctx._source.users.add(params)",
            "lang": "painless",
            "params": {
                "name": "%s",
				"id": %d,
				"created_at": "%s"
            }
        }
    }`, userName, userID, userCreatedAt)

	req := esapi.UpdateRequest{
		Index:      projects_mapping_index, // Update with your actual index name
		DocumentID: strconv.Itoa(projectID),
		Body:       strings.NewReader(script),
	}

	res, err := req.Do(ctx, esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("update document failed: %v", res)
	}

	return nil
}
