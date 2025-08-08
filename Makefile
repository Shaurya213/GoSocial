.PHONY: generate-mocks test test-coverage test-race clean

# Generate mocks (Document requirement)
generate-mocks:
	@echo "🔧 Generating mocks as per requirement document..."
	@go generate ./internal/chat/service/...
	@go generate ./internal/chat/handler/...
	@echo "✅ Mocks generated successfully"

# Install test dependencies
install-deps:
	@echo "📦 Installing test dependencies..."
	@go get github.com/DATA-DOG/go-sqlmock@v1.5.0
	@go get github.com/golang/mock@v1.6.0
	@go get github.com/stretchr/testify@v1.8.4
	@go mod tidy

# Run all tests (Document requirement: ≥80% coverage)
test:
	@echo "🧪 Running unit tests (≥80% coverage target)..."
	@go test -v ./internal/...

# Test with coverage report (Document mandate)
test-coverage:
	@echo "📊 Checking coverage (Document requirement: ≥80%)..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "📄 Coverage report generated: coverage.html"

# Race detection (Document mandate)
test-race:
	@echo "⚡ Race detection (Document mandate)..."
	@go test -race ./internal/...
	@echo "✅ No race conditions detected"

# Run specific component tests
test-chat:
	@echo "💬 Running chat component tests..."
	@go test -v -race ./internal/chat/...

test-mongodb:
	@echo "🍃 Running MongoDB component tests..."
	@go test -v -race ./internal/dbmongo/...

test-config:
	@echo "⚙️  Running configuration tests..."
	@go test -v -race ./internal/config/...

# Clean test artifacts
clean:
	@rm -f coverage.out coverage.html
	@echo "🧹 Cleaned test artifacts"

# Development workflow
dev-test: install-deps generate-mocks test-coverage test-race
	@echo "🚀 Full development test suite completed"

