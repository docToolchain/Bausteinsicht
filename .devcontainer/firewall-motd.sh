#!/bin/sh
# Firewall status message — shown on every terminal login
# Sourced via /etc/zsh/zshrc and /etc/profile.d/

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  FIREWALL ACTIVE — outbound restricted to whitelist     ║"
echo "║                                                         ║"
echo "║  iptables/ipset: LOCKED (not modifiable)                ║"
echo "║                                                         ║"
echo "║  Blocked connections are logged to the host kernel log. ║"
echo "║  Read on host:  dmesg | grep FW-BLOCKED                ║"
echo "║  Or live:       dmesg -wT | grep FW-BLOCKED            ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
