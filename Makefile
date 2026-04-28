DIST := dist

.PHONY: build build_all \
        build_linux_amd64 build_linux_arm64 \
        build_darwin_amd64 build_darwin_arm64 \
        build_windows_amd64 build_windows_arm64 \
        test test-race bench vet staticcheck gosec nilaway govulncheck \
        gitleaks golangci-lint check clean install-tools install-hooks

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

# Run all tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run all checks (lint + security + tests)
check: vet staticcheck gosec nilaway govulncheck test-race

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
	@echo "Install golangci-lint via: https://golangci-lint.run/welcome/install/"
	@echo "Install gitleaks via: https://github.com/gitleaks/gitleaks#installing"

# Install git pre-commit hook
install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."

clean:
	rm -f bausteinsicht
	rm -rf $(DIST)
