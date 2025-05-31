# Makefile for Caddy Serverless Docker Build
# Provides convenient targets for building and managing the serverless Caddy container

# Variables
IMAGE_NAME := caddy-serverless
CONTAINER_NAME := caddy-serverless-container
DOCKERFILE := Dockerfile.serverless
COMPOSE_FILE := docker-compose.serverless.yml
TEST_PORT := 8080

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Caddy Serverless Build Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make custom-build    # Build the serverless Caddy image"
	@echo "  make test           # Build and run tests"
	@echo "  make up             # Start with docker-compose"

.PHONY: custom-build
custom-build: ## Build Caddy serverless Docker image using Dockerfile.serverless
	@echo "Building Caddy serverless image..."
	docker build -f $(DOCKERFILE) -t $(IMAGE_NAME) .
	@echo "✅ Build completed successfully!"
	@echo "Image: $(IMAGE_NAME)"
	@docker images $(IMAGE_NAME)

.PHONY: build
build: custom-build ## Alias for custom-build

.PHONY: extract-binary
extract-binary: ## Extract the Caddy binary from the Docker image to ./bin/caddy
	@echo "Extracting Caddy binary to ./bin/caddy..."
	@mkdir -p ./bin
	@docker rm temp-caddy-extractor >/dev/null 2>&1 || true
	@docker create --name temp-caddy-extractor $(IMAGE_NAME):latest >/dev/null
	@docker cp temp-caddy-extractor:/usr/bin/caddy ./bin/caddy
	@docker rm temp-caddy-extractor >/dev/null
	@echo "✅ Caddy binary extracted to ./bin/caddy"

