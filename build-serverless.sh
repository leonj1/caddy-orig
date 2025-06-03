#!/bin/bash

# Caddy Serverless Docker Build Script
# This script builds and tests the Caddy serverless container

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

print_status "Starting Caddy Serverless build process..."

# Build the Docker image
print_status "Building Caddy serverless Docker image..."
if docker build -f Dockerfile.serverless -t caddy-serverless .; then
    print_success "Docker image built successfully!"
else
    print_error "Failed to build Docker image"
    exit 1
fi

# Check if we should run tests
if [[ "$1" == "--test" || "$1" == "-t" ]]; then
    print_status "Running tests..."
    
    # Start the container
    print_status "Starting Caddy serverless container..."
    docker run -d \
        --name caddy-serverless-test \
        -p 8080:8080 \
        -v "$(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro" \
        caddy-serverless
    
    # Wait for container to start
    print_status "Waiting for container to start..."
    sleep 5
    
    # Test endpoints
    print_status "Testing endpoints..."
    
    # Test main endpoint
    if curl -s http://localhost:8080 | grep -q "Hello from Caddy Serverless"; then
        print_success "Main endpoint test passed"
    else
        print_error "Main endpoint test failed"
    fi
    
    # Test health endpoint
    if curl -s http://localhost:8080/health | grep -q "OK"; then
        print_success "Health endpoint test passed"
    else
        print_error "Health endpoint test failed"
    fi
    
    # Test plugin verification
    if docker exec caddy-serverless-test caddy list-modules | grep -q "http.handlers.serverless"; then
        print_success "Serverless plugin loaded successfully"
    else
        print_warning "Serverless plugin not found in module list"
    fi
    
    if docker exec caddy-serverless-test caddy list-modules | grep -q "dns.providers.jwt"; then
        print_success "JWT plugin loaded successfully"
    else
        print_warning "JWT plugin not found in module list"
    fi
    
    # Show container logs
    print_status "Container logs:"
    docker logs caddy-serverless-test
    
    # Cleanup
    print_status "Cleaning up test container..."
    docker stop caddy-serverless-test >/dev/null 2>&1
    docker rm caddy-serverless-test >/dev/null 2>&1
    
    print_success "Tests completed!"
fi

# Check if we should use docker-compose
if [[ "$1" == "--compose" || "$1" == "-c" ]]; then
    print_status "Starting with docker-compose..."
    docker-compose -f docker-compose.serverless.yml up -d
    
    print_status "Waiting for services to start..."
    sleep 10
    
    print_success "Services started! Access your serverless Caddy at:"
    echo "  - Main endpoint: http://localhost:8080"
    echo "  - Health check: http://localhost:8080/health"
    echo "  - Backend proxy: http://localhost:8080/api/"
    echo ""
    echo "To stop services: docker-compose -f docker-compose.serverless.yml down"
    exit 0
fi

# Show usage information
print_success "Build completed successfully!"
echo ""
echo "Usage options:"
echo "  ./build-serverless.sh --test     # Build and run tests"
echo "  ./build-serverless.sh --compose  # Build and start with docker-compose"
echo ""
echo "Manual run commands:"
echo "  # Simple run:"
echo "  docker run -d -p 8080:8080 -v \$(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro caddy-serverless"
echo ""
echo "  # With docker-compose:"
echo "  docker-compose -f docker-compose.serverless.yml up -d"
echo ""
echo "Test endpoints:"
echo "  curl http://localhost:8080"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8080/api/"
