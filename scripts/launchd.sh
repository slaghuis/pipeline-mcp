#!/usr/bin/env bash
# Run pipeline-mcp as a per-user launchd agent.
#
# Usage: scripts/launchd.sh {install|uninstall|restart|reload|status|logs|plist}
#
#   install    build the binary, write the LaunchAgent plist, and start it
#   uninstall  stop the agent and remove the plist (binary and logs are kept)
#   restart    restart the running agent
#   reload     send SIGHUP so pipeline-mcp re-reads services.yaml
#   status     show launchd state and check the /metrics endpoint
#   logs       follow the log file
#   plist      print the plist that install would write
#
# Overrides:
#   PIPELINE_MCP_CONFIG  config file  (default ~/.config/ai-factory/pipeline-mcp.yaml)
#   PIPELINE_MCP_BIN     binary path  (default ~/.local/bin/pipeline-mcp)
#   PIPELINE_MCP_ENV     env file sourced on every start, for secrets such as
#                        REGISTRY_PASSWORD or KUBECONFIG
#                        (default ~/.config/ai-factory/pipeline-mcp.env)
#
# launchd does not load your shell profile, so the PATH of the shell running
# `install` is baked into the plist. Re-run install after changing PATH.
set -euo pipefail

LABEL="com.slaghuis.pipeline-mcp"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="${PIPELINE_MCP_CONFIG:-$HOME/.config/ai-factory/pipeline-mcp.yaml}"
BIN="${PIPELINE_MCP_BIN:-$HOME/.local/bin/pipeline-mcp}"
ENV_FILE="${PIPELINE_MCP_ENV:-$HOME/.config/ai-factory/pipeline-mcp.env}"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"
LOG="$HOME/Library/Logs/pipeline-mcp.log"
DOMAIN="gui/$(id -u)"
SERVICE="$DOMAIN/$LABEL"

die() { echo "error: $*" >&2; exit 1; }

xml_escape() { sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' <<<"$1"; }

is_loaded() { launchctl print "$SERVICE" >/dev/null 2>&1; }

listen_port() {
	local port=""
	[ -f "$CONFIG" ] && port=$(grep -E '^listen:' "$CONFIG" | grep -oE '[0-9]+' | tail -1 || true)
	echo "${port:-8766}"
}

healthy() { curl -fs -o /dev/null --max-time 2 "http://127.0.0.1:$(listen_port)/metrics"; }

print_plist() {
	cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>$LABEL</string>
	<key>ProgramArguments</key>
	<array>
		<string>/bin/sh</string>
		<string>-c</string>
		<string>if [ -f "\$0" ]; then set -a; . "\$0"; set +a; fi; exec "\$1" -config "\$2"</string>
		<string>$(xml_escape "$ENV_FILE")</string>
		<string>$(xml_escape "$BIN")</string>
		<string>$(xml_escape "$CONFIG")</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>$(xml_escape "$PATH")</string>
	</dict>
	<key>WorkingDirectory</key>
	<string>$(xml_escape "$HOME")</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ThrottleInterval</key>
	<integer>10</integer>
	<key>StandardOutPath</key>
	<string>$(xml_escape "$LOG")</string>
	<key>StandardErrorPath</key>
	<string>$(xml_escape "$LOG")</string>
</dict>
</plist>
EOF
}

stop_agent() {
	is_loaded || return 0
	launchctl bootout "$SERVICE" 2>/dev/null || true
	for _ in $(seq 1 20); do
		is_loaded || return 0
		sleep 0.5
	done
	die "timed out waiting for $LABEL to stop"
}

wait_healthy() {
	for _ in $(seq 1 20); do
		healthy && return 0
		sleep 0.5
	done
	echo "pipeline-mcp did not answer on port $(listen_port). Last log lines:" >&2
	tail -n 20 "$LOG" >&2 || true
	return 1
}

cmd_install() {
	[ -f "$CONFIG" ] || die "config not found: $CONFIG (see README step 5, or set PIPELINE_MCP_CONFIG)"
	command -v go >/dev/null || die "go not found in PATH"

	echo "Building $BIN"
	mkdir -p "$(dirname "$BIN")" "$(dirname "$PLIST")" "$(dirname "$LOG")"
	(cd "$REPO_DIR" && go build -o "$BIN" ./cmd/pipeline-mcp)

	stop_agent
	local port
	port=$(listen_port)
	if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
		die "port $port is already in use; stop the manually started pipeline-mcp first"
	fi

	print_plist >"$PLIST"
	plutil -lint "$PLIST" >/dev/null
	launchctl bootstrap "$DOMAIN" "$PLIST"
	echo "Installed $PLIST"

	wait_healthy
	echo "pipeline-mcp is running on port $port (logs: $LOG)"
	[ -f "$ENV_FILE" ] || echo "note: no env file at $ENV_FILE; create it to pass secrets to pipelines"
}

cmd_uninstall() {
	stop_agent
	rm -f "$PLIST"
	echo "Removed $LABEL (binary $BIN and log $LOG were kept)"
}

cmd_restart() {
	is_loaded || die "$LABEL is not loaded; run install first"
	launchctl kickstart -k "$SERVICE"
	wait_healthy
	echo "Restarted $LABEL"
}

cmd_reload() {
	is_loaded || die "$LABEL is not loaded; run install first"
	launchctl kill HUP "$SERVICE"
	echo "Sent SIGHUP to $LABEL; services.yaml reloaded"
}

cmd_status() {
	if ! is_loaded; then
		echo "$LABEL is not loaded"
		exit 1
	fi
	launchctl print "$SERVICE" | grep -E $'^\t(state|pid|last exit code|runs) ='
	if healthy; then
		echo "health = ok (port $(listen_port))"
	else
		echo "health = not responding (port $(listen_port))"
		exit 1
	fi
}

case "${1:-}" in
install) cmd_install ;;
uninstall) cmd_uninstall ;;
restart) cmd_restart ;;
reload) cmd_reload ;;
status) cmd_status ;;
logs) touch "$LOG" && tail -f "$LOG" ;;
plist) print_plist ;;
*)
	sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'
	exit 2
	;;
esac
