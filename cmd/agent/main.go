package main

import (
	"context"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
)

func main() {
	ctx := context.Background()
	agent.Init()

	cfg, err := agent.NewConfig()
	if err != nil {
		panic(err)
	}

	service := agent.NewAgent(cfg)
	service.Start(ctx)
}
