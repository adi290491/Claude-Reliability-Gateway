#!/bin/bash

BASE_URL="http://localhost:8090"
MSG="What is the ticket price to London for 5 persons?"

curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is the ticket price to London for 5 persons?"
  }' | jq .

echo ""
echo "=== Check circuit state ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== TEST 2: Set getTicketPrices to 100% failure ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
-H "Content-Type: application/json" \
-d '{
  "tool": "getTicketPrices",
  "failure_rate": 1.0
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
echo "=== Check after Call 4 — still OPEN ==="
curl -s "$BASE_URL/metrics" | jq .

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
echo "=== Wait 10 seconds for timeout to pass ==="
sleep 10

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
echo "=== TEST 4: Retry logic — set 50% failure rate ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
  -H "Content-Type: application/json" \
  -d '{"tool": "getTicketPrices", "failure_rate": 0.5}' | jq .

echo ""
echo "=== Retry Call 1 — may succeed after retries ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to London?"}' | jq .

echo ""
echo "=== Retry Call 2 — observe retry logs in server output ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Paris?"}' | jq .

echo ""
echo "=== Restore to normal ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
  -H "Content-Type: application/json" \
  -d '{"tool": "getTicketPrices", "failure_rate": 0.0}' | jq .

echo ""
echo "=== TEST 5: Circuit opens after retries exhausted ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
  -H "Content-Type: application/json" \
  -d '{"tool": "getTicketPrices", "failure_rate": 1.0}' | jq .

echo ""
echo "=== Exhausted Call 1 — 3 retries all fail ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Tokyo?"}' | jq .

echo ""
echo "=== Exhausted Call 2 — 3 retries all fail ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Berlin?"}' | jq .

echo ""
echo "=== Exhausted Call 3 — circuit should open after this ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the price of a ticket to Paris?"}' | jq .

echo ""
echo "=== Check state — should show OPEN ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== TEST 6: Multi-tool request — both tools called ==="
curl -s -X POST "$BASE_URL/debug/simulate-failure" \
  -H "Content-Type: application/json" \
  -d '{"tool": "getTicketPrices", "failure_rate": 0.0}' | jq .

echo ""
echo "=== Wait 10s for circuit to recover ==="
sleep 10

echo ""
echo "=== Multi-tool call — ticket price AND calculation ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the ticket price to London for 3 persons? Calculate the total cost."}' | jq .

echo ""
echo "=== TEST 7: Unknown tool falls back to default circuit breaker ==="
curl -s -X POST "$BASE_URL/message" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the current weather in Chicago?"}' | jq .

echo ""
echo "=== Check metrics — default CB should appear ==="
curl -s "$BASE_URL/metrics" | jq .

echo ""
echo "=== ALL TESTS COMPLETE ==="

