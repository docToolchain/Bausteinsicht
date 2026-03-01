# Bausteinsicht development container
# Provides Go toolchain, static analysis, security scanners, draw.io CLI, Claude Code, and human CLI.

FROM mcr.microsoft.com/devcontainers/go:1.24

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# ──────────────────────────────────────────────
# System packages (draw.io headless dependencies)
# ──────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    # draw.io / Electron headless requirements
    xvfb \
    xauth \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libxss1 \
    libxtst6 \
    libatk-bridge2.0-0 \
    libgtk-3-0 \
    libnss3 \
    libasound2 \
    libgbm1 \
    libdrm2 \
    # general utilities
    curl \
    wget \
    jq \
    make \
    && rm -rf /var/lib/apt/lists/*

# ──────────────────────────────────────────────
# Go analysis & security tools (via go install)
# Some tools require newer Go than the base image; GOTOOLCHAIN=auto lets go download it.
# ──────────────────────────────────────────────
RUN GOTOOLCHAIN=auto go install honnef.co/go/tools/cmd/staticcheck@latest \
 && GOTOOLCHAIN=auto go install github.com/securego/gosec/v2/cmd/gosec@latest \
 && GOTOOLCHAIN=auto go install go.uber.org/nilaway/cmd/nilaway@latest \
 && GOTOOLCHAIN=auto go install golang.org/x/vuln/cmd/govulncheck@latest

# ──────────────────────────────────────────────
# golangci-lint (binary install, recommended by upstream)
# ──────────────────────────────────────────────
RUN curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b /usr/local/bin

# ──────────────────────────────────────────────
# gitleaks (secret scanner — prevents committing credentials)
# ──────────────────────────────────────────────
RUN GITLEAKS_VERSION=$(curl -sL https://api.github.com/repos/gitleaks/gitleaks/releases/latest | jq -r '.tag_name' | sed 's/^v//') \
 && wget -q "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" -O /tmp/gitleaks.tar.gz \
 && tar -xzf /tmp/gitleaks.tar.gz -C /usr/local/bin gitleaks \
 && rm -f /tmp/gitleaks.tar.gz

# ──────────────────────────────────────────────
# draw.io Desktop (headless export via xvfb-run)
# ──────────────────────────────────────────────
RUN DRAWIO_VERSION=$(curl -sL https://api.github.com/repos/jgraph/drawio-desktop/releases/latest | jq -r '.tag_name' | sed 's/^v//') \
 && wget -q "https://github.com/jgraph/drawio-desktop/releases/download/v${DRAWIO_VERSION}/drawio-amd64-${DRAWIO_VERSION}.deb" -O /tmp/drawio.deb \
 && apt-get update && apt-get install -y --no-install-recommends /tmp/drawio.deb \
 && rm -f /tmp/drawio.deb && rm -rf /var/lib/apt/lists/*

# Wrapper script so `drawio` runs headless via xvfb
RUN printf '#!/bin/sh\nxvfb-run drawio "$@"\n' > /usr/local/bin/drawio-export \
 && chmod +x /usr/local/bin/drawio-export

# ──────────────────────────────────────────────
# Node.js (needed for Claude Code CLI)
# ──────────────────────────────────────────────
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
 && apt-get install -y --no-install-recommends nodejs \
 && rm -rf /var/lib/apt/lists/*

# ──────────────────────────────────────────────
# Claude Code CLI
# ──────────────────────────────────────────────
RUN npm install -g @anthropic-ai/claude-code

# ──────────────────────────────────────────────
# human CLI (gethuman.sh — AI agent issue tracker integration)
# ──────────────────────────────────────────────
RUN curl -sSfL https://gethuman.sh/install.sh | bash

# Fix permissions: let the vscode user write to Go module cache and GOPATH/bin
RUN mkdir -p /go/pkg/mod/cache /go/bin \
 && chown -R vscode:vscode /go

# Clean up
RUN apt-get clean && rm -rf /tmp/*
