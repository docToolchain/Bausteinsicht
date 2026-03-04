.PHONY: build test test-race vet staticcheck gosec nilaway govulncheck gitleaks golangci-lint check clean install-tools install-hooks

# Ensure GOPATH/bin is in PATH for installed tools
export PATH := $(PATH):$(shell go env GOPATH)/bin

# Build
build:
	go build -o bausteinsicht ./cmd/bausteinsicht/

# Run all tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

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
