# Claude-Reliability-Gateway

An HTTP middleware written in Go that makes AI agent tool use production-ready. Sits transparently between your application and the Anthropic API, adding circuit breakers, retry with exponential backoff and jitter, per-tool failure isolation, and structured observability — without changing your application code.

---

## Problem

Production AI agents break when tools fail. A web search API times out, an external pricing service errors, a database query fails — without reliability infrastructure, a single tool failure cascades into a broken user experience. The agent loop stalls, errors propagate to users, and there is no visibility into what went wrong or why.

This is not an edge case. In production systems, dependencies fail constantly. The question is whether your infrastructure handles failures gracefully or exposes them directly to users.

---

## Solution

The gateway intercepts every tool call in the agent loop and applies reliability patterns transparently:

- **Circuit Breaker** — stops calling a failing tool after a threshold, preventing wasted retries against a dependency that is clearly down
- **Retry with Exponential Backoff and Jitter** — handles transient failures automatically before escalating to the circuit breaker
- **Per-Tool Failure Isolation** — each tool has its own independent circuit breaker, so a failing web search does not affect a working calculator
- **Graceful Degradation** — when tools fail beyond retry thresholds, the agent receives a meaningful fallback response rather than an error, keeping the loop running
- **Structured Observability** — every tool call logged with tool name, duration, attempt count, and circuit breaker state

---

## Architecture

```
Your Application
       │
       ▼
┌──────────────────────────────────────────┐
│           Reliability Gateway            │
│                                          │
│  ┌─────────────┐    ┌─────────────────┐  │
│  │  Agent Loop │───▶│  Tool Executor  │  │
│  │             │    │                 │  │
│  │  message    │    │  ┌───────────┐  │  │
│  │  history    │    │  │  Retry    │  │  │
│  │  management │    │  │  Logic    │  │  │
│  └─────────────┘    │  └─────┬─────┘  │  │
│                     │        │        │  │
│                     │  ┌─────▼─────┐  │  │
│                     │  │  Circuit  │  │  │
│                     │  │  Breaker  │  │  │
│                     │  └─────┬─────┘  │  │
│                     │        │        │  │
│                     │  ┌─────▼─────┐  │  │
│                     │  │   Tool    │  │  │
│                     │  │ Function  │  │  │
│                     │  └───────────┘  │  │
│                     └─────────────────┘  │
└──────────────────────────────────────────┘
       │
       ▼
  Anthropic API
```

**Request flow:**

1. Application sends a message to the gateway
2. Gateway forwards to Anthropic API with tool definitions
3. Claude decides which tool to call and returns a `tool_use` block
4. Gateway intercepts the tool call and runs it through the reliability stack:
   - Check circuit breaker state — reject immediately if open
   - Execute tool with retry logic — up to 3 attempts with backoff and jitter
   - Record success or failure — update circuit breaker state
   - Return result or fallback to Claude
5. Claude receives the tool result and continues the agent loop
6. Final response returned to application

---

## Reliability Patterns

### Circuit Breaker

Each tool has its own independent circuit breaker with three states:

```
           3 consecutive failures
CLOSED ──────────────────────────▶ OPEN
  ▲                                  │
  │         Recovery succeeds        │  5 second timeout
  │                                  ▼
CLOSED ◀────────────────────── HALF-OPEN
           (1 test request allowed)
```

**Why per-tool isolation?**

Tool failures are independent. A network-dependent external API has completely different reliability characteristics than a local arithmetic function. A shared circuit breaker would couple unrelated components — one failing tool would block all tools. Per-tool circuit breakers ensure failure containment.

**Observed behavior from server logs:**

```
// Circuit opens after 3 consecutive failures
level=WARN  msg="retrying tool execution"      tool=getTicketPrices attempt=1
level=WARN  msg="retrying tool execution"      tool=getTicketPrices attempt=2
level=INFO  msg="circuit breaker state changed" FROM=closed TO=open
level=WARN  msg="retrying tool execution"      tool=getTicketPrices attempt=3
level=ERROR msg="tool execution failed"        error="All attempts fail: #1 #2 #3"

// Subsequent calls blocked immediately (0ms — no tool execution attempted)
level=ERROR msg="tool execution failed" duration_ms=0 error="circuit breaker is open"

// Recovery cycle
level=INFO  msg="circuit breaker state changed" FROM=open TO=half-open
level=INFO  msg="circuit breaker state changed" FROM=half-open TO=closed
level=INFO  msg="tool executed successfully"    tool=getTicketPrices
```

### Retry with Exponential Backoff and Jitter

Transient failures — network blips, momentary timeouts — are handled transparently before the circuit breaker tracks them.

**Backoff formula:** `delay = base_delay * 2^attempt + random_jitter`

**Why jitter?** Without randomization, concurrent requests retry at identical intervals, creating a thundering herd that amplifies the original failure. Jitter spreads retry load across time.

**Retry does not retry open circuit errors.** When the circuit is open, retrying immediately is wasteful — the dependency is known to be failing. The gateway detects `ErrOpenState` and skips retry attempts entirely.

**Observed behavior:**

```
// Retry succeeds on third attempt (50% failure rate)
level=WARN msg="retrying tool execution" tool=getTicketPrices attempt=1
level=WARN msg="retrying tool execution" tool=getTicketPrices attempt=2
level=INFO msg="tool executed successfully" tool=getTicketPrices duration_ms=146

// Circuit open — retry skipped, fails immediately
level=ERROR msg="tool execution failed" error="All attempts fail:\n#1: circuit breaker is open"
```

### Graceful Degradation

When a tool fails beyond retry thresholds, the gateway returns a meaningful error message to Claude rather than propagating an exception. Claude receives the failure context and responds accordingly — informing the user of the temporary unavailability rather than producing an unhandled error.

