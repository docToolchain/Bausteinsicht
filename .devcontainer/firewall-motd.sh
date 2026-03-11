#!/bin/sh
# Firewall status message — shown on every terminal login
# Sourced from /etc/profile.d/ by zsh/bash

if [ -f /var/log/firewall.log ]; then
    blocked_count=$(wc -l < /var/log/firewall.log 2>/dev/null || echo "0")
    last_blocked=$(tail -1 /var/log/firewall.log 2>/dev/null || echo "none")

    echo ""
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║  FIREWALL ACTIVE — outbound restricted to whitelist     ║"
    echo "║                                                         ║"
    echo "║  Log file:  /var/log/firewall.log                       ║"
    echo "║  Blocked:   ${blocked_count} connection(s) so far"
    echo "║  Last:      ${last_blocked:-none}"
    echo "║                                                         ║"
    echo "║  View live: tail -f /var/log/firewall.log               ║"
    echo "║  iptables/ipset: LOCKED (not modifiable)                ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo ""
fi
