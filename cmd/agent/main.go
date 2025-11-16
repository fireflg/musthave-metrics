package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var flagRunAddr string
var flagReportInterval int
var flagPoolInterval int

func parseAgentParams() {
	address := os.Getenv("ADDRESS")
	reportInterval := os.Getenv("REPORT_INTERVAL")
	poolInterval := os.Getenv("POOL_INTERVAL")
	if address == "" {
		flag.StringVar(&flagRunAddr, "a", "http://localhost:8080", "address and port to run server")
	} else {
		flagRunAddr = address
	}
	if reportInterval == "" {
		flag.IntVar(&flagReportInterval, "r", 10, "report metrics interval")
	} else {
		reportInterval, err := strconv.Atoi(reportInterval)
		if err != nil {
			fmt.Println("report interval must be a number!")
			os.Exit(1)
		}
		flagReportInterval = reportInterval
	}
	if os.Getenv("POOL_INTERVAL") == "" {
		flag.IntVar(&flagReportInterval, "r", 10, "report metrics interval")
	} else {
		poolInterval, err := strconv.Atoi(poolInterval)
		if err != nil {
			fmt.Println("report interval must be a number!")
			os.Exit(1)
		}
		flagPoolInterval = poolInterval
		if unknownFlag := flag.Args(); len(unknownFlag) > 0 {
			fmt.Fprintf(os.Stderr, "unknown flag(s): %v\n", unknownFlag)
			os.Exit(2)
		}
		flag.Parse()
	}
}

func main() {
	parseAgentParams()

	fmt.Println("Send metrics to server", flagRunAddr)
	fmt.Println("Pool metrics interval", flagPoolInterval)
	fmt.Println("Report metrics interval", flagReportInterval)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	if !strings.Contains(flagRunAddr, "http://") {
		flagRunAddr = "http://" + flagRunAddr
	}
	agentService := agent.NewAgentService(client, flagRunAddr, flagPoolInterval, flagReportInterval)
	agentService.Start(context.Background())
}
