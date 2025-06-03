# Use a slim Go image for a smaller final image
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod ./
# Copy go.sum if it exists (for modules with dependencies)
COPY go.su[m] ./
RUN go mod download

# Copy the source code
COPY main.go .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /echoserver main.go

# Use a minimal alpine image for the final stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /echoserver .

# Expose the port the server will listen on (can be overridden by ECHOSERVER_PORT env var)
EXPOSE 8080

# Command to run the application
CMD ["./echoserver"]
