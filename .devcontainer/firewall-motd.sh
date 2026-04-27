#!/bin/sh
# Firewall status message — shown on every terminal login
# Sourced via /etc/zsh/zshrc and /etc/profile.d/

if [ -f /var/log/firewall.log ]; then
    blocked_count=$(wc -l < /var/log/firewall.log 2>/dev/null || echo "0")

    echo ""
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║  FIREWALL ACTIVE — outbound restricted to whitelist     ║"
    echo "║                                                         ║"
    echo "║  Blocked:      ${blocked_count} connection(s) so far"
    echo "║  Show log:     firewall-log                             ║"
    echo "║  Follow live:  firewall-log -f                          ║"
    echo "║  iptables:     LOCKED (not modifiable)                  ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo ""
fi
