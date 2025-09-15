# Caddy Fly-Replay Integration Test Makefile

# Variables
BIN_DIR := bin
TEST_DIR := test
PLATFORM_BIN := $(BIN_DIR)/platform
USER_APP_BIN := $(BIN_DIR)/user-app

# Build targets
.PHONY: build
build: build-platform build-user-app

.PHONY: build-platform
build-platform: | $(BIN_DIR)
	@echo "Building platform..."
	@cd $(TEST_DIR) && go build -o ../$(PLATFORM_BIN) platform.go

.PHONY: build-user-app
build-user-app: | $(BIN_DIR)
	@echo "Building user-app..."
	@cd $(TEST_DIR) && go build -o ../$(USER_APP_BIN) user-app.go

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

# Start/Stop targets
.PHONY: start-caddy
start-caddy:
	@echo "Starting Caddy..."
	@if ! lsof -ti:3000 > /dev/null 2>&1; then \
		./caddy run --config test/Caddyfile > caddy.log 2>&1 & \
		echo "Caddy started on port 3000 (PID: $$!)"; \
	else \
		echo "Caddy already running on port 3000"; \
	fi

.PHONY: stop-caddy
stop-caddy:
	@echo "Stopping Caddy..."
	@lsof -ti:3000 | xargs kill -9 2>/dev/null || true
	@echo "Caddy stopped"

.PHONY: start-test-apps
start-test-apps: build
	@echo "Starting test apps..."
	@# Kill any existing processes on our ports
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9001 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9002 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9003 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@# Start platform
	@$(PLATFORM_BIN) > platform.log 2>&1 & \
		echo "Platform started on port 8080 (PID: $$!)"
	@# Start user apps
	@$(USER_APP_BIN) -port 9001 -user user123 > user123.log 2>&1 & \
		echo "User app (user123) started on port 9001 (PID: $$!)"
	@$(USER_APP_BIN) -port 9002 -user user456 > user456.log 2>&1 & \
		echo "User app (user456) started on port 9002 (PID: $$!)"
	@$(USER_APP_BIN) -port 9003 -user user789 > user789.log 2>&1 & \
		echo "User app (user789) started on port 9003 (PID: $$!)"
	@sleep 2

.PHONY: stop-test-apps
stop-test-apps:
	@echo "Stopping test apps..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9001 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9002 | xargs kill -9 2>/dev/null || true
	@lsof -ti:9003 | xargs kill -9 2>/dev/null || true
	@echo "Test apps stopped"

.PHONY: start-test-env
start-test-env: start-caddy start-test-apps
	@echo "Test environment started"

.PHONY: stop-test-env
stop-test-env: stop-test-apps stop-caddy
	@echo "Test environment stopped"

# Test target
.PHONY: test
test:
	@echo "=== Caddy Fly-Replay Integration Test ==="
	@# Check if services are already running
	@if ! curl -s http://localhost:3000 > /dev/null 2>&1; then \
		echo "Starting test environment..."; \
		$(MAKE) start-test-env; \
		sleep 2; \
		echo "Running integration tests..."; \
		cd $(TEST_DIR) && go run integration_test_runner.go; \
		TEST_EXIT_CODE=$$?; \
		echo "Stopping test environment..."; \
		cd .. && $(MAKE) stop-test-env; \
		exit $$TEST_EXIT_CODE; \
	else \
		echo "Services already running, executing tests..."; \
		cd $(TEST_DIR) && go run integration_test_runner.go; \
	fi

# Clean target
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@rm -f *.log
	@rm -f $(TEST_DIR)/*.log
	@echo "Cleanup complete"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build platform and user-app binaries"
	@echo "  build-platform  - Build only the platform binary"
	@echo "  build-user-app  - Build only the user-app binary"
	@echo "  start-caddy     - Start Caddy server"
	@echo "  stop-caddy      - Stop Caddy server"
	@echo "  start-test-apps - Build and start platform and user apps"
	@echo "  stop-test-apps  - Stop platform and user apps"
	@echo "  start-test-env  - Start complete test environment (Caddy + apps)"
	@echo "  stop-test-env   - Stop complete test environment"
	@echo "  test            - Run integration tests (auto-manages environment)"
	@echo "  clean           - Remove binaries and log files"
	@echo "  help            - Show this help message"
