binary_name := "lazytf"
main_package := "./cmd/lazytf"
coverage_file := "coverage.out"
coverage_html := "coverage.html"

# ===== Core =====

# Default recipe to display help
[group('core')]
default:
	@just --list

# Build the application
[group('core')]
build:
	@echo "Building {{binary_name}}..."
	@version="$(git describe --tags --exact-match 2>/dev/null || true)"; if [ -n "$version" ]; then version="${version#v}"; else version="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"; fi; go build -ldflags "-s -w -X github.com/ushiradineth/lazytf/internal/consts.Version=$version" -o bin/{{binary_name}} {{main_package}}
	@echo "Build complete: bin/{{binary_name}}"

# Single verification gate command
[group('core')]
check: generate-check verify quality-gate integration e2e profiling-tests release-check nix-check
	@echo "✓ Verification gate passed!"

# Run with hot reload using gow
[group('core')]
dev *args:
	@echo "Running with hot reload..."
	@command -v gow >/dev/null 2>&1 || { echo "❌ gow not installed. Run 'just deps-tooling'"; exit 1; }
	TF_LOG="${TF_LOG:-TRACE}" TF_LOG_PATH="${TF_LOG_PATH:-/tmp/lazytf-tf.log}" gow run {{main_package}} {{args}}

# Format code with gofumpt and organize imports
[group('core')]
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
[group('core')]
fmt-lines:
	@echo "Formatting long lines with golines..."
	@command -v golines >/dev/null 2>&1 || { echo "❌ golines not installed. Run 'just deps-tooling'"; exit 1; }
	golines -w -m 120 --ignore-generated .
	@echo "✓ Line formatting complete"

# Run go generate
[group('core')]
generate:
	@echo "Running go generate..."
	go generate ./...

# Verify generated files are up to date
[group('core')]
generate-check:
	@echo "Running generators and checking freshness..."
	@before_status="$(git status --porcelain)"; go generate ./...; after_status="$(git status --porcelain)"; if [ "$before_status" != "$after_status" ]; then git status --short; git diff --stat; echo "❌ Generated files are out of date. Run 'just generate' and commit changes."; exit 1; fi

# Install the application to $GOPATH/bin
[group('core')]
install: build
	@echo "Installing {{binary_name}}..."
	@version="$(git describe --tags --exact-match 2>/dev/null || true)"; if [ -n "$version" ]; then version="${version#v}"; else version="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"; fi; go install -ldflags "-s -w -X github.com/ushiradineth/lazytf/internal/consts.Version=$version" {{main_package}}
	@echo "Install complete: {{binary_name}} installed to $(go env GOPATH)/bin"

# Run golangci-lint with selectable mode: normal, fix, all
[group('core')]
lint mode="normal":
	@echo "Running golangci-lint (mode={{mode}})..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
	@case "{{mode}}" in \
		normal) golangci-lint run --timeout 5m ./... ;; \
		fix) golangci-lint run --fix --timeout 5m ./... ;; \
		all) golangci-lint run --enable-all --timeout 5m ./... ;; \
		*) echo "❌ Unknown lint mode '{{mode}}'. Use: normal, fix, all"; exit 1 ;; \
	esac

# Run nix flake checks
[group('core')]
nix-check:
	@echo "Running nix checks..."
	nix flake check

# Run quality checks with coverage and non-blocking security scan
[group('core')]
quality-gate:
	@echo "Running quality checks..."
	just test coverage
	just security || true

# Run the application
[group('core')]
run *args:
	@version="$(git describe --tags --exact-match 2>/dev/null || true)"; if [ -n "$version" ]; then version="${version#v}"; else version="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"; fi; go run -ldflags "-X github.com/ushiradineth/lazytf/internal/consts.Version=$version" {{main_package}} {{args}}

# Run CI-equivalent verify checks
[group('core')]
verify:
	@echo "Running verify checks..."
	go mod verify
	go vet ./...
	go test ./...
	@command -v golangci-lint >/dev/null 2>&1 || { echo "❌ golangci-lint not installed. Refer flake.nix for installation"; exit 1; }
	BASE_REV=$(git rev-parse HEAD~1 2>/dev/null || git rev-list --max-parents=0 HEAD); golangci-lint run --timeout 5m --new-from-rev="$BASE_REV" ./...
	go build ./...

# Run go vet
[group('core')]
vet:
	@echo "Running go vet..."
	go vet ./...

