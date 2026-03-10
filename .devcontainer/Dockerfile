# Bausteinsicht development container
# Provides Go toolchain, static analysis, security scanners, draw.io CLI, Claude Code, and human CLI.

FROM mcr.microsoft.com/devcontainers/go:1.24

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# JRuby (used by asciidoctor-diagram in docToolchain) needs the system CA bundle
# to connect to kroki.io for server-side diagram rendering
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# ──────────────────────────────────────────────
# System packages (draw.io headless dependencies)
# ──────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    # draw.io / Electron headless requirements
    xvfb \
    xauth \
    dbus \
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
    # shell enhancements
    zsh \
    fzf \
    # firewall (for --dangerously-skip-permissions security)
    iptables \
    ipset \
    dnsutils \
    iproute2 \
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
# Pin draw.io version for reproducibility. Update and test before changing.
# ──────────────────────────────────────────────
ARG DRAWIO_VERSION=29.5.2
RUN wget -q "https://github.com/jgraph/drawio-desktop/releases/download/v${DRAWIO_VERSION}/drawio-amd64-${DRAWIO_VERSION}.deb" -O /tmp/drawio.deb \
 && apt-get update && apt-get install -y --no-install-recommends /tmp/drawio.deb \
 && rm -f /tmp/drawio.deb && rm -rf /var/lib/apt/lists/*

# Wrapper script so `drawio` runs headless via xvfb with dbus
# -a: auto-pick a free display number (avoids conflicts with existing Xvfb)
# --no-sandbox: required in container environments without user namespaces
# dbus: Electron requires a running system bus for IPC
RUN printf '#!/bin/sh\n# Ensure dbus is running (required by Electron/draw.io)\nif [ ! -S /run/dbus/system_bus_socket ]; then\n  sudo mkdir -p /run/dbus\n  sudo dbus-daemon --system --fork 2>/dev/null || true\nfi\nxvfb-run -a drawio --no-sandbox "$@"\n' > /usr/local/bin/drawio-export \
 && chmod +x /usr/local/bin/drawio-export

# ──────────────────────────────────────────────
# Firewall script (restricts outbound to whitelisted domains)
# Enables safe use of `claude --dangerously-skip-permissions`.
# ──────────────────────────────────────────────
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
RUN chmod +x /usr/local/bin/init-firewall.sh \
 && echo "vscode ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" >> /etc/sudoers.d/firewall

# ──────────────────────────────────────────────
# zsh setup (better interactive shell with fzf integration)
# ──────────────────────────────────────────────
RUN chsh -s /bin/zsh vscode \
 && mkdir -p /commandhistory \
 && touch /commandhistory/.zsh_history \
 && chown -R vscode:vscode /commandhistory

# ──────────────────────────────────────────────
# Claude Code CLI (native installer, no Node.js dependency)
# Install as vscode user so binary lands in ~/.local/bin/
# ──────────────────────────────────────────────
USER vscode
RUN curl -fsSL https://claude.ai/install.sh | bash
USER root

# ──────────────────────────────────────────────
# human CLI (gethuman.sh — AI agent issue tracker integration)
# ──────────────────────────────────────────────
RUN curl -sSfL https://gethuman.sh/install.sh | bash

# Fix permissions: let the vscode user write to Go module cache and GOPATH/bin
RUN mkdir -p /go/pkg/mod/cache /go/bin \
 && chown -R vscode:vscode /go

# Clean up
RUN apt-get clean && rm -rf /tmp/*
