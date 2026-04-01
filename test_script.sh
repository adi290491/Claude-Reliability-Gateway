
MSG="What is the ticket price to London for 5 persons?"

curl -X POST http://localhost:8090/message \
  -H "content-type: application/json" \
  -d "{
    \"message\": \"$MSG\"
  }"

# MESSAGE="{\"message\": \"$MSG\"}"
# echo "$MESSAGE"