SHELL := /bin/bash

run:
	go run main.go postgres.go elasticsearch.go get_mappings.go seed_data.go query.go sync_elasticsearch.go