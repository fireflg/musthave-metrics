package main

import (
	"flag"
	"fmt"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"net/http"
	"os"
	"time"
)

var flagRunAddr string
var flagReportInterval int
var flagPoolInterval int

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "http://localhost:8080", "address and port to run server")
	flag.IntVar(&flagPoolInterval, "p", 2, "pool metrics interval")
	flag.IntVar(&flagReportInterval, "r", 10, "report metrics interval")
	if unknownFlag := flag.Args(); len(unknownFlag) > 0 {
		fmt.Fprintf(os.Stderr, "unknown flag(s): %v\n", unknownFlag)
		os.Exit(2)
	}
	flag.Parse()
}

func main() {
	parseFlags()

	fmt.Println("Send metrics to server", flagRunAddr)
	fmt.Println("Pool metrics interval", flagPoolInterval)
	fmt.Println("Report metrics interval", flagReportInterval)

	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       5 * time.Second,
	}
	agentService := agent.NewAgentService(client, "http://"+flagRunAddr, flagPoolInterval, flagReportInterval)
	agentService.Start()
}
