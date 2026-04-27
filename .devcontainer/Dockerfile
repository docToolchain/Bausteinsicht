# ══════════════════════════════════════════════════════════════
# SHARED BASE — identical across projects, do not edit per-project.
# To update: copy this block from the reference project.
# Configurable via ARGs: BASE_IMAGE, DEV_USER
# ══════════════════════════════════════════════════════════════
ARG BASE_IMAGE=mcr.microsoft.com/devcontainers/go:1.25
ARG DEV_USER=vscode

FROM ${BASE_IMAGE}

# Re-declare ARGs after FROM (Docker discards them)
ARG DEV_USER=vscode

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# UTF-8 locale for proper Unicode support (special chars, box-drawing, emojis)
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8

# ──────────────────────────────────────────────
# System packages (shared across all projects)
# ──────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    # general utilities
    curl \
    wget \
    jq \
    make \
    # shell enhancements
    zsh \
    fzf \
    tmux \
    # firewall (for --dangerously-skip-permissions security)
    iptables \
    ipset \
    dnsutils \
    iproute2 \
    # firewall logging (NFLOG userspace logger — iptables LOG doesn't work in containers)
    ulogd2 \
    && rm -rf /var/lib/apt/lists/*

# ──────────────────────────────────────────────
# Firewall script (restricts outbound to whitelisted domains)
# Enables safe use of `claude --dangerously-skip-permissions`.
# After running, the script:
#   - Locks iptables/ipset binaries (chmod 000)
#   - Restricts sudo to specific commands only
#   - Starts ulogd2 writing blocked connections to /var/log/firewall.log
# ──────────────────────────────────────────────
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
COPY firewall-log.sh /usr/local/bin/firewall-log
RUN chmod +x /usr/local/bin/firewall-log
COPY ulogd-firewall.conf /etc/ulogd-firewall.conf
RUN chmod +x /usr/local/bin/init-firewall.sh \
 && echo "${DEV_USER} ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" >> /etc/sudoers.d/firewall \
 && mkdir -p /var/log \
 && touch /var/log/firewall.log \
 && chown root:${DEV_USER} /var/log/firewall.log \
 && chmod 640 /var/log/firewall.log

# ──────────────────────────────────────────────
# zsh setup (better interactive shell with fzf integration)
# ──────────────────────────────────────────────
RUN chsh -s /bin/zsh ${DEV_USER} \
 && mkdir -p /commandhistory \
 && touch /commandhistory/.zsh_history \
 && chown -R ${DEV_USER}:${DEV_USER} /commandhistory \
 && echo 'PROMPT="%F{green}[SANDBOX]%f $PROMPT"' >> /etc/zsh/zshrc

# ──────────────────────────────────────────────
# tmux setup (mouse support enabled by default)
# ──────────────────────────────────────────────
RUN echo 'set -g mouse on' > /home/${DEV_USER}/.tmux.conf \
 && chown ${DEV_USER}:${DEV_USER} /home/${DEV_USER}/.tmux.conf

# ──────────────────────────────────────────────
# Login message: show firewall status on every terminal open
# ──────────────────────────────────────────────
COPY firewall-motd.sh /etc/profile.d/firewall-motd.sh
RUN chmod +r /etc/profile.d/firewall-motd.sh \
 && echo '[ -f /etc/profile.d/firewall-motd.sh ] && . /etc/profile.d/firewall-motd.sh' >> /etc/zsh/zshrc

# ──────────────────────────────────────────────
# Claude Code CLI + Kiro CLI (native installers)
# Install as dev user so binaries land in ~/.local/bin/
# ──────────────────────────────────────────────
USER ${DEV_USER}
RUN curl -fsSL https://claude.ai/install.sh | bash
RUN curl -fsSL https://cli.kiro.dev/install | bash
USER root

# ──────────────────────────────────────────────
# human CLI (gethuman.sh — AI agent issue tracker integration)
# ──────────────────────────────────────────────
RUN curl -sSfL https://gethuman.sh/install.sh | bash

# Clean up (shared)
RUN apt-get clean && rm -rf /tmp/*

# ══════════════════════════════════════════════════════════════
# END SHARED BASE
# ══════════════════════════════════════════════════════════════

# ══════════════════════════════════════════════════════════════
# PROJECT-SPECIFIC — everything below is unique to this project
# ══════════════════════════════════════════════════════════════

# JRuby (used by asciidoctor-diagram in docToolchain) needs the system CA bundle
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Project-specific system packages (draw.io headless dependencies)
RUN apt-get update && apt-get install -y --no-install-recommends \
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
    && rm -rf /var/lib/apt/lists/*

# Go analysis & security tools
RUN GOTOOLCHAIN=auto go install honnef.co/go/tools/cmd/staticcheck@latest \
 && GOTOOLCHAIN=auto go install github.com/securego/gosec/v2/cmd/gosec@latest \
 && GOTOOLCHAIN=auto go install go.uber.org/nilaway/cmd/nilaway@latest \
 && GOTOOLCHAIN=auto go install golang.org/x/vuln/cmd/govulncheck@latest

# golangci-lint (binary install, recommended by upstream)
RUN curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b /usr/local/bin

# gitleaks (secret scanner)
RUN GITLEAKS_VERSION=$(curl -sL https://api.github.com/repos/gitleaks/gitleaks/releases/latest | jq -r '.tag_name' | sed 's/^v//') \
 && wget -q "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" -O /tmp/gitleaks.tar.gz \
 && tar -xzf /tmp/gitleaks.tar.gz -C /usr/local/bin gitleaks \
 && rm -f /tmp/gitleaks.tar.gz

# draw.io Desktop (headless export via xvfb-run)
ARG DRAWIO_VERSION=29.5.2
RUN wget -q "https://github.com/jgraph/drawio-desktop/releases/download/v${DRAWIO_VERSION}/drawio-amd64-${DRAWIO_VERSION}.deb" -O /tmp/drawio.deb \
 && apt-get update && apt-get install -y --no-install-recommends /tmp/drawio.deb \
 && rm -f /tmp/drawio.deb && rm -rf /var/lib/apt/lists/*

RUN printf '#!/bin/sh\n# Ensure dbus is running (required by Electron/draw.io)\nif [ ! -S /run/dbus/system_bus_socket ]; then\n  sudo mkdir -p /run/dbus\n  sudo dbus-daemon --system --fork 2>/dev/null || true\nfi\nxvfb-run -a drawio --no-sandbox "$@"\n' > /usr/local/bin/drawio-export \
 && chmod +x /usr/local/bin/drawio-export

# Fix permissions: let the dev user write to Go module cache and GOPATH/bin
RUN mkdir -p /go/pkg/mod/cache /go/bin \
 && chown -R ${DEV_USER}:${DEV_USER} /go

# Final cleanup
RUN apt-get clean && rm -rf /tmp/*
