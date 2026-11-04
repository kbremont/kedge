#!/usr/bin/env bash
# Kedge Dev on a phone: the app of its own that `make dev` and `make
# dev-install` run, beside the production one. See mobile/README.md.
#
#   scripts/dev.sh run       the demo server and Metro, from this workspace
#   scripts/dev.sh install   build Kedge Dev and install it on a phone
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
metro_port=8081
server_port=3000
password=dev
# Outside the repo, so every workspace can see what the phone is running.
installed_file="$HOME/.config/kedge/dev-install-fingerprint"

export APP_VARIANT=development

# A new workspace has no dependencies, and a merge can change them.
install_deps() {
  cd "$root/mobile"
  if [ ! -f node_modules/.package-lock.json ] || [ package-lock.json -nt node_modules/.package-lock.json ]; then
    echo "==> Installing the app's dependencies"
    npm ci --no-audit --no-fund
  fi
}

fingerprint() {
  (cd "$root/mobile" && npx --no -- expo-updates fingerprint:generate --platform ios 2>/dev/null) |
    node -pe 'JSON.parse(require("fs").readFileSync(0)).hash'
}

# An earlier session's server left on a port would answer in place of this
# workspace's, and the phone would show old code with no sign of why.
free_port() {
  local pids
  pids=$(lsof -nP -iTCP:"$1" -sTCP:LISTEN -t 2>/dev/null || true)
  [ -z "$pids" ] && return 0
  echo "==> Stopping what's on port $1"
  # shellcheck disable=SC2086
  kill $pids 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    lsof -nP -iTCP:"$1" -sTCP:LISTEN -t >/dev/null 2>&1 || return 0
    sleep 0.5
  done
  echo "Port $1 is still in use. Stop what's using it and try again." >&2
  exit 1
}

# Metro doesn't check that the JavaScript it serves fits the native code on the
# phone, so a mismatch would otherwise show up as a crash or a missing module.
check_installed() {
  local current
  if [ ! -f "$installed_file" ]; then
    echo "!! Kedge Dev isn't installed yet. Plug in your phone and run: make dev-install"
    return 0
  fi
  current=$(fingerprint)
  if [ "$current" != "$(cat "$installed_file")" ]; then
    echo "!! This workspace's native code differs from the Kedge Dev on your phone."
    echo "!! Plug in your phone and run: make dev-install"
  fi
}

run() {
  install_deps
  free_port "$metro_port"
  free_port "$server_port"
  check_installed

  echo "==> Starting the demo server"
  local log
  log="${TMPDIR:-/tmp}/kedge-dev-server.log"
  cd "$root/server"
  # Built directly: make demo runs code generation first, which is slow and can
  # leave stray changes in a workspace.
  go build -o bin/kedge-dev ./cmd/server
  AUTH_PASSWORD=$password DATA_DIR=./data-demo ADDR=":$server_port" ./bin/kedge-dev -seed-demo >"$log" 2>&1 &
  local server=$!
  trap 'kill "$server" 2>/dev/null || true' EXIT

  local up=""
  for _ in $(seq 1 60); do
    if curl -fsS -o /dev/null "http://localhost:$server_port/api/v1/info" 2>/dev/null; then
      up=yes
      break
    fi
    kill -0 "$server" 2>/dev/null || break
    sleep 0.5
  done
  if [ -z "$up" ]; then
    echo "The demo server didn't start:" >&2
    tail -20 "$log" >&2
    exit 1
  fi

  cat <<EOF

==> Ready. On your phone, on the same Wi-Fi as this Mac:
    1. Open Kedge Dev, and pick this Mac's server if it asks.
    2. If it asks to connect:
         Server address  http://$(scutil --get LocalHostName).local:$server_port
         Password        $password
    Changes you save here appear on the phone. Press Ctrl-C to stop.

EOF

  cd "$root/mobile"
  # CI=false keeps Metro watching for changes when there's no terminal, as when
  # a Claude session starts this in the background.
  CI=false npx --no -- expo start --dev-client --port "$metro_port"
}

install() {
  install_deps
  cd "$root/mobile"

  # CocoaPods fails on a non-UTF-8 locale, such as a shell that didn't start
  # from a terminal.
  export LC_ALL=en_US.UTF-8 LANG=en_US.UTF-8

  # ios/ is generated, and may be from another variant. Only it is replaced, so
  # uncommitted work elsewhere doesn't matter.
  echo "==> Generating the Kedge Dev project"
  EXPO_NO_GIT_STATUS=1 npx --no -- expo prebuild --clean -p ios

  echo "==> Building and installing Kedge Dev"
  local device=(--device)
  # A name or UDID, to skip the prompt when there's more than one device.
  [ -n "${DEVICE:-}" ] && device+=("$DEVICE")
  npx --no -- expo run:ios "${device[@]}" --no-bundler

  mkdir -p "$(dirname "$installed_file")"
  fingerprint >"$installed_file"
  echo "==> Kedge Dev is installed. Next: make dev"
}

case "${1:-}" in
  run) run ;;
  install) install ;;
  *)
    echo "usage: $0 run|install" >&2
    exit 2
    ;;
esac
