# Caddy Serverless Functions Plugin

This plugin enables Caddy to execute serverless functions using Docker containers. When a request matches a configured route, the plugin starts a Docker container, proxies the request to it, and returns the response.

## Features

- **HTTP Method Matching**: Support for GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- **Path Pattern Matching**: Regex-based URL path matching
- **Docker Integration**: Automatic container lifecycle management
- **Environment Variables**: Pass custom environment variables to containers
- **Volume Mounts**: Mount host directories into containers
- **Configurable Timeouts**: Set execution timeouts for functions
- **Port Configuration**: Specify container listening ports
- **Automatic Cleanup**: Containers are automatically stopped after execution

## Configuration

### JSON Configuration

```json
{
  "handler": "serverless",
  "functions": [
    {
      "methods": ["GET", "POST"],
      "path": "/api/users/.*",
      "image": "my-function:latest",
      "command": ["/app/handler"],
      "environment": {
        "DATABASE_URL": "postgres://localhost/mydb",
        "API_KEY": "secret"
      },
      "volumes": [
        {
          "source": "/host/data",
          "target": "/app/data",
          "readonly": true
        }
      ],
      "timeout": "30s",
      "port": 8080
    }
  ]
}
```

### Caddyfile Configuration

```caddyfile
example.com {
    serverless {
        function {
            methods GET POST
            path /api/users/.*
            image my-function:latest
            command /app/handler
            env DATABASE_URL=postgres://localhost/mydb
            env API_KEY=secret
            volume /host/data:/app/data:ro
            timeout 30s
            port 8080
        }
        
        function {
            methods DELETE
            path /api/admin/.*
            image admin-function:latest
            timeout 60s
        }
    }
}
```

## Configuration Options

### Function Configuration

- **methods** (required): Array of HTTP methods this function handles
- **path** (required): Regex pattern for URL path matching
- **image** (required): Docker image to run
- **command** (optional): Command to execute in the container
- **environment** (optional): Environment variables to pass to the container
- **volumes** (optional): Volume mounts for the container
- **timeout** (optional): Maximum execution time (default: 30s)
- **port** (optional): Port the container listens on (default: 8080)

### Volume Mount Configuration

- **source** (required): Absolute path on the host
- **target** (required): Absolute path in the container
- **readonly** (optional): Whether the mount is read-only (default: false)

## How It Works

1. **Request Matching**: When a request arrives, the plugin checks if it matches any configured function based on HTTP method and URL path
2. **Container Startup**: If a match is found, a new Docker container is started with the specified configuration
3. **Health Check**: The plugin waits for the container to be ready to accept connections
4. **Request Proxying**: The original HTTP request is proxied to the container
5. **Response Handling**: The container's response is returned to the client
6. **Cleanup**: The container is automatically stopped and removed

## Example Use Cases

### Simple API Function

```caddyfile
api.example.com {
    serverless {
        function {
            methods GET POST
            path /hello
            image nginx:alpine
            command /bin/sh -c "echo 'HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World' | nc -l -p 8080"
            port 8080
        }
    }
}
```

### Python Flask Application

```caddyfile
app.example.com {
    serverless {
        function {
            methods GET POST PUT DELETE
            path /.*
            image python:3.9-slim
            command python -c "
                from flask import Flask
                app = Flask(__name__)
                @app.route('/', defaults={'path': ''})
                @app.route('/<path:path>')
                def catch_all(path):
                    return f'Hello from {path}'
                app.run(host='0.0.0.0', port=8080)
            "
            env FLASK_ENV=production
            timeout 60s
            port 8080
        }
    }
}
```

### Node.js Function with Volume Mount

```caddyfile
node.example.com {
    serverless {
        function {
            methods GET
            path /files/.*
            image node:16-alpine
            command node -e "
                const http = require('http');
                const fs = require('fs');
                const server = http.createServer((req, res) => {
                    const files = fs.readdirSync('/data');
                    res.writeHead(200, {'Content-Type': 'application/json'});
                    res.end(JSON.stringify(files));
                });
                server.listen(8080, '0.0.0.0');
            "
            volume /host/files:/data:ro
            port 8080
        }
    }
}
```

## Requirements

- Docker must be installed and accessible via the `docker` command
- The Caddy process must have permission to execute Docker commands
- Docker images must be available locally or pullable from a registry

## Security Considerations

- Containers run with default Docker security settings
- Volume mounts should use absolute paths and appropriate permissions
- Consider using read-only mounts when possible
- Environment variables may contain sensitive data - handle with care
- Network isolation depends on Docker configuration

## Troubleshooting

### Container Fails to Start
- Check if the Docker image exists and is accessible
- Verify Docker daemon is running
- Check Caddy logs for detailed error messages

### Container Not Ready
- Ensure the container listens on the configured port
- Check if the application starts quickly enough within the timeout
- Verify the container doesn't exit immediately

### Request Timeout
- Increase the timeout value if functions need more time
- Check container logs for application errors
- Verify the container is responding on the correct port

## Performance Notes

- Each request starts a new container, which has overhead
- Consider container startup time when setting timeouts
- Use lightweight base images for faster startup
- Pre-built images start faster than those requiring compilation
