#!/bin/bash
# Firewall for devcontainer — restricts outbound to whitelisted domains only.
# Adapted from the Claude Code reference devcontainer.
# Enables safe use of `claude --dangerously-skip-permissions`.
#
# Security features:
# - Logs blocked connections to /var/log/firewall.log (via ulogd2 NFLOG)
# - Locks down iptables binaries after setup (prevents manipulation)
# - Restricts sudo access to prevent firewall changes
set -euo pipefail
IFS=$'\n\t'

LOGFILE="/var/log/firewall.log"

# Prepare log file (readable by vscode user, writable only by root)
touch "$LOGFILE"
chown root:vscode "$LOGFILE"
chmod 640 "$LOGFILE"

# ──────────────────────────────────────────────
# Start ulogd2 for NFLOG-based firewall logging
# (iptables LOG doesn't work in Docker — writes to host kernel log)
# ──────────────────────────────────────────────
echo "Starting ulogd2 firewall logger..."
ulogd -c /etc/ulogd-firewall.conf -d
sleep 0.5
if pgrep -x ulogd > /dev/null; then
    echo "ulogd2 started (PID: $(pgrep -x ulogd)), writing to $LOGFILE"
else
    echo "WARNING: ulogd2 failed to start:"
    cat /var/log/ulogd.log 2>/dev/null || echo "  (no log file)"
fi

# 1. Extract Docker DNS info BEFORE any flushing
DOCKER_DNS_RULES=$(iptables-save -t nat | grep "127\.0\.0\.11" || true)

# Flush existing rules and delete existing ipsets
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
ipset destroy allowed-domains 2>/dev/null || true

# 2. Selectively restore ONLY internal Docker DNS resolution
if [ -n "$DOCKER_DNS_RULES" ]; then
    echo "Restoring Docker DNS rules..."
    iptables -t nat -N DOCKER_OUTPUT 2>/dev/null || true
    iptables -t nat -N DOCKER_POSTROUTING 2>/dev/null || true
    echo "$DOCKER_DNS_RULES" | xargs -L 1 iptables -t nat
else
    echo "No Docker DNS rules to restore"
fi

# First allow DNS and localhost before any restrictions
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A INPUT -p udp --sport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --sport 22 -m state --state ESTABLISHED -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# Create ipset with CIDR support
ipset create allowed-domains hash:net

