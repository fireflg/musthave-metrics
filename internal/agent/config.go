package agent

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	flagRunAddr        string
	flagReportInterval int
	flagPoolInterval   int
)

func Init() {
	parseAgentParams()
}

func RunAddr() string {
	if !strings.Contains(flagRunAddr, "http://") {
		flagRunAddr = "http://" + flagRunAddr
	}
	return flagRunAddr
}

func ReportInterval() int {
	return flagReportInterval
}

func PoolInterval() int {
	return flagPoolInterval
}

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
		ri, err := strconv.Atoi(reportInterval)
		if err != nil {
			fmt.Println("report interval must be a number!")
			os.Exit(1)
		}
		flagReportInterval = ri
	}

	if poolInterval == "" {
		flag.IntVar(&flagPoolInterval, "p", 2, "pool metrics interval")
	} else {
		pi, err := strconv.Atoi(poolInterval)
		if err != nil {
			fmt.Println("pool interval must be a number!")
			os.Exit(1)
		}
		flagPoolInterval = pi
	}

	flag.Parse()

	if unknownFlag := flag.Args(); len(unknownFlag) > 0 {
		fmt.Fprintf(os.Stderr, "unknown flag(s): %v\n", unknownFlag)
		os.Exit(2)
	}
}

func NewConfig() (*Config, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	return &Config{
		ServerURL:      RunAddr(),
		PollInterval:   time.Duration(PoolInterval()) * time.Second,
		ReportInterval: time.Duration(ReportInterval()) * time.Second,
		Logger:         logger.Sugar(),
	}, nil
}
