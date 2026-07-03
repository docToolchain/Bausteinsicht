DIST := dist

.PHONY: build build_all \
        build_linux_amd64 build_linux_arm64 \
        build_darwin_amd64 build_darwin_arm64 \
        build_windows_amd64 build_windows_arm64 \
        schema-generate schema-validate \
        build-extension package-extension \
        test test-race bench coverage coverage-report e2e-test-report arc42-docs arc42-sequences arc42-drift-check vet staticcheck gosec nilaway govulncheck deadcode \
        gitleaks golangci-lint check check-duplicates clean install-tools install-hooks

# Ensure GOPATH/bin is in PATH for installed tools
export PATH := $(PATH):$(shell go env GOPATH)/bin

# Build for the current platform
build:
	go build -o bausteinsicht ./cmd/bausteinsicht/

# Build for all supported platforms → dist/
build_all: build_linux_amd64 build_linux_arm64 build_darwin_amd64 build_darwin_arm64 build_windows_amd64 build_windows_arm64

build_linux_amd64:
	@mkdir -p $(DIST)/bausteinsicht_linux_amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(DIST)/bausteinsicht_linux_amd64/bausteinsicht ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_linux_amd64/bausteinsicht"

build_linux_arm64:
	@mkdir -p $(DIST)/bausteinsicht_linux_arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(DIST)/bausteinsicht_linux_arm64/bausteinsicht ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_linux_arm64/bausteinsicht"

build_darwin_amd64:
	@mkdir -p $(DIST)/bausteinsicht_darwin_amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(DIST)/bausteinsicht_darwin_amd64/bausteinsicht ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_darwin_amd64/bausteinsicht"

build_darwin_arm64:
	@mkdir -p $(DIST)/bausteinsicht_darwin_arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(DIST)/bausteinsicht_darwin_arm64/bausteinsicht ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_darwin_arm64/bausteinsicht"

build_windows_amd64:
	@mkdir -p $(DIST)/bausteinsicht_windows_amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(DIST)/bausteinsicht_windows_amd64/bausteinsicht.exe ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_windows_amd64/bausteinsicht.exe"

build_windows_arm64:
	@mkdir -p $(DIST)/bausteinsicht_windows_arm64
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $(DIST)/bausteinsicht_windows_arm64/bausteinsicht.exe ./cmd/bausteinsicht/
	@echo "→ $(DIST)/bausteinsicht_windows_arm64/bausteinsicht.exe"

# JSON Schema — Generate schema from Go types
schema-generate: build
	./bausteinsicht schema generate
	@echo "✅ Schema generated"

# JSON Schema — Validate schema is current (CI check)
schema-validate: build
	./bausteinsicht schema generate
	@if git diff --quiet schemas/bausteinsicht.schema.json; then \
		echo "✅ Schema is up to date"; \
	else \
		echo "❌ Schema is out of date. Run 'make schema-generate' and commit."; \
		exit 1; \
	fi

# VS Code Extension — Build VSIX package with all dependencies
build-extension:
	cd vscode-extension && npm install

package-extension: build-extension
	cd vscode-extension && npm run build && npm run package
	@echo "→ vscode-extension/bausteinsicht-0.1.0.vsix"

# Run all tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Measure code coverage and generate reports
coverage:
	@mkdir -p coverage
	@echo "Measuring code coverage..."
	@go test -coverprofile=coverage/coverage.out ./... > /dev/null 2>&1 || true
	@if [ -f coverage/coverage.out ]; then \
		go tool cover -html=coverage/coverage.out -o coverage/coverage.html; \
		echo "📊 Coverage report generated: coverage/coverage.html"; \
		echo ""; \
		echo "Coverage by package:"; \
		go tool cover -func=coverage/coverage.out | awk 'NR>1 {printf "  %-50s %s\n", $$1, $$NF}' | sort -k2 -rn; \
	else \
		echo "⚠️  Coverage report could not be generated"; \
	fi

# Side-by-side unit vs. E2E coverage comparison (writes coverage-report.md)
coverage-report:
	@echo "Generating side-by-side coverage report..."
	@scripts/coverage-report.sh > coverage-report.md
	@echo "📊 Report written to coverage-report.md"

# Line-accurate PASS/FAIL/SKIP report for E2E-Test-Plan.adoc (see #519)
e2e-test-report:
	@echo "Generating line-accurate E2E test report..."
	@scripts/e2e-test-report.sh > e2e-test-report.adoc
	@echo "📋 Report written to e2e-test-report.adoc"

# Regenerate arc42 chapter 5's generated artifacts (element tables, .puml,
# PNG diagrams) from src/docs/arc42/architecture.jsonc (see #524, #526)
arc42-docs:
	@scripts/generate-arc42-docs.sh

# Regenerate chapter 6's runtime sequence diagrams from `dynamicViews` in
# architecture.jsonc (see #535). No-op until dynamicViews content exists.
arc42-sequences:
	@scripts/generate-arc42-sequences.sh

# Verify every real internal/ and cmd/ package has a matching container
# element in architecture.jsonc, and vice versa (see #524, #526)
arc42-drift-check:
	@scripts/check-arc42-drift.sh

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Check for duplicate branches
check-duplicates:
	@echo "🔍 Checking for duplicate branches..."
	bash scripts/check-duplicate-branches.sh

# Run all checks (lint + security + tests)
check: vet staticcheck gosec nilaway govulncheck test-race schema-validate

# go vet — built-in static analysis
vet:
	go vet ./...

# staticcheck — advanced static analysis
staticcheck:
	staticcheck ./...

# gosec — security scanner
gosec:
	gosec ./...

# nilaway — nil pointer analysis
nilaway:
	nilaway ./...

# govulncheck — vulnerability scanner
govulncheck:
	govulncheck ./...

# deadcode — dead code detector (report only, does not fail)
deadcode:
	deadcode ./... || true
	@echo "✅ Deadcode scan complete"

# gitleaks — scan for secrets
gitleaks:
	gitleaks detect --source . --no-git

# golangci-lint — meta-linter (includes many linters)
golangci-lint:
	golangci-lint run ./...

# Install all required tools
install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install go.uber.org/nilaway/cmd/nilaway@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/deadcode@latest
	@echo "Install golangci-lint via: https://golangci-lint.run/welcome/install/"
	@echo "Install gitleaks via: https://github.com/gitleaks/gitleaks#installing"

# Install git hooks (pre-commit and pre-push)
install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	cp scripts/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	@echo "Git hooks installed (pre-commit, pre-push)."

clean:
	rm -f bausteinsicht
	rm -rf $(DIST)
