binary_name := "lazytf"
main_package := "./cmd/lazytf"
coverage_file := "coverage.out"
coverage_html := "coverage.html"

# Default recipe to display help
default:
    @just --list

# Spawn a nix development shell
shell:
    @echo "Spawning a nix development shell..."
    nix develop --command $SHELL

# Run the application
run *args:
    go run {{main_package}} {{args}}

# Run with hot reload using gow
dev *args:
    @echo "Running with hot reload..."
    @command -v gow >/dev/null 2>&1 || { echo "❌ gow not installed. Run 'just deps-tooling'"; exit 1; }
    gow run {{main_package}} {{args}}

# Build the application
build:
	@echo "Building {{binary_name}}..."
	go build -o bin/{{binary_name}} {{main_package}}
	@echo "Build complete: bin/{{binary_name}}"

# Install the application to $GOPATH/bin
install: build
    @echo "Installing {{binary_name}}..."
    go install {{main_package}}
    @echo "Install complete: {{binary_name}} installed to $(go env GOPATH)/bin"

# ===== Testing =====

# Run tests
test:
    @echo "Running tests..."
    go test -v -race ./...

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -race -coverprofile={{coverage_file}} -covermode=atomic ./...
    go tool cover -html={{coverage_file}} -o {{coverage_html}}
    @echo "Coverage report generated: {{coverage_html}}"

# Display test coverage in terminal
coverage:
    @echo "Running tests and displaying coverage..."
    go test -v -race -coverprofile={{coverage_file}} -covermode=atomic ./...
    go tool cover -func={{coverage_file}}

# ===== Formatting =====

# Format code with gofumpt and organize imports
fmt:
    @echo "Formatting code with gofumpt..."
    @command -v gofumpt >/dev/null 2>&1 || { echo "❌ gofumpt not installed. Refer flake.nix for installation"; exit 1; }
    gofumpt -l -w .
    @echo "Organizing imports with goimports..."
    @command -v goimports-reviser >/dev/null 2>&1 || { echo "❌ goimports-reviser not installed. Refer flake.nix for installation"; exit 1; }
    goimports-reviser -rm-unused ./...
    @echo "Running golangci-lint fmt..."
    @command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
    golangci-lint fmt ./...
    @echo "✓ Formatting complete"

# Format long lines with golines
fmt-lines:
    @echo "Formatting long lines with golines..."
    @command -v golines >/dev/null 2>&1 || { echo "❌ golines not installed. Run 'just deps-tooling'"; exit 1; }
    golines -w -m 120 --ignore-generated .
    @echo "✓ Line formatting complete"

# ===== Linting =====

# Run golangci-lint
lint:
    @echo "Running golangci-lint..."
    @command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
    golangci-lint run --timeout 5m ./...

# Run golangci-lint with auto-fix
lint-fix:
    @echo "Running golangci-lint with auto-fix..."
    @command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
    golangci-lint run --fix --timeout 5m ./...

# Run golangci-lint with all linters (no exclusions)
lint-all:
    @echo "Running golangci-lint with ALL linters..."
    @command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
    golangci-lint run --enable-all --timeout 5m ./...

# Run go vet
vet:
    @echo "Running go vet..."
    go vet ./...

# ===== Code Quality Checks =====

# Check code complexity (cyclomatic and cognitive)
complexity:
    @echo "=== Cyclomatic Complexity (threshold: 15) ==="
    @golangci-lint run --disable-all --enable gocyclo --timeout 5m ./... || true
    @echo ""
    @echo "=== Cognitive Complexity (threshold: 20) ==="
    @golangci-lint run --disable-all --enable gocognit --timeout 5m ./... || true
    @echo ""
    @echo "=== Function Length (threshold: 80 lines, 50 statements) ==="
    @golangci-lint run --disable-all --enable funlen --timeout 5m ./... || true
    @echo ""
    @echo "=== Nested If Complexity (threshold: 5) ==="
    @golangci-lint run --disable-all --enable nestif --timeout 5m ./... || true
    @echo ""
    @echo "=== Maintainability Index (threshold: under 20) ==="
    @golangci-lint run --disable-all --enable maintidx --timeout 5m ./... || true

# Find code duplication
dupl:
    @echo "=== Code Duplication Detection ==="
    @golangci-lint run --disable-all --enable dupl --timeout 5m ./... || true

# Find repeated strings that should be constants
constants:
    @echo "=== Repeated Strings (candidates for constants) ==="
    @golangci-lint run --disable-all --enable goconst --timeout 5m ./... || true

# Find TODO/FIXME/BUG comments
todo:
    @echo "=== TODO/FIXME/BUG/HACK Comments ==="
    @golangci-lint run --disable-all --enable godox --timeout 5m ./... || true

# Find unused code (variables, functions, types, constants)
unused:
    @echo "=== Unused Code Detection ==="
    @golangci-lint run --disable-all --enable unused --enable ineffassign --enable unparam --timeout 5m ./... || true

# Find dead code with deadcode tool
deadcode:
    @echo "=== Dead Code Detection ==="
    @command -v deadcode >/dev/null 2>&1 || { echo "Installing deadcode..."; go install golang.org/x/tools/cmd/deadcode@latest; }
    deadcode -test ./...

