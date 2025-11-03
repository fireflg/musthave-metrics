package main

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"net/http"
	"time"
)

func main() {
	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       5 * time.Second,
	}
	backendURL := "http://localhost:8080"
	agentService := agent.NewAgentService(client, backendURL)
	agentService.Start()
}
