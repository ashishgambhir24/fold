# Setup
Clone this repo

## Installations

### Go
Make sure go is installed on your machine by checking go version - 
> go version

If go is not installed, install using following commad - 
> brew install go

### Postgres
Install or update postgresql version to `postgresql@15`-
> brew install postgresql@15

Start postgresql - 
>brew services start postgresql@15

Create postgres DB - 
> createdb fold-finance

Verify whether DB is created -
> psql -U <local_username> -h /tmp fold-finance

Postgres shell will open up. You can further use it for DB CRUD operations.

Also Update `POSTGRES_USER` variable in .env file of this project.
> POSTGRES_USER=<local_username>


### ElasticSearch
Download ElasticSearch Locally using this [link](https://www.elastic.co/downloads/elasticsearch) and follow its steps to start elasticsearch.
Navigate to elasticsearch folder and and make following changes in config/elasticsearch.yml -
> Uncomment `transport.host` setting at the end of the file

> turn `xpack.security.enabled` flag to `true`

In elasticsearch folder, start elasticsearch using following command - 
> bin/elasticsearch

## Service
Start data pipeline service using following command in this repo - 
> make run

It will start local server and bind it to localhost:8080

Some seed data is added to postgresDB which got synced to elasticsearch as well. You can apply CRUD operations on fold-finance DB tables from postgres shell and it would sync with elasticsearch as well.
All documents from elasticsearch could be fetched from API Endpoints.

# API Testing
Import `fold_data_pipeline.postman_collection.json` file in postman. It contains collection of all query Endpoints for elasticsearch.

# TroubleShooting
### psql command not found
If psql is not identified by the terminal, export postgres path in `.bashrc` file. Add following line in `.bashrc` file - 
> export PATH=/path/to/postgres/bin:$PATH






