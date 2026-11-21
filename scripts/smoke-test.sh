#!/usr/bin/env bash
# Smoke test: creates a link, reads it back, and checks the main error cases.
# Usage: ./scripts/smoke-test.sh [API_URL]   (default http://localhost:8080)
set -u

API="${1:-http://localhost:8080}"
CODE="smk$(date +%s | tail -c 6)"   # unique 8-char code per run
URL="https://example.com/smoke"
PASS=0; FAIL=0

check() { # name, expected, actual
  if [ "$2" = "$3" ]; then echo "PASS  $1 ($3)"; PASS=$((PASS+1));
  else echo "FAIL  $1: expected $2, got $3"; FAIL=$((FAIL+1)); fi
}

create() { # url, code -> prints "status body"
  curl -s -o /tmp/smoke_body -w '%{http_code}' -X POST "$API/links" \
    -H 'Content-Type: application/json' \
    -H "X-Forwarded-For: 10.0.0.$((RANDOM % 250 + 1))" \
    -d "{\"url\":\"$1\",\"code\":\"$2\"}"
}

echo "Testing $API with code $CODE"

# Writes
check "create new link"            201 "$(create "$URL" "$CODE")"
check "shortUrl returned"          "{\"shortUrl\":\"t/$CODE\"}" "$(tr -d ' \n' < /tmp/smoke_body)"
check "same url + code again"      200 "$(create "$URL" "$CODE")"
check "code taken by other url"    409 "$(create "https://other.com" "$CODE")"
check "invalid code"               400 "$(create "$URL" "ab")"
check "invalid url"                400 "$(create "ftp://x.com" "${CODE}x")"

# Reads
read_status=$(curl -s -o /dev/null -w '%{http_code}' "$API/$CODE")
location=$(curl -s -o /dev/null -w '%{redirect_url}' "$API/$CODE")
upper=$(echo "$CODE" | tr '[:lower:]' '[:upper:]')
check "redirect status"            301 "$read_status"
check "redirect location"          "$URL" "$location"
check "redirect is case-insensitive" 301 "$(curl -s -o /dev/null -w '%{http_code}' "$API/$upper")"
check "unknown code"               404 "$(curl -s -o /dev/null -w '%{http_code}' "$API/nosuchcode")"

rm -f /tmp/smoke_body
echo "Done: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
