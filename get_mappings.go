package main

import (
	"context"
	"encoding/json"
	"fmt"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func getAllMappings(esClient *elasticsearch.Client) (map[string]interface{}, error) {
	ctx := context.Background()

	// Build the Elasticsearch get mapping request
	req := esapi.IndicesGetMappingRequest{
		Index: []string{"*"}, // Retrieve mappings for all indices
	}

	// Execute the request
	res, err := req.Do(ctx, esClient)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to retrieve mappings: %s", res.Status())
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