# ===== Diagnostics =====

# Check code complexity (cyclomatic and cognitive)
[group('diagnostics')]
complexity:
	@echo "=== Cyclomatic Complexity (threshold: 15) ==="
	@golangci-lint run --default=none --enable gocyclo --timeout 5m ./... || true
	@echo ""
	@echo "=== Cognitive Complexity (threshold: 20) ==="
	@golangci-lint run --default=none --enable gocognit --timeout 5m ./... || true
	@echo ""
	@echo "=== Function Length (threshold: 80 lines, 50 statements) ==="
	@golangci-lint run --default=none --enable funlen --timeout 5m ./... || true
	@echo ""
	@echo "=== Nested If Complexity (threshold: 5) ==="
	@golangci-lint run --default=none --enable nestif --timeout 5m ./... || true
	@echo ""
	@echo "=== Maintainability Index (threshold: under 20) ==="
	@golangci-lint run --default=none --enable maintidx --timeout 5m ./... || true

# Find repeated strings that should be constants
[group('diagnostics')]
constants:
	@echo "=== Repeated Strings (candidates for constants) ==="
	@golangci-lint run --default=none --enable goconst --timeout 5m ./... || true

# Find dead code with deadcode tool
[group('diagnostics')]
deadcode:
	@echo "=== Dead Code Detection ==="
	@command -v deadcode >/dev/null 2>&1 || { echo "Installing deadcode..."; go install golang.org/x/tools/cmd/deadcode@latest; }
	deadcode -test ./...

# Find code duplication
[group('diagnostics')]
dupl:
	@echo "=== Code Duplication Detection ==="
	@golangci-lint run --default=none --enable dupl --timeout 5m ./... || true

# Run gosec security linter
[group('diagnostics')]
gosec:
	@echo "=== Security Analysis (gosec) ==="
	@golangci-lint run --default=none --enable gosec --timeout 5m ./... || true

# Run mutation testing for a target package
[group('diagnostics')]
mutest package="./internal/diff":
	@echo "Running go-mutesting for {{package}}..."
	@command -v go-mutesting >/dev/null 2>&1 || { echo "Installing go-mutesting..."; go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest; }
	go-mutesting {{package}}

# Comprehensive code quality report (like Credo for Elixir)
[group('diagnostics')]
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
	@golangci-lint run --default=none --enable gocyclo,gocognit,funlen,nestif,maintidx --timeout 5m ./... 2>&1 | head -30 || true
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "▸ Code Duplication"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@golangci-lint run --default=none --enable dupl --timeout 5m ./... 2>&1 | head -20 || true
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "▸ Security Issues"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@golangci-lint run --default=none --enable gosec --timeout 5m ./... 2>&1 | head -20 || true
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "▸ TODO/FIXME Comments"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@golangci-lint run --default=none --enable godox --timeout 5m ./... 2>&1 | head -20 || true
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "▸ Unused Code"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@golangci-lint run --default=none --enable unused,ineffassign,unparam --timeout 5m ./... 2>&1 | head -20 || true
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════╗"
	@echo "║              QUALITY REPORT COMPLETE                              ║"
	@echo "╚══════════════════════════════════════════════════════════════════╝"

# Quick quality check (fast subset of linters)
[group('diagnostics')]
quality-quick:
	@echo "Running quick quality check..."
	@golangci-lint run \
		--enable-only errcheck,govet,staticcheck,unused,gosec \
		--timeout 2m ./...

# Run security checks with govulncheck
[group('diagnostics')]
security:
	@echo "Running security checks..."
	@command -v govulncheck >/dev/null 2>&1 || { echo "❌ govulncheck not installed. Refer flake.nix for installation"; exit 1; }
	GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.8}" govulncheck ./...

# Find TODO/FIXME/BUG comments
[group('diagnostics')]
todo:
	@echo "=== TODO/FIXME/BUG/HACK Comments ==="
	@golangci-lint run --default=none --enable godox --timeout 5m ./... || true

# Find unused code (variables, functions, types, constants)
[group('diagnostics')]
unused:
	@echo "=== Unused Code Detection ==="
	@golangci-lint run --default=none --enable unused --enable ineffassign --enable unparam --timeout 5m ./... || true

# ===== Maintenance =====

# Clean build artifacts, cache, and modules
[group('maintenance')]
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f {{coverage_file}} {{coverage_html}}
	go clean -cache -testcache
	go clean -modcache
	@echo "✓ Clean complete"

