package main

import (
	"context"
	"fmt"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var buildVersion string
var buildDate string
var buildCommit string

func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}
	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

func main() {
	printBuildInfo()

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
