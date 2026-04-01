package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	t "github.com/adi290491/Claude-Reliability-Gateway/server/tools"
	"github.com/anthropics/anthropic-sdk-go"
)

type GatewayConfig struct {
	// call anthropic
	client anthropic.Client
	// logger
	logger *slog.Logger
	// circuit breaker
	// call tools
}

func NewGateway(anthropicClient anthropic.Client, logger *slog.Logger) *GatewayConfig {
	return &GatewayConfig{
		client: anthropicClient,
		logger: logger,
	}
}

func (g *GatewayConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.logger.Info("Serving user messages...")

	var userMessage struct {
		UserMessage string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userMessage); err != nil {
		slog.Error("Failed to decode user message request", "error", err)
		return
	}

	tools := t.CreateToolParams()

	println(color("[user]: ") + userMessage.UserMessage)

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage.UserMessage)),
	}

	for {
		response, err := g.client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_5,
			MaxTokens: 1024,
			Messages:  messages,
			Tools:     tools,
		})

		if err != nil {
			panic(err)
		}

		messages = append(messages, response.ToParam())
		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, block := range response.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.ToolUseBlock:
				print(color("[user (" + block.Name + ")]: "))

				result, err := t.HandleToolUse(block, variant)
				if err != nil {
					slog.Error("Error while executing tool", "error", err)
					return
				}

				b, err := json.Marshal(result)
				if err != nil {
					panic(err)
				}

				println(string(b))

				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, string(b), false))
			}

		}

		if len(toolResults) == 0 {
			break
		}

		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	fmt.Printf("%+v", messages)
}

func color(s string) string {
	return fmt.Sprintf("\033[1;%sm%s\033[0m", "33", s)
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
