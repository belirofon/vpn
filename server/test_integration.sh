#!/usr/bin/env bash
set -e
[ -n "$DEBUG" ] && set -x
CURL="curl -sS --max-time 5"
PASS=0; FAIL=0
BASE="http://localhost:8080"

ok()   { echo -e "  \033[32mPASS: $1\033[0m"; PASS=$((PASS+1)); }
fail() { echo -e "  \033[31mFAIL: $1\033[0m"; FAIL=$((FAIL+1)); }

chk()   { local b; b=$($CURL "$1"); echo "$b" | grep -q "$2" && ok "$3" || fail "$3"; }
nchk()  { local b; b=$($CURL "$1"); ! echo "$b" | grep -q "$2" && ok "$3" || fail "$3"; }
code()  { local c; c=$($CURL -o /dev/null -w '%{http_code}' "$1"); [ "$c" = "$2" ] && ok "$3" || fail "$3 ($c)"; }
postcode() { local c; c=$($CURL -X POST -o /dev/null -w '%{http_code}' "$1"); [ "$c" = "$2" ] && ok "$3" || fail "$3 ($c)"; }
postchk()  { local b; b=$($CURL -X POST "$1"); echo "$b" | grep -q "$2" && ok "$3" || fail "$3"; }

cleanup() { kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true; rm -f server; }
trap cleanup EXIT

echo "===== VPN Server Integration Tests ====="
go build -o server ./cmd/server/... && echo "  Build OK" || { echo "  Build FAILED"; exit 1; }

# ── Test with mock configs ──
MOCK_CONFIGS=true ./server &
PID=$!; sleep 1

echo ""
echo "--- 1. Health ---"
code  "$BASE/health"         200 "GET /health → 200"
chk   "$BASE/health"         '"status":"ok"'                   "health response has status=ok"

echo "--- 2. Configs (RU filtered out) ---"
chk   "$BASE/api/configs"    '"configs"'                       "configs response has configs array"
chk   "$BASE/api/configs"    '"total":3'                       "total=3 (4 mock - 1 RU)"
nchk  "$BASE/api/configs"    '"country":"RU"'                  "no config with country=RU"
nchk  "$BASE/api/configs"    'ru-1'                            "no config from ru-1.example.com"

echo "--- 3. Best config ---"
chk   "$BASE/api/best-config" '"config"'                        "best-config has config object"
chk   "$BASE/api/best-config" '"country":"DE"'                  "fastest is DE (45ms), not RU (filtered)"
chk   "$BASE/api/best-config" '"latency_ms":45'                 "fastest latency is 45ms"

echo "--- 4. Refresh ---"
postcode "$BASE/api/refresh"  200 "POST /api/refresh → 200"
postchk  "$BASE/api/refresh" '"refreshing"'                     "refresh returns refreshing status"

# ── Test without subscription (no mock, no URL) ──
kill $PID; wait $PID 2>/dev/null || true; sleep 1
# Skip .env loading to simulate clean start
SKIP_DOTENV=true ./server &
PID=$!; sleep 2

echo "--- 5. No configs → 503 ---"
code  "$BASE/api/best-config" 503 "no configs → 503"

echo ""
echo "===== $PASS passed, $FAIL failed ====="
[ "$FAIL" -eq 0 ] && echo -e "\033[32m  ALL TESTS PASSED!\033[0m" || exit 1
