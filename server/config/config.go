package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/adi290491/Claude-Reliability-Gateway/server/gateway"
	"github.com/adi290491/Claude-Reliability-Gateway/server/tools"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

func LoadConfig() *gateway.GatewayConfig {

	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", "error", err)
		os.Exit(1)
	}

	anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY")

	if err := validateAPIKey(anthropicAPIKey); err != nil {
		slog.Error("ANTHROPIC_API_KEY validation error:", "error", err)
		os.Exit(1)
	}
	client := anthropic.NewClient(
		option.WithAPIKey(anthropicAPIKey),
	)

	logger := setupGatewayLogging()

	toolNames := tools.GetToolParamNames()

	gateway := gateway.NewGateway(client, logger, toolNames)

	return gateway
}

func setupGatewayLogging() *slog.Logger {
	var logger *slog.Logger
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	slog.SetDefault(logger)
	return logger
}

func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("no API key was found - please head over to the troubleshooting notebook in this folder to identify & fix!")
	}

	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return fmt.Errorf("an API key was found, but it doesn't start sk-proj-; please check you're using the right key - see troubleshooting notebook")
	}

	if strings.TrimSpace(apiKey) != apiKey {
		return fmt.Errorf("an API key was found, but it looks like it might have space or tab characters at the start or end - please remove them - see troubleshooting notebook")
	}

	slog.Info("API key found and looks good so far!")
	return nil
}