# ===== Security =====

# Run security checks with govulncheck
security:
    @echo "Running security checks..."
    @command -v govulncheck >/dev/null 2>&1 || { echo "❌ govulncheck not installed. Refer flake.nix for installation"; exit 1; }
    govulncheck ./...

# Run gosec security linter
gosec:
    @echo "=== Security Analysis (gosec) ==="
    @golangci-lint run --disable-all --enable gosec --timeout 5m ./... || true

# ===== Module Management =====

# Run go mod tidy
tidy:
    @echo "Tidying go modules..."
    go mod tidy
    go mod verify

# Check for unused dependencies
deps-unused:
    @echo "=== Checking for unused dependencies ==="
    @go mod tidy -v 2>&1 | grep -E "^unused" || echo "No unused dependencies found"

# Run go generate
generate:
    @echo "Running go generate..."
    go generate ./...

# ===== Combined Quality Checks =====

# Run all quality checks
check: fmt vet lint test
    @echo "✓ All checks passed!"

# Run all checks and generate coverage
check-all: fmt vet lint test-coverage security
    @echo "✓ All checks and coverage complete!"

# Run CI checks locally (mirrors GitHub Actions CI)
ci: tidy vet lint test-coverage build security
    @echo "✓ All CI checks passed locally!"

# Comprehensive code quality report (like Credo for Elixir)
quality:
    @echo "╔══════════════════════════════════════════════════════════════════╗"
    @echo "║              CODE QUALITY REPORT                                  ║"
    @echo "╚══════════════════════════════════════════════════════════════════╝"
    @echo ""
    @echo "▸ Running full lint analysis..."
    @golangci-lint run --timeout 5m ./... 2>&1 | tail -20 || true
    @echo ""
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "▸ Complexity Metrics"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @golangci-lint run --disable-all --enable gocyclo,gocognit,funlen,nestif,maintidx --timeout 5m ./... 2>&1 | head -30 || true
    @echo ""
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "▸ Code Duplication"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @golangci-lint run --disable-all --enable dupl --timeout 5m ./... 2>&1 | head -20 || true
    @echo ""
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "▸ Security Issues"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @golangci-lint run --disable-all --enable gosec --timeout 5m ./... 2>&1 | head -20 || true
    @echo ""
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "▸ TODO/FIXME Comments"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @golangci-lint run --disable-all --enable godox --timeout 5m ./... 2>&1 | head -20 || true
    @echo ""
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "▸ Unused Code"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @golangci-lint run --disable-all --enable unused,ineffassign,unparam --timeout 5m ./... 2>&1 | head -20 || true
    @echo ""
    @echo "╔══════════════════════════════════════════════════════════════════╗"
    @echo "║              QUALITY REPORT COMPLETE                              ║"
    @echo "╚══════════════════════════════════════════════════════════════════╝"

# Quick quality check (fast subset of linters)
quality-quick:
    @echo "Running quick quality check..."
    @golangci-lint run --disable-all \
        --enable errcheck \
        --enable govet \
        --enable staticcheck \
        --enable unused \
        --enable gosec \
        --timeout 2m ./...

# ===== Cleanup =====

# Clean build artifacts, cache, and modules
clean:
    @echo "Cleaning..."
    rm -rf bin/
    rm -f {{coverage_file}} {{coverage_html}}
    go clean -cache -testcache
    go clean -modcache
    @echo "✓ Clean complete"

# ===== Dependencies =====

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download

# Update dependencies
deps-up:
    @echo "Updating dependencies..."
    go get -u ./...
    go mod tidy

# Install Go tooling (for non-Nix users)
deps-tooling:
    @echo "Installing Go development tools..."
    @echo "→ Installing gopls..."
    go install golang.org/x/tools/gopls@latest
    @echo "→ Installing gofumpt..."
    go install mvdan.cc/gofumpt@latest
    @echo "→ Installing goimports-reviser..."
    go install github.com/incu6us/goimports-reviser/v3@latest
    @echo "→ Installing golines..."
    go install github.com/segmentio/golines@latest
    @echo "→ Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "→ Installing govulncheck..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
    @echo "→ Installing gow..."
    go install github.com/mitranim/gow@latest
    @echo "→ Installing deadcode..."
    go install golang.org/x/tools/cmd/deadcode@latest
    @echo "✓ All tools installed to $(go env GOPATH)/bin"
    @echo ""
    @echo "Make sure $(go env GOPATH)/bin is in your PATH"

# ===== Pre-commit Hook =====

# Install git pre-commit hook
hook-install:
    @echo "Installing pre-commit hook..."
    @echo '#!/bin/sh' > .git/hooks/pre-commit
    @echo 'just pre-commit' >> .git/hooks/pre-commit
    @chmod +x .git/hooks/pre-commit
    @echo "✓ Pre-commit hook installed"

# Run pre-commit checks
pre-commit:
    @echo "Running pre-commit checks..."
    @just fmt
    @just vet
    @just quality-quick
    @echo "✓ Pre-commit checks passed"
