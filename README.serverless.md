# Caddy Serverless Docker Build

This repository contains a `Dockerfile.serverless` that builds a custom Caddy binary with the serverless plugin and JWT authentication support using xcaddy.

## Features

- **Serverless Plugin**: Built-in serverless functionality from `modules/caddyhttp/serverless/`
- **JWT Authentication**: External JWT plugin from `github.com/caddy-dns/jwt`
- **Multi-stage Build**: Optimized for minimal final image size (~25-30MB)
- **Alpine-based**: Secure, minimal runtime environment
- **Non-root Execution**: Security-hardened container
- **Health Checks**: Built-in container health monitoring

## Quick Start

### Build the Image

```bash
# Build the serverless Caddy image
docker build -f Dockerfile.serverless -t caddy-serverless .
```

### Run with Example Configuration

```bash
# Run with the example Caddyfile
docker run -d \
  --name caddy-serverless \
  -p 8080:8080 \
  -v $(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro \
  caddy-serverless
```

### Test the Deployment

```bash
# Test basic response
curl http://localhost:8080

# Test health endpoint
curl http://localhost:8080/health

# Test protected endpoint (will require JWT)
curl http://localhost:8080/protected/data
```

## Configuration

### Environment Variables

- `CADDY_CONFIG_DIR`: Configuration directory (default: `/config/caddy`)
- `CADDY_DATA_DIR`: Data directory (default: `/data/caddy`)

### Volume Mounts

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/your/Caddyfile:/etc/caddy/Caddyfile:ro \
  -v /path/to/static/files:/var/www:ro \
  -v caddy_data:/data/caddy \
  -v caddy_config:/config/caddy \
  caddy-serverless
```

## Serverless Deployment Examples

### AWS Lambda with Container Images

```bash
# Tag for ECR
docker tag caddy-serverless:latest 123456789012.dkr.ecr.us-east-1.amazonaws.com/caddy-serverless:latest

# Push to ECR
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/caddy-serverless:latest
```

### Google Cloud Run

```bash
# Tag for Google Container Registry
docker tag caddy-serverless:latest gcr.io/your-project/caddy-serverless:latest

# Push to GCR
docker push gcr.io/your-project/caddy-serverless:latest

# Deploy to Cloud Run
gcloud run deploy caddy-serverless \
  --image gcr.io/your-project/caddy-serverless:latest \
  --platform managed \
  --port 8080
```

### Azure Container Instances

```bash
# Create container group
az container create \
  --resource-group myResourceGroup \
  --name caddy-serverless \
  --image caddy-serverless:latest \
  --ports 8080 \
  --dns-name-label caddy-serverless-unique
```

## Customization

### Adding More Plugins

Modify the `xcaddy build` command in `Dockerfile.serverless`:

```dockerfile
RUN xcaddy build \
    --with github.com/caddyserver/caddy/v2/modules/caddyhttp/serverless=./modules/caddyhttp/serverless \
    --with github.com/caddy-dns/jwt \
    --with github.com/caddyserver/cache-handler \
    --with github.com/greenpau/caddy-security
```

### Custom Caddyfile

Create your own Caddyfile and mount it:

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/your/Caddyfile:/etc/caddy/Caddyfile:ro \
  caddy-serverless
```

## Security Considerations

- Container runs as non-root user (`caddy:caddy`)
- Minimal Alpine base image reduces attack surface
- CA certificates included for HTTPS verification
- Health checks for container monitoring
- Capability-based privilege management for port binding

## Troubleshooting

### Check Container Logs

```bash
docker logs caddy-serverless
```

### Debug Mode

```bash
docker run -it --rm \
  -p 8080:8080 \
  -v $(pwd)/Caddyfile.serverless.example:/etc/caddy/Caddyfile:ro \
  caddy-serverless run --config /etc/caddy/Caddyfile --adapter caddyfile --debug
```

### Verify Plugins

```bash
docker run --rm caddy-serverless list-modules
```

## Performance Optimization

### For Serverless Environments

- Disable automatic HTTPS (`auto_https off`)
- Disable admin API (`admin off`)
- Disable config persistence (`persist_config off`)
- Use serverless listener wrapper
- Minimize startup time with pre-compiled binary

### Memory Usage

The container typically uses:
- **Idle**: ~10-15MB RAM
- **Under load**: ~20-50MB RAM
- **Image size**: ~25-30MB

## License

This Dockerfile and configuration are provided under the same license as Caddy itself. See the main Caddy repository for license details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test the Docker build
5. Submit a pull request

## Support

For issues related to:
- **Caddy core**: [Caddy GitHub Issues](https://github.com/caddyserver/caddy/issues)
- **Serverless plugin**: Check the `modules/caddyhttp/serverless/` directory
- **JWT plugin**: [caddy-dns/jwt repository](https://github.com/caddy-dns/jwt)
- **Docker build**: Create an issue in this repository
