# Project variables
APP_NAME=rp-proxy
CONFIG=config.yaml
DOCKER_IMAGE=rp-proxy:v1.0
PORT?=8080
ENVIRONMENT?=dev

# Go commands
GO=go
GOFMT=gofmt -s -w
GOTEST=$(GO) test -v ./...
GOBUILD=$(GO) build -o $(APP_NAME)

# Default target
all: fmt test build

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) .

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST)

# Build binary
build:
	@echo "Building $(APP_NAME)..."
	$(GOBUILD)

# Run locally with config
run: build
	@echo "Running $(APP_NAME) on port $(PORT)..."
	./$(APP_NAME) -config $(CONFIG) -port $(PORT)

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	-$(RM) $(APP_NAME)

# ===== Docker Targets =====

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

# Run container with mounted config
docker-run:
	@echo "Running Docker container with ENVIRONMENT=$(ENVIRONMENT)..."
	docker-compose up --build

# Stop containers
docker-stop:
	docker-compose down
# Clean up dangling images/containers
docker-clean:
	@echo "Removing dangling images..."
	docker image prune -f
# ===== Environment Shortcuts =====

run-dev:
	@echo "Running in DEV environment..."
	$(MAKE) docker-run ENVIRONMENT=dev PORT=$(PORT)

run-staging:
	@echo "Running in STAGING environment..."
	$(MAKE) docker-run ENVIRONMENT=staging PORT=$(PORT) 

run-prod:
	@echo "Running in PROD environment..."
	ENVIRONMENT=prod PORT=$(PORT) docker-compose up --build -d