# Download dependencies
[group('maintenance')]
deps:
	@echo "Downloading dependencies..."
	go mod download

# Install Go tooling (for non-Nix users)
[group('maintenance')]
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
	@echo "→ Installing go-mutesting..."
	go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest
	@echo "✓ All tools installed to $(go env GOPATH)/bin"
	@echo ""
	@echo "Make sure $(go env GOPATH)/bin is in your PATH"

# Check for unused dependencies
[group('maintenance')]
deps-unused:
	@echo "=== Checking for unused dependencies ==="
	@go mod tidy -v 2>&1 | grep -E "^unused" || echo "No unused dependencies found"

# Update dependencies
[group('maintenance')]
deps-up:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Spawn a nix development shell
[group('maintenance')]
shell:
	@echo "Spawning a nix development shell..."
	nix develop --command $SHELL

# Run go mod tidy
[group('maintenance')]
tidy:
	@echo "Tidying go modules..."
	go mod tidy
	go mod verify

# ===== Profiling =====

# Run benchmarks with profiling
[group('profiling')]
bench *pattern:
	@echo "Running benchmarks..."
	go test -bench={{ if pattern == "" { "." } else { pattern } }} -benchmem -cpuprofile=bench-cpu.prof -memprofile=bench-mem.prof ./internal/ui/components/
	@echo "Benchmark profiles written: bench-cpu.prof, bench-mem.prof"

# Run with all profiling
[group('profiling')]
profile-all *args:
	@echo "Running with all profiling (cpu,mem,trace,stats)..."
	LAZYTF_PROFILE=all go run {{main_package}} {{args}}
	@echo "Profiles written. Analyze with: just profile-analyze"

# Analyze profile files
[group('profiling')]
profile-analyze type="help":
	./scripts/profile-analyze.sh {{type}}

# Clean profile files
[group('profiling')]
profile-clean:
	@echo "Cleaning profile files..."
	rm -f lazytf-*.prof lazytf-*.out lazytf-*.csv
	@echo "✓ Profile files cleaned"

# Run with CPU profiling
[group('profiling')]
profile-cpu *args:
	@echo "Running with CPU profiling..."
	LAZYTF_PROFILE=cpu go run {{main_package}} {{args}}
	@echo "CPU profile written. Analyze with: just profile-analyze cpu"

# List profile files
[group('profiling')]
profile-list:
	@echo "Available profile files:"
	@ls -lh lazytf-*.prof lazytf-*.out lazytf-*.csv 2>/dev/null || echo "No profile files found"

# Run with memory profiling
[group('profiling')]
profile-mem *args:
	@echo "Running with memory profiling..."
	LAZYTF_PROFILE=mem go run {{main_package}} {{args}}
	@echo "Memory profile written. Analyze with: just profile-analyze mem"

# ===== Release =====

# Validate GoReleaser configuration
[group('release')]
release-check:
	@echo "Running release config checks..."
	go run github.com/goreleaser/goreleaser/v2@v2.12.7 check

# Build local snapshot release artifacts (no publish)
[group('release')]
release-snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "❌ goreleaser not installed"; exit 1; }
	goreleaser release --snapshot --clean

# ===== Testing =====

# Run end-to-end tests
[group('testing')]
e2e:
	@echo "Running e2e tests..."
	go test -tags=e2e ./test/e2e

# Run integration tests
[group('testing')]
integration:
	@echo "Running integration tests..."
	go test -tags=integration ./test/integration

# Run profiling tests
[group('testing')]
profiling-tests:
	@echo "Running profiling tests..."
	go test -run '^TestProfiler' ./internal/profile

# Run tests with selectable mode: default, coverage, summary
[group('testing')]
test mode="default":
	@echo "Running tests (mode={{mode}})..."
	@case "{{mode}}" in \
		default) go test -v -race ./... ;; \
		coverage) go test -v -race -coverprofile={{coverage_file}} -covermode=atomic ./... && go tool cover -html={{coverage_file}} -o {{coverage_html}} && echo "Coverage report generated: {{coverage_html}}" ;; \
		summary) go test -v -race -coverprofile={{coverage_file}} -covermode=atomic ./... && go tool cover -func={{coverage_file}} ;; \
		*) echo "❌ Unknown test mode '{{mode}}'. Use: default, coverage, summary"; exit 1 ;; \
	esac
