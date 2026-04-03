package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/adi290491/Claude-Reliability-Gateway/server/circuitbreaker"
	t "github.com/adi290491/Claude-Reliability-Gateway/server/tools"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/avast/retry-go"
	"github.com/sony/gobreaker/v2"
)

type UserResponse struct {
	Response string `json:"response"`
}

type GatewayConfig struct {
	// call anthropic
	client anthropic.Client
	// logger
	logger *slog.Logger
	// circuit breaker
	circuitBreakers       map[string]*gobreaker.CircuitBreaker[any]
	defaultCircuitBreaker *gobreaker.CircuitBreaker[any]
}

func NewGateway(anthropicClient anthropic.Client, logger *slog.Logger, toolNames []string) *GatewayConfig {
	return &GatewayConfig{
		client: anthropicClient,
		logger: logger,
		circuitBreakers: func() map[string]*gobreaker.CircuitBreaker[any] {

			cbMap := make(map[string]*gobreaker.CircuitBreaker[any])
			for _, name := range toolNames {
				cbMap[name] = circuitbreaker.CreateCircuitBreaker(name)
			}
			return cbMap
		}(),
		defaultCircuitBreaker: circuitbreaker.CreateCircuitBreaker("default"),
	}
}

func (g *GatewayConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.logger.Info("Serving user messages...")

	var userMessage struct {
		UserMessage string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userMessage); err != nil {
		slog.Error("Failed to decode user message request", "error", err)
		RespondWithError(w, err, http.StatusInternalServerError)
		return
	}

	tools := t.CreateToolParams()

	g.logger.Info("user message received", "message", userMessage.UserMessage)

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage.UserMessage)),
	}

	userResponse := UserResponse{}

	for {
		response, err := g.client.Messages.New(r.Context(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_5,
			MaxTokens: 1024,
			Messages:  messages,
			Tools:     tools,
		})

		if err != nil {
			slog.Error("LLM Error", "error", err)
			RespondWithError(w, fmt.Errorf("received error from LLM: %v", err), http.StatusInternalServerError)
			return
		}

		messages = append(messages, response.ToParam())
		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, block := range response.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.ToolUseBlock:

				g.logger.Info("tool invoked", "tool", block.Name)

				cb, exist := g.circuitBreakers[block.Name]

				if !exist {
					cb = g.defaultCircuitBreaker
				}

				start := time.Now()
				var cbResult any
				retryErr := retry.Do(
					func() error {
						cbResult, err = cb.Execute(func() (any, error) {
							return t.HandleToolUse(block, variant)
						})
						return err
					},
					retry.Attempts(3),
					retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
					retry.Delay(100*time.Millisecond),
					retry.MaxJitter(50*time.Millisecond),
					retry.RetryIf(func(err error) bool {
						return err != gobreaker.ErrOpenState
					}),
					retry.OnRetry(func(n uint, err error) {
						g.logger.Warn("retrying tool execution",
							"tool", block.Name,
							"attempt", n+1,
							"error", err,
						)
					}),
				)

				duration := time.Since(start)

				if retryErr != nil {
					g.logger.Error("tool execution failed after retries",
						"tool", block.Name,
						"duration in ms", duration.Milliseconds(), "error", retryErr)

					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, fmt.Sprintf("Tool %s is temporarily unavailable: %v", block.Name, err), true))
					continue
				}

				g.logger.Info("tool executed successfully", "tool", block.Name, "duration in ms", duration.Milliseconds())

				b, err := json.Marshal(cbResult)
				if err != nil {
					slog.Error("Failed to decode user message request", "error", err)
					RespondWithError(w, err, http.StatusInternalServerError)
					return
				}

				g.logger.Debug("tool result", "tool", block.Name, "result", string(b))

				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, string(b), false))
			}

		}

		if len(toolResults) == 0 {
			userResponse.Response = response.Content[0].Text
			break
		}

		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	WriteJSON(w, &userResponse, http.StatusOK)
}

func (g *GatewayConfig) SimulateFailureHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tool        string  `json:"tool"`
		FailureRate float64 `json:"failure_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, err, http.StatusBadRequest)
		return
	}

	t.SetFailureRate(req.Tool, req.FailureRate)

	g.logger.Info("failure simulation updated",
		"tool", req.Tool,
		"failure_rate", req.FailureRate,
	)

	WriteJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (g *GatewayConfig) MetricHandler(w http.ResponseWriter, r *http.Request) {
	metrics := make(map[string]any)

	for toolName, cb := range g.circuitBreakers {
		metrics[toolName] = map[string]any{
			"circuit_state": cb.State().String(),
			"failure_rate":  t.FailureSimulation[toolName],
		}
	}

	WriteJSON(w, metrics, http.StatusOK)
}

type ErrorResponse struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func RespondWithError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := ErrorResponse{
		Message:    err.Error(),
		StatusCode: statusCode,
	}

	json.NewEncoder(w).Encode(errResp)
}

func WriteJSON(w http.ResponseWriter, data interface{}, statusCode int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}
