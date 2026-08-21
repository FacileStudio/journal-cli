#!/usr/bin/env bash
# End-to-end smoke test: run the real `journal` binary against a mock that
# mimics Journal's auth + read API, covering the whole login→read→logout loop.
#
# The mock is a checked-in Go file over in ./scripts/mockmain.go, built once
# here — no nested `go run`, no readiness race.

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

BIN="$(mktemp -d)/journal"
go build -o "$BIN" .

# Isolate config so the test never touches a real login.
export XDG_CONFIG_HOME="$(mktemp -d)"
export JOURNAL_URL=""
export NO_COLOR=1

PORT=4987
MOCKPID=""
trap '[ -n "$MOCKPID" ] && kill "$MOCKPID"' EXIT

# Build the mock once, serve it in the background.
MOCKBIN="$(mktemp -d)/mock"
(cd ./scripts/mock && go build -o "$MOCKBIN" .)
"$MOCKBIN" &
MOCKPID=$!

URL="http://127.0.0.1:$PORT"
# Wait until the mock answers; a readiness loop beats any fixed sleep.
for _ in $(seq 1 40); do
  if curl -fsS "$URL/api/auth/config" >/dev/null 2>&1; then break; fi
  sleep 0.25
done

fail() { echo "FAIL: $1" >&2; exit 1; }

# 1. login writes a config
OUT="$("$BIN" --url "$URL" login --email a@b.c --password secret 2>&1)"
echo "$OUT" | grep -q "signed in" || fail "login output: $OUT"
[ -f "$XDG_CONFIG_HOME/journal/config.yml" ] || fail "no config written"

# 2. logs without login args now works (token from file)
OUT="$("$BIN" --url "$URL" logs 2>&1)"
echo "$OUT" | grep -q "upload failed" || fail "logs text: $OUT"

# 3. logs --json is valid and carries ids
OUT="$("$BIN" --url "$URL" logs --json 2>&1)"
echo "$OUT" | grep -q '"id": 3' || fail "logs json ids: $OUT"

# 4. filter is sent to the server (mock rejects unknown level)
OUT="$("$BIN" --url "$URL" logs --level error 2>&1)"
echo "$OUT" | grep -q "upload failed" || fail "logs filter: $OUT"

# 5. apps table
OUT="$("$BIN" --url "$URL" apps 2>&1)"
echo "$OUT" | grep -q "sablier" || fail "apps: $OUT"

# 6. context accepts an id
OUT="$("$BIN" --url "$URL" context 3 2>&1)"
echo "$OUT" | grep -q "upload failed" || fail "context: $OUT"

# 7. logout clears the token, then a read fails as it should
"$BIN" --url "$URL" logout >/dev/null 2>&1
if "$BIN" --url "$URL" logs >/dev/null 2>&1; then
  fail "read after logout should fail"
fi

echo "ALL E2E CHECKS PASSED"