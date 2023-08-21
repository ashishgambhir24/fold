package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
)

const projects_mapping_index = "projects_index"

var esClient *elasticsearch.Client

func main() {
	// Initialize PostgreSQL connection
	godotenv.Load(".env")
	pgConnStr := fmt.Sprintf("user=%s dbname=fold-finance sslmode=disable", os.Getenv("POSTGRES_USER"))

	pgDB, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Create postgres missing tables
	err = createPostgresTables(pgDB)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// Set up triggers for data changes
	err = createTriggers(pgDB)
	if err != nil {
		log.Fatalf("Error creating triggers: %v", err)
	}

	// Initialize Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200", // Update with your Elasticsearch URL
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	// Check whether Elasticsearch is accepting requests
	// _, err = esClient.Ping()
	// if err != nil {
	// 	fmt.Println("Elasticsearch ping failed: %v", err)
	// }

	// Create elasticsearch missing mappings
	err = createElasticSearchMappings(esClient)
	if err != nil {
		log.Fatalf("Error creating mappings: %v", err)
	}

	// // Get all mappings
	// mappings, err := getAllMappings(esClient)
	// if err != nil {
	// 	log.Fatalf("Error retrieving mappings: %v", err)
	// }

	// // Print the mappings
	// formattedMappings, _ := json.MarshalIndent(mappings, "", "  ")
	// fmt.Printf("Mappings:\n%s\n", formattedMappings)

	// Start the listener in a separate goroutine
	go startNotificationListener(pgDB, esClient, pgConnStr)
	time.Sleep(1 * time.Second)

	err = seedData(pgDB)
	if err != nil {
		log.Fatalf("Error adding seed data: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Additional endpoint to verify elasticsearch sync
	router.GET("/all_documents", func(c *gin.Context) {
		documents, err := queryAllDocuments(esClient)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query documents"})
			return
		}
		c.JSON(http.StatusOK, documents)
	})

	// Define your endpoint routes
	router.GET("/projects-by-user", func(c *gin.Context) {
		getProjectsCreatedByUser(c, esClient)
	})

	router.GET("/projects-by-hashtags", func(c *gin.Context) {
		getProjectsWithHashtags(c, esClient)
	})

	router.GET("/projects-by-fuzzy-search", func(c *gin.Context) {
		fuzzySearchProjects(c, esClient)
	})

	// Set up a signal listener to handle server shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Println("Server is shutting down...")

		// Clean up by truncating or deleting the tables
		clearTables(pgDB)

		// Clear Elasticsearch indices
		clearElasticsearchIndices(esClient)

		os.Exit(0)
	}()

	// Start the server
	port := 8080
	router.Run(fmt.Sprintf(":%d", port))
}
