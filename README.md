# Caddy Fly-Replay Module

A Caddy module that implements Fly.io's replay pattern for request routing with caching support for local development scenario. 

95% written by claude code.

## Features

- **Intelligent Request Routing**: Routes requests based on `fly-replay` headers from upstream services
- **Built-in Caching**: Cache routing decisions to reduce latency for subsequent requests
- **Cache Bypass Support**: Allows clients to skip cached routes when authorized
- **Cache Status Visibility**: Apps receive `fly-replay-cache-status` header indicating cache hit/miss/bypass
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

See test/Caddyfile.

## How It Works

1. **Initial Request**: Requests first go to the platform/router service
2. **Routing Decision**: Platform responds with `fly-replay: app=appname` header
3. **Caching**: Routing decision is cached based on `fly-replay-cache` pattern
4. **Replay**: Request is forwarded to the designated app with cache status
5. **Subsequent Requests**: Cached routes are used directly, bypassing the platform
6. **Cache Bypass**: Clients can skip cache with `fly-replay-cache-control: skip` when allowed

### Headers

#### Request Flow
- Custom headers are preserved throughout the routing chain
- Request bodies are buffered and preserved for POST/PUT operations
- Trace IDs are maintained for distributed tracing

#### Platform Response Headers
- `fly-replay`: Indicates which app should handle the request
- `fly-replay-cache`: Pattern for caching the routing decision
- `fly-replay-cache-ttl-secs`: Override default cache TTL
- `fly-replay-cache-allow-bypass`: Set to "yes" to allow cache bypass
- `X-Trace-ID`: Distributed tracing identifier

#### Client Request Headers
- `fly-replay-cache-control`: Set to "skip" to bypass cached routes (when allowed)

#### App Receives Headers
- `fly-replay-cache-status`: Cache status sent to your app
  - `hit`: Request served from cache, avoiding platform routing
  - `miss`: Cache miss, request was routed through platform
  - `bypass`: Cache bypassed at client's request (when allowed)
  - Absent: Request not served via replay mechanism

#### Debug Headers (when debug mode enabled)
- `X-Cache-Action`: STORED/INVALIDATED when cache is modified
- `X-Cache-Pattern`: The pattern used for caching
- `X-Cached-App`: App name when serving from cache
- `X-Cache-Allow-Bypass`: Indicates if bypass is allowed for this route
- `X-Forwarded-To`: Final destination domain

## Cache Bypass Example

When the platform sets cache with bypass allowed:
```
fly-replay: app=user123-app
fly-replay-cache: /en-US/user123/*
fly-replay-cache-allow-bypass: yes
```

Clients can then skip the cache:
```bash
curl -H "fly-replay-cache-control: skip" http://example.com/en-US/user123/profile
```

The app will receive:
```
fly-replay-cache-status: bypass
```

## Testing

The module includes integration tests:

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
- Cache status header (`fly-replay-cache-status`: hit/miss/bypass)
- Cache bypass functionality with `fly-replay-cache-control: skip`
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
