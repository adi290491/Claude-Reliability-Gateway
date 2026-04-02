#!/bin/bash

BASE_URL="http://localhost:8090"
MSG="What is the ticket price to London for 5 persons?"

curl -s -X POST "$BASEURL/message" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is the ticket price to London for 5 persons?"
  }' | jq .

echo ""
echo "=== Check circuit state ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== TEST 2: Set getTicketPrices to 100% failure ==="
curl -s -X POST "$BASE_URL/debug/simulate-faulure" \
-H "Content-Type : application/json" \
-d '{
  "tool" : "getTicketPrices"
  "failure_rate" : 1.0
}' | jq .

echo ""
echo "=== Call 1 — should fail, circuit still CLOSED ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Paris?"}' | jq .

echo ""
echo "=== Call 2 — should fail, circuit still CLOSED ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Tokyo?"}' | jq .

echo ""
echo "=== Call 3 — should fail, circuit OPENS after this ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Berlin?"}' | jq .

echo ""
echo "=== Check circuit state — should show OPEN ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== Call 4 — circuit OPEN, tool call blocked, fallback returned ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to London?"}' | jq .

echo ""
echo "=== TEST 3: Calculator still works despite ticket prices being down ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is 42 multiplied by 7?"}' | jq .

echo ""
echo "=== Restore ticket prices to normal ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
  -H "Content-Type: application/json" \
  -d '{"tool": "getTicketPrices", "failure_rate": 0.0}'

echo ""
echo "=== Wait 30 seconds for circuit to move to HALF-OPEN ==="
echo "=== (or reduce Timeout to 5s in your CB config for faster testing) ==="
sleep 30

echo ""
echo "=== Check circuit state — should show HALF-OPEN ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== Call in HALF-OPEN — should succeed and close circuit ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to London?"}' | jq .

echo ""
echo "=== Check circuit state — should show CLOSED again ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== ALL TESTS COMPLETE ==="
# MESSAGE="{\"message\": \"$MSG\"}"
# echo "$MESSAGE"