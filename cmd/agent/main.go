package main

import (
	"context"
	"crypto/rsa"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/buildinfo"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	buildinfo.Print()

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

	var publicKey *rsa.PublicKey
	if cfg.CryptoKeyPath != "" {
		publicKey, err = crypto.LoadPublicKey(cfg.CryptoKeyPath)
		if err != nil {
			logger.Fatal("Failed to load public key", zap.Error(err))
		}
	}

	reporter := agent.NewReporter(cfg.ServerURL, cfg.SecretKey, publicKey)
	provider := agent.Provider{}

	agent := agent.NewAgent(cfg, &provider, reporter, logger)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
	)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := agent.Start(ctx); err != nil {
			logger.Error("Agent failed", zap.Error(err))
		}
	}()

	<-ctx.Done()

	logger.Info("Starting graceful shutdown...")

	<-done
	logger.Info("Shutdown complete")
}
