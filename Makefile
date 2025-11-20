.PHONY: help build run test clean docker-up docker-down docker-logs

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go application
	go build -o tgw-neo4j main.go

run: ## Run the application
	go run main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f tgw-neo4j
	go clean

docker-up: ## Start Neo4j using Docker Compose
	docker-compose up -d
	@echo "Waiting for Neo4j to be ready..."
	@sleep 10
	@echo "Neo4j is ready!"
	@echo "Neo4j Browser: http://localhost:7474"
	@echo "Credentials: neo4j/password"

docker-down: ## Stop Neo4j
	docker-compose down

docker-logs: ## Show Neo4j logs
	docker-compose logs -f neo4j

docker-clean: ## Stop Neo4j and remove volumes
	docker-compose down -v

deps: ## Download Go dependencies
	go mod download
	go mod tidy

fmt: ## Format Go code
	go fmt ./...

lint: ## Run Go linter
	go vet ./...

all: deps fmt build ## Download deps, format, and build
