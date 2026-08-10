package main

import (
	"context"
	"crypto/rsa"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/buildinfo"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
	"io"
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
		logger.Fatalw("Failed to load config", "error", err)
	}

	var publicKey *rsa.PublicKey
	if cfg.CryptoKeyPath != "" {
		publicKey, err = crypto.LoadPublicKey(cfg.CryptoKeyPath)
		if err != nil {
			logger.Fatalw("Failed to load public key", "error", err)
		}
	}

	reporter, err := newReporter(cfg, publicKey)
	if err != nil {
		logger.Fatalw("Failed to initialize reporter", "error", err)
	}
	if closer, ok := reporter.(io.Closer); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				logger.Errorw("Failed to close reporter", "error", err)
			}
		}()
	}

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
			logger.Errorw("Agent failed", "error", err)
		}
	}()

	<-ctx.Done()

	logger.Info("Starting graceful shutdown...")

	<-done
	logger.Info("Shutdown complete")
}

// newReporter выбирает транспорт отправки метрик: gRPC, если задан адрес
// gRPC-сервера, иначе - HTTP.
func newReporter(cfg *agent.Config, publicKey *rsa.PublicKey) (agent.MetricsReporter, error) {
	if cfg.GRPCAddr != "" {
		return agent.NewGRPCReporter(cfg.GRPCAddr)
	}
	return agent.NewReporter(cfg.ServerURL, cfg.SecretKey, publicKey), nil
}