# Fetch GitHub IP ranges
echo "Fetching GitHub IP ranges..."
gh_ranges=$(curl -s https://api.github.com/meta)
if [ -z "$gh_ranges" ]; then
    echo "ERROR: Failed to fetch GitHub IP ranges"
    exit 1
fi

if ! echo "$gh_ranges" | jq -e '.web and .api and .git' >/dev/null; then
    echo "ERROR: GitHub API response missing required fields"
    exit 1
fi

echo "Processing GitHub IPs..."
while read -r cidr; do
    if [[ ! "$cidr" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/[0-9]{1,2}$ ]]; then
        echo "WARNING: Skipping invalid CIDR range from GitHub meta: $cidr"
        continue
    fi
    ipset add allowed-domains "$cidr" 2>/dev/null || true
done < <(echo "$gh_ranges" | jq -r '(.web + .api + .git)[]')

# Resolve and add other allowed domains (parallel for speed).
# Anthropic API + telemetry, Go module proxy, Docker Hub, VS Code marketplace, gethuman.
DOMAINS="api.anthropic.com sentry.io statsig.anthropic.com statsig.com \
proxy.golang.org sum.golang.org storage.googleapis.com gethuman.sh cli.kiro.dev \
oidc.us-east-1.amazonaws.com registry-1.docker.io auth.docker.io \
production.cloudflare.docker.com marketplace.visualstudio.com \
vscode.blob.core.windows.net update.code.visualstudio.com"

echo "Resolving $(echo $DOMAINS | wc -w) domains in parallel..."
RESOLVED_IPS=$(echo "$DOMAINS" | tr ' ' '\n' | xargs -P 8 -I{} sh -c \
    'dig +noall +answer +short A "$1" 2>/dev/null' _ {} | sort -u)

while read -r ip; do
    if [[ "$ip" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        ipset add allowed-domains "$ip" 2>/dev/null || true
    fi
done < <(echo "$RESOLVED_IPS")
echo "Resolved $(echo "$RESOLVED_IPS" | grep -c '^[0-9]') IPs"

# Get host IP from default route
HOST_IP=$(ip route | grep default | cut -d" " -f3)
if [ -z "$HOST_IP" ]; then
    echo "ERROR: Failed to detect host IP"
    exit 1
fi

HOST_NETWORK=$(echo "$HOST_IP" | sed "s/\.[0-9]*$/.0\/24/")
echo "Host network detected as: $HOST_NETWORK"

# Allow host network (for Docker host communication)
iptables -A INPUT -s "$HOST_NETWORK" -j ACCEPT
iptables -A OUTPUT -d "$HOST_NETWORK" -j ACCEPT

# Set default policies to DROP
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT DROP

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow only outbound traffic to whitelisted domains
iptables -A OUTPUT -m set --match-set allowed-domains dst -j ACCEPT

# Log blocked connections via NFLOG (userspace, rate-limited to avoid flooding)
iptables -A OUTPUT -m limit --limit 10/min --limit-burst 30 \
    -j NFLOG --nflog-group 1 --nflog-prefix "FW-BLOCKED:"
iptables -A OUTPUT -j REJECT --reject-with icmp-admin-prohibited

echo "Firewall configuration complete"
echo "Verifying firewall rules..."
if curl --connect-timeout 5 https://example.com >/dev/null 2>&1; then
    echo "ERROR: Firewall verification failed - was able to reach https://example.com"
    exit 1
else
    echo "OK: example.com blocked as expected"
fi

if ! curl --connect-timeout 5 https://api.github.com/zen >/dev/null 2>&1; then
    echo "ERROR: Firewall verification failed - unable to reach https://api.github.com"
    exit 1
else
    echo "OK: api.github.com reachable as expected"
fi

# Capture rule count BEFORE lockdown
RULE_COUNT=$(iptables -L OUTPUT -n | tail -n +3 | wc -l)

# ──────────────────────────────────────────────
# LOCKDOWN: Prevent firewall manipulation from within the container
# ──────────────────────────────────────────────
echo "Locking down firewall tools..."

# Remove execute permissions on all firewall-related binaries
for bin in iptables ip6tables iptables-save iptables-restore ip6tables-save ip6tables-restore \
           iptables-legacy ip6tables-legacy iptables-nft ip6tables-nft \
           ipset nft; do
    binary_path=$(which "$bin" 2>/dev/null || true)
    if [ -n "$binary_path" ]; then
        chmod 000 "$binary_path"
        echo "  Locked: $binary_path"
    fi
done

# Restrict sudo: replace blanket NOPASSWD:ALL with specific allowed commands
# Keep only what's needed for normal development workflow
echo "Restricting sudo access..."
rm -f /etc/sudoers.d/vscode
cat > /etc/sudoers.d/vscode-restricted << 'SUDOERS'
# Restricted sudo for devcontainer — no firewall manipulation allowed
# dbus-daemon: needed for draw.io / Electron
# mkdir: needed for /run/dbus
# docker/dockerd: needed for docker-in-docker feature
# chown: needed for Go module cache permissions
vscode ALL=(root) NOPASSWD: /usr/bin/dbus-daemon, /usr/bin/mkdir, /usr/bin/docker, /usr/bin/dockerd, /usr/bin/chown
SUDOERS
chmod 440 /etc/sudoers.d/vscode-restricted

# Remove the firewall-specific sudoers entry (no longer needed)
rm -f /etc/sudoers.d/firewall

echo "Sudo restricted to: dbus-daemon, mkdir, docker, dockerd, chown"

echo "=== Firewall setup complete ==="
echo "  - Rules active: $RULE_COUNT OUTPUT rules"
echo "  - Firewall tools: LOCKED"
echo "  - Sudo: RESTRICTED"
echo "  - Log file: $LOGFILE (via ulogd2)"
echo "  - View live: tail -f $LOGFILE"
