package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gin-gonic/gin"
)

func queryAllDocuments(esClient *elasticsearch.Client) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// Build the Elasticsearch search query
	query := `
    {
      "query": {
        "match_all": {}
      }
    }`

	// Execute the query
	req := esapi.SearchRequest{
		Index: []string{projects_mapping_index},
		Body:  strings.NewReader(query),
	}

	res, err := req.Do(ctx, esClient)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to query documents: %s", res.Status())
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Extract the documents from the response
	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	documents := make([]map[string]interface{}, len(hits))
	for i, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		documents[i] = source.(map[string]interface{})
	}

	return documents, nil
}

func getProjectsCreatedByUser(c *gin.Context, esClient *elasticsearch.Client) {
	userID := c.Query("user_id")
	documents, err := queryProjectsCreatedByUser(esClient, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query documents"})
		return
	}

	response := formatResponse("Retrieved projects created by user", documents)
	c.JSON(http.StatusOK, response)
}

func getProjectsWithHashtags(c *gin.Context, esClient *elasticsearch.Client) {
	hashtag := c.Query("hashtag")
	documents, err := queryProjectsWithHashtags(esClient, hashtag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query documents"})
		return
	}

	response := formatResponse("Retrieved projects with hashtag", documents)
	c.JSON(http.StatusOK, response)
}

func fuzzySearchProjects(c *gin.Context, esClient *elasticsearch.Client) {
	slug := c.Query("slug")
	description := c.Query("description")
	documents, err := fuzzySearchProjectsInDescriptionAndSlug(esClient, slug, description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query documents"})
		return
	}

	response := formatResponse("Retrieved projects with fuzzy search", documents)
	c.JSON(http.StatusOK, response)
}

func executeSearchRequest(esClient *elasticsearch.Client, ctx context.Context, req esapi.SearchRequest) ([]map[string]interface{}, error) {
	res, err := req.Do(ctx, esClient)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search request failed: %s", res.String())
	}
	fmt.Println("fuzzy reached")

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		fmt.Println("decoder failed: %v", err)
		return nil, err
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	documents := make([]map[string]interface{}, len(hits))
	for i, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		documents[i] = source.(map[string]interface{})
	}

	return documents, nil
}

func queryProjectsCreatedByUser(esClient *elasticsearch.Client, userID string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	query := `
	{
		"query": {
			"nested": {
				"path": "users",
				"query": {
					"match": { "users.id": %s }
				}
			}
		}
	}`

	req := esapi.SearchRequest{
		Index: []string{projects_mapping_index},
		Body:  strings.NewReader(fmt.Sprintf(query, userID)),
	}

	return executeSearchRequest(esClient, ctx, req)
}

func queryProjectsWithHashtags(esClient *elasticsearch.Client, hashtag string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	query := `
	{
		"query": {
			"match": {
				"hashtags": "%s"
			}
		}
	}`

	req := esapi.SearchRequest{
		Index: []string{projects_mapping_index},
		Body:  strings.NewReader(fmt.Sprintf(query, hashtag)),
	}

	return executeSearchRequest(esClient, ctx, req)
}

func fuzzySearchProjectsInDescriptionAndSlug(esClient *elasticsearch.Client, slug string, description string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	query := ""

	if slug != "" && description != "" {
		query = fmt.Sprintf(`
			{
				"query": {
					"bool": {
						"must": [
							{ 
								"fuzzy": {
									"slug": {
										"value": "%s",
										"fuzziness": "AUTO"
									}
								}
							},
							{ 
								"fuzzy": {
									"description": {
										"value": "%s",
										"fuzziness": "AUTO"
									}
								}
							}
						]
					}
				}
			}`, slug, description)
	} else if slug != "" {
		query = fmt.Sprintf(`
			{
				"query": {
					"fuzzy": {
						"slug": {
							"value": "%s",
							"fuzziness": "AUTO"
						}
					}
				}
			}`, slug)
	} else if description != "" {
		query = fmt.Sprintf(`
			{
				"query": {
					"fuzzy": {
						"description": {
							"value": "%s",
							"fuzziness": "AUTO"
						}
					}
				}
			}`, description)
	}

	fmt.Println(query)
	req := esapi.SearchRequest{
		Index: []string{projects_mapping_index},
		Body:  strings.NewReader(query),
	}

	return executeSearchRequest(esClient, ctx, req)
}

func formatResponse(message string, data []map[string]interface{}) gin.H {
	return gin.H{
		"message": message,
		"data":    data,
	}
}