This keeps the agent loop alive even under partial tool failure.

### Observability

Every tool call emits structured log entries:

```
level=INFO  msg="tool invoked"               tool=getTicketPrices
level=INFO  msg="tool called"                tool=getTicketPrices city=London
level=INFO  msg="tool executed successfully" tool=getTicketPrices duration_ms=0
level=DEBUG msg="tool result"                tool=getTicketPrices result="..."
```

A `/metrics` endpoint provides real-time circuit breaker state per tool:

```json
{
  "getTicketPrices": {
    "circuit_state": "open",
    "failure_rate": 1
  },
  "calculateEquation": {
    "circuit_state": "closed",
    "failure_rate": 0
  }
}
```

---

## Design Decisions

### Reliability layer in the gateway, not in tools

Tool functions contain only execution logic. They have no knowledge of circuit breakers, retry counts, or fallback behavior. This separation means:

- New tools can be added without thinking about reliability infrastructure
- Reliability behavior can be changed without touching tool implementations
- Tools remain testable in isolation

### Build vs buy for circuit breaker

Used `sony/gobreaker` rather than a custom implementation. Reliability primitives should be battle-tested rather than home-grown — subtle bugs in circuit breaker state machines create exactly the production failures they are designed to prevent. gobreaker is the most widely adopted Go circuit breaker library with extensive production usage.

### Per-tool circuit breaker configuration

Circuit breaker settings are currently uniform across tools. In a production deployment, each tool would have tuned thresholds based on its reliability characteristics — an external API with known instability would have a lower failure threshold than a local computation function.

### LLM provider agnostic pattern

The gateway pattern is not specific to Anthropic. Swapping the Anthropic SDK for OpenAI, Gemini, or any other provider requires changing only the client initialization and response parsing — the reliability infrastructure applies unchanged.

---

## Demonstration

### Test 1 — Normal operation

```bash
curl -X POST http://localhost:8090/message \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the ticket price to London for 5 persons?"}'

# Response
{
  "response": "The total ticket price for 5 persons to London would be $3,995."
}
```

### Test 2 — Circuit breaker lifecycle

```bash
# Set tool to 100% failure
curl -X POST http://localhost:8090/debug/simulate-failure \
  -d '{"tool": "getTicketPrices", "failure_rate": 1.0}'

# After 3 failed requests — circuit opens
curl http://localhost:8090/metrics
{
  "getTicketPrices": { "circuit_state": "open" },
  "calculateEquation": { "circuit_state": "closed" }
}

# Calculator unaffected by ticket price failure
curl -X POST http://localhost:8090/message \
  -d '{"message": "What is 42 multiplied by 7?"}'
{
  "response": "42 multiplied by 7 equals 294."
}
```

### Test 3 — Retry with partial failure

```bash
# Set 50% failure rate
curl -X POST http://localhost:8090/debug/simulate-failure \
  -d '{"tool": "getTicketPrices", "failure_rate": 0.5}'

# Request succeeds after retries — visible in server logs
level=WARN msg="retrying tool execution" tool=getTicketPrices attempt=1
level=WARN msg="retrying tool execution" tool=getTicketPrices attempt=2
level=INFO msg="tool executed successfully" tool=getTicketPrices duration_ms=146
```

---

## Getting Started

**Prerequisites:** Go 1.21+, Anthropic API key

```bash
git clone https://github.com/adi290491/Claude-Reliability-Gateway
cd Claude-Reliability-Gateway

# Create .env file
echo "ANTHROPIC_API_KEY=sk-ant-..." > .env

# Build and run
go build -o server ./server && ./server/server
```

**Send a message:**

```bash
curl -X POST http://localhost:8090/message \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the ticket price to Tokyo?"}'
```

**Available endpoints:**

| Endpoint | Method | Description |
|---|---|---|
| `/message` | POST | Send a message through the agent loop |
| `/metrics` | GET | Circuit breaker state per tool |
| `/debug/simulate-failure` | POST | Set failure rate for a tool (testing) |

---

## Project Structure

```
server/
├── main.go                    # Server setup and routing
├── config/
│   └── config.go              # API key validation, client initialization
├── gateway/
│   └── gateway.go             # Agent loop, reliability orchestration
├── circuitbreaker/
│   └── circuit_breaker.go     # Circuit breaker configuration per tool
└── tools/
    ├── tool_params.go          # Tool definitions (schema for Claude)
    └── tools.go                # Tool implementations + failure simulation
```

---

## Tech Stack

- **Go** — net/http for HTTP server, slog for structured logging
- **Anthropic Go SDK** — Claude API integration and tool use
- **sony/gobreaker** — Circuit breaker implementation
- **avast/retry-go** — Retry with exponential backoff and jitter
- **joho/godotenv** — Environment configuration

---

## Future Work

- **SSE stream parsing** — Intercept tool calls from the response stream rather than buffered responses, enabling lower latency and mid-stream reliability handling
- **Concurrent tool execution** — Execute multiple tool calls in parallel using goroutines when Claude requests several tools in one response
- **Per-tool rate limiting** — Token bucket rate limiter per tool type to prevent quota exhaustion on external APIs
- **MCP server integration** — Extend reliability patterns to Model Context Protocol servers for standardized tool infrastructure
- **Configurable thresholds** — Per-tool circuit breaker and retry configuration via config file or environment variables

---

## Related Projects

- [**Systemic — Distributed Session Analytics Platform**](https://github.com/adi290491/productivity-planner) — Cloud-native Go microservices platform with custom rate limiter, service-to-service authentication, and API gateway
- [**Semantic Search Cache Gateway**](https://github.com/adi290491/go-semantic-cache) — LLM API cost optimization middleware using Redis HNSW vector search, reducing API costs by 25% through semantic caching
