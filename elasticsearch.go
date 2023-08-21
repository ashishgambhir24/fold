package main

import (
	"context"
	"log"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Function to create missing elasticsearch mappings
func createElasticSearchMappings(esClient *elasticsearch.Client) error {
	ctx := context.Background()

	// Define the Elasticsearch index mappings for projects
	projectsMapping := `
	{
        "mappings": {
            "properties": {
                "id": { "type": "integer" },
                "name": { "type": "text" },
                "slug": { "type": "text" },
                "description": { "type": "text" },
                "created_at": { "type": "date" },
                "users": {
                    "type": "nested",
                    "properties": {
                        "id": { "type": "integer" },
                        "name": { "type": "text" },
						"created_at": { "type": "date" }
                    }
                },
                "hashtags": { "type": "keyword" }
            }
        }
    }`

	// Check if index already exists
	exists, err := indexExists(ctx, esClient, projects_mapping_index)
	if err != nil {
		log.Printf("Error checking index existence: %v", err)
		return err
	}

	// Create index with mappings in Elasticsearch if it doesn't exist
	if !exists {
		err = createIndexWithMapping(ctx, esClient, projects_mapping_index, projectsMapping)
		if err != nil {
			log.Printf("Error creating index %s: %v", projects_mapping_index, err)
		} else {
			log.Printf("Index %s created.", projects_mapping_index)
		}
	} else {
		log.Printf("Index %s already exists.", projects_mapping_index)
	}

	return nil
}

func indexExists(ctx context.Context, esClient *elasticsearch.Client, indexName string) (bool, error) {
	res, err := esapi.IndicesExistsRequest{Index: []string{indexName}}.Do(ctx, esClient)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, nil
	}
	return res.StatusCode == 200, nil
}

func createIndexWithMapping(ctx context.Context, esClient *elasticsearch.Client, indexName, mapping string) error {
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}
	res, err := req.Do(ctx, esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return err
	}
	return nil
}

func clearElasticsearchIndices(esClient *elasticsearch.Client) {
	ctx := context.Background()

	indicesToClear := []string{
		projects_mapping_index,
	}

	for _, indexName := range indicesToClear {
		req := esapi.IndicesDeleteRequest{
			Index: []string{indexName},
		}
		_, err := req.Do(ctx, esClient)
		if err != nil {
			log.Printf("Error deleting index %s: %v", indexName, err)
		} else {
			log.Printf("Index %s deleted.", indexName)
		}
	}
}
