#!/bin/sh
# Show firewall log with resolved hostnames
# Usage: firewall-log        (show all entries)
#        firewall-log -f     (follow/live mode)
#        firewall-log -n 10  (last N entries)

LOGFILE="/var/log/firewall.log"

resolve_ip() {
    ip="$1"
    host=$(host "$ip" 2>/dev/null | awk '/domain name pointer/ {sub(/\.$/, "", $NF); print $NF; exit}')
    if [ -n "$host" ]; then
        echo "$host"
    else
        echo "$ip"
    fi
}

format_line() {
    line="$1"
    timestamp=$(echo "$line" | awk '{print $1, $2, $3}')
    dst=$(echo "$line" | grep -oP 'DST=\K[0-9.]+')
    dpt=$(echo "$line" | grep -oP 'DPT=\K[0-9]+')
    proto=$(echo "$line" | grep -oP 'PROTO=\K[A-Z]+')
    uid=$(echo "$line" | grep -oP 'UID=\K[0-9]+')

    if [ -n "$dst" ]; then
        hostname=$(resolve_ip "$dst")
        printf "%s  %-40s %-5s port %-5s (UID=%s)\n" "$timestamp" "$hostname" "$proto" "$dpt" "$uid"
    else
        echo "$line"
    fi
}

if [ ! -f "$LOGFILE" ]; then
    echo "No firewall log found at $LOGFILE"
    exit 1
fi

case "${1:-}" in
    -f)
        echo "Following firewall log (Ctrl+C to stop)..."
        echo ""
        tail -f "$LOGFILE" | while IFS= read -r line; do
            format_line "$line"
        done
        ;;
    -n)
        count="${2:-10}"
        tail -n "$count" "$LOGFILE" | while IFS= read -r line; do
            format_line "$line"
        done
        ;;
    -h|--help)
        echo "Usage: firewall-log [OPTIONS]"
        echo ""
        echo "Show blocked connections with resolved hostnames."
        echo ""
        echo "Options:"
        echo "  -f       Follow mode (like tail -f)"
        echo "  -n N     Show last N entries (default: 10)"
        echo "  -h       Show this help"
        ;;
    *)
        while IFS= read -r line; do
            format_line "$line"
        done < "$LOGFILE"
        ;;
esac