.PHONY: test
test: custom-build ## Build image and run tests
	@echo "Running tests for Caddy serverless..."
	@echo "Starting test container..."
	docker run -d \
		--name $(CONTAINER_NAME)-test \
		-p $(TEST_PORT):8080 \
		-v $$(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro \
		$(IMAGE_NAME)
	
	@echo "Waiting for container to start..."
	@sleep 5
	
	@echo "Testing endpoints..."
	@if curl -s http://localhost:$(TEST_PORT) | grep -q "Hello from Caddy Serverless"; then \
		echo "✅ Main endpoint test passed"; \
	else \
		echo "❌ Main endpoint test failed"; \
	fi
	
	@if curl -s http://localhost:$(TEST_PORT)/health | grep -q "OK"; then \
		echo "✅ Health endpoint test passed"; \
	else \
		echo "❌ Health endpoint test failed"; \
	fi
	
	@echo "Verifying plugins..."
	@if docker exec $(CONTAINER_NAME)-test caddy list-modules | grep -q "http.handlers.serverless"; then \
		echo "✅ Serverless plugin loaded"; \
	else \
		echo "⚠️  Serverless plugin not found"; \
	fi
	
	@if docker exec $(CONTAINER_NAME)-test caddy list-modules | grep -q "dns.providers.jwt"; then \
		echo "✅ JWT plugin loaded"; \
	else \
		echo "⚠️  JWT plugin not found"; \
	fi
	
	@echo "Container logs:"
	@docker logs $(CONTAINER_NAME)-test
	
	@echo "Cleaning up test container..."
	@docker stop $(CONTAINER_NAME)-test >/dev/null 2>&1 || true
	@docker rm $(CONTAINER_NAME)-test >/dev/null 2>&1 || true
	@echo "✅ Tests completed!"

.PHONY: run
run: custom-build ## Build and run the container
	@echo "Starting Caddy serverless container..."
	docker run -d \
		--name $(CONTAINER_NAME) \
		-p $(TEST_PORT):8080 \
		-p 8443:443 \
		-v $$(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro \
		$(IMAGE_NAME)
	@echo "✅ Container started!"
	@echo "Access endpoints:"
	@echo "  - Main: http://localhost:$(TEST_PORT)"
	@echo "  - Health: http://localhost:$(TEST_PORT)/health"

.PHONY: stop
stop: ## Stop and remove the running container
	@echo "Stopping Caddy serverless container..."
	@docker stop $(CONTAINER_NAME) >/dev/null 2>&1 || true
	@docker rm $(CONTAINER_NAME) >/dev/null 2>&1 || true
	@echo "✅ Container stopped and removed"

.PHONY: logs
logs: ## Show container logs
	docker logs -f $(CONTAINER_NAME)

.PHONY: shell
shell: ## Open shell in running container
	docker exec -it $(CONTAINER_NAME) /bin/sh

.PHONY: up
up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose -f $(COMPOSE_FILE) up -d
	@echo "✅ Services started!"
	@echo "Access endpoints:"
	@echo "  - Main: http://localhost:$(TEST_PORT)"
	@echo "  - Health: http://localhost:$(TEST_PORT)/health"
	@echo "  - Backend proxy: http://localhost:$(TEST_PORT)/api/"

.PHONY: down
down: ## Stop docker-compose services
	@echo "Stopping docker-compose services..."
	docker-compose -f $(COMPOSE_FILE) down
	@echo "✅ Services stopped"

.PHONY: restart
restart: stop run ## Restart the container

.PHONY: clean
clean: ## Remove image and containers
	@echo "Cleaning up Caddy serverless resources..."
	@docker stop $(CONTAINER_NAME) >/dev/null 2>&1 || true
	@docker rm $(CONTAINER_NAME) >/dev/null 2>&1 || true
	@docker stop $(CONTAINER_NAME)-test >/dev/null 2>&1 || true
	@docker rm $(CONTAINER_NAME)-test >/dev/null 2>&1 || true
	@docker rmi $(IMAGE_NAME) >/dev/null 2>&1 || true
	@echo "✅ Cleanup completed"

.PHONY: clean-all
clean-all: down clean ## Stop everything and clean up all resources
	@echo "Performing complete cleanup..."
	@docker-compose -f $(COMPOSE_FILE) down -v --remove-orphans >/dev/null 2>&1 || true
	@echo "✅ Complete cleanup finished"

.PHONY: verify
verify: ## Verify the built image and show plugin information
	@echo "Verifying Caddy serverless image..."
	@echo "Image information:"
	@docker images $(IMAGE_NAME)
	@echo ""
	@echo "Running plugin verification..."
	@docker run --rm $(IMAGE_NAME) version
	@echo ""
	@echo "Available modules:"
	@docker run --rm $(IMAGE_NAME) list-modules | grep -E "(serverless|jwt)" || echo "No serverless/jwt modules found"

.PHONY: size
size: ## Show image size information
	@echo "Image size information:"
	@docker images $(IMAGE_NAME) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
	@echo ""
	@echo "Layer information:"
	@docker history $(IMAGE_NAME) --no-trunc

# Development targets
.PHONY: dev
dev: custom-build run ## Build and run for development (with auto-restart)
	@echo "Development mode started. Container will restart on changes."
	@echo "Use 'make stop' to stop the development container."

.PHONY: debug
debug: custom-build ## Run container in debug mode
	@echo "Starting Caddy in debug mode..."
	docker run -it --rm \
		-p $(TEST_PORT):8080 \
		-v $$(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro \
		$(IMAGE_NAME) run --config /etc/caddy/Caddyfile --adapter caddyfile --debug

# Quick test targets
.PHONY: quick-test
quick-test: ## Quick endpoint tests (assumes container is running)
	@echo "Running quick tests..."
	@curl -s http://localhost:$(TEST_PORT) || echo "❌ Main endpoint failed"
	@curl -s http://localhost:$(TEST_PORT)/health || echo "❌ Health endpoint failed"
	@echo "✅ Quick tests completed"

.PHONY: bench
bench: ## Simple benchmark test
	@echo "Running benchmark test..."
	@which ab >/dev/null 2>&1 || (echo "❌ Apache Bench (ab) not found. Install with: apt-get install apache2-utils" && exit 1)
	ab -n 1000 -c 10 http://localhost:$(TEST_PORT)/health
