# Caddy Fly-Replay Module

A Caddy module that implements Fly.io's replay pattern for intelligent request routing with caching support.

95% written by claude code.

## Features

- **Intelligent Request Routing**: Routes requests based on `fly-replay` headers from upstream services
- **Built-in Caching**: Cache routing decisions to reduce latency for subsequent requests
- **Configurable TTL**: Control cache duration per routing pattern
- **Debug Mode**: Optional debug headers for monitoring routing decisions
- **Request Body Preservation**: Properly handles POST/PUT requests with bodies

## Installation

### Using xcaddy

```bash
xcaddy build --with github.com/kahgeh/caddy-fly-replay
```

### Local Development

```bash
xcaddy build --with github.com/kahgeh/caddy-fly-replay=.
```

## Configuration

### Caddyfile Example

```caddyfile
{
    order fly_replay before reverse_proxy
    auto_https off
}

http://localhost:3000 {
    fly_replay {
        enable_cache true
        cache_dir ./cache
        cache_ttl 300  # 5 minutes
        debug true
        
        # Map app names to destinations
        apps {
            user123-app {
                domain localhost:9001
            }
            user456-app {
                domain localhost:9002
            }
            user789-app {
                domain localhost:9003
            }
        }
    }
    
    # The platform/router service
    reverse_proxy localhost:8080
}
```

## How It Works

1. **Initial Request**: Requests first go to the platform/router service
2. **Routing Decision**: Platform responds with `fly-replay: app=appname` header
3. **Caching**: Routing decision is cached based on `fly-replay-cache` pattern
4. **Replay**: Request is forwarded to the designated app
5. **Subsequent Requests**: Cached routes are used directly, bypassing the platform

### Headers

#### Request Flow
- Custom headers are preserved throughout the routing chain
- Request bodies are buffered and preserved for POST/PUT operations
- Trace IDs are maintained for distributed tracing

#### Platform Response Headers
- `fly-replay`: Indicates which app should handle the request
- `fly-replay-cache`: Pattern for caching the routing decision
- `fly-replay-cache-ttl-secs`: Override default cache TTL
- `X-Trace-ID`: Distributed tracing identifier

#### Debug Headers (when enabled)
- `X-Cache`: HIT/MISS status
- `X-Cache-Action`: STORED/INVALIDATED
- `X-Cache-Pattern`: The pattern used for caching
- `X-Cached-App`: App name when serving from cache
- `X-Forwarded-To`: Final destination domain

## Testing

The module includes comprehensive integration tests:

```bash
# Run all tests
make test

# Start test environment
make start-test-env

# Run tests manually
cd test && go run integration_test_runner.go

# Stop test environment
make stop-test-env
```

### Test Coverage
- GET requests with custom headers
- POST/PUT requests with JSON bodies
- Cache behavior (MISS/HIT)
- Header forwarding and preservation
- Special characters in headers
- Large request bodies

## Development

### Project Structure
```
caddy-fly-replay/
├── cache.go           # Cache implementation
├── config.go          # Configuration structures
├── handler.go         # Main request handler
├── plugin.go          # Caddy module registration
├── go.mod            # Go module definition
├── Makefile          # Build and test automation
├── test/             # Integration tests
│   ├── platform.go
│   ├── user-app.go
│   └── integration_test_runner.go
└── examples/         # Example configurations
    └── Caddyfile.example
```

### Building

```bash
# Build Caddy with the module
make build

# Clean build artifacts
make clean
```

## License

MIT License

## Contributing

Contributions are welcome! Please ensure all tests pass before submitting a pull request.

```bash
# Run tests before committing
make test
```
