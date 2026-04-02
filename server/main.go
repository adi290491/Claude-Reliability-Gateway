package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/adi290491/Claude-Reliability-Gateway/server/config"
)

// func init() {
// 	setupLogging()
// }

// func setupLogging() {
// 	var logger *slog.Logger
// 	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
// 		Level: slog.LevelDebug,
// 	}))

// 	slog.SetDefault(logger)
// }

func main() {

	gateway := config.LoadConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /message", gateway.ServeHTTP)
	mux.HandleFunc("POST /debug/simulate-faulure", gateway.SimulateFailureHandler)
	mux.HandleFunc("/metrics", gateway.MetricHandler)

	s := &http.Server{
		Addr:         ":8090",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		Handler:      mux,
	}

	slog.Info("Server listening", "port", "8090")
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
