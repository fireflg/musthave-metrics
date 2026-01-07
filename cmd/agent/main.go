package main

import (
	"context"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger := l.Sugar()
	defer logger.Sync()

	cfg, err := agent.LoadAgentConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	reporter := agent.NewReporter(cfg.ServerURL, cfg.SecretKey)
	provider := agent.Provider{}

	agent := agent.NewAgent(cfg, &provider, reporter, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := agent.Start(ctx); err != nil {
			logger.Error("Agent failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Starting graceful shutdown...")
	cancel()

	time.Sleep(1 * time.Second)
	logger.Info("Shutdown complete")
}
