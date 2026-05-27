#!/usr/bin/env bash
# v-star API example via cURL.
# Requires: curl, jq (optional, for pretty-printing)
# Usage: ./curl.sh
# Requires v-star server running on http://localhost:8080.

BASE="http://localhost:8080"
QXS=$(python3 -c "import json; print(json.dumps([0.001]*111))")

echo "=== Present Value ==="
curl -s -X POST "$BASE/value" \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}' | jq .

echo ""
echo "=== Monte Carlo ==="
curl -s -X POST "$BASE/simulate" \
  -H "Content-Type: application/json" \
  -d '{"num_paths":10000,"steps":10,"initial_rate":0.05,"drift":0.02,"volatility":0.15}' | jq .

echo ""
echo "=== Annuity ==="
curl -s -X POST "$BASE/annuity" \
  -H "Content-Type: application/json" \
  -d "{\"interest_rate\":0.05,\"qxs\":$QXS,\"age\":30,\"amount\":1000,\"computation\":\"whole_life_immediate\"}" | jq .

echo ""
echo "=== Reserve ==="
curl -s -X POST "$BASE/reserve" \
  -H "Content-Type: application/json" \
  -d "{\"interest_rate\":0.05,\"qxs\":$QXS,\"age\":30,\"term\":20,\"sum_assured\":100000,\"method\":\"net_premium\"}" | jq .
