package observer_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"
)

func ExampleFileObserver() {
	file, err := os.CreateTemp("", "audit-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())
	file.Close()

	obs, err := observer.NewFileObserver(file.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer obs.Close()

	ctx := observer.WithClientIP(context.Background(), "192.168.1.100")

	obs.Notify(ctx, "metric_counter")

	obs.NotifyBatch(ctx, []string{"metric_gauge1", "metric_gauge2"})

	data, err := os.ReadFile(file.Name())
	if err != nil {
		log.Fatal(err)
	}
	_ = data
	fmt.Println("audit log written successfully")
	// Output: audit log written successfully
}

func ExampleNewObservers() {
	observers, err := observer.NewObservers("", "")
	if err != nil {
		log.Fatal(err)
	}

	if observers == nil {
		fmt.Println("no observers created")
	}

	// Output:
	// no observers created
}

func ExampleClientIP() {
	ctx := context.Background()
	fmt.Printf("empty context: %q\n", observer.ClientIP(ctx))

	ctx = observer.WithClientIP(context.Background(), "10.0.0.1")
	fmt.Printf("with IP: %q\n", observer.ClientIP(ctx))

	// Output:
	// empty context: ""
	// with IP: "10.0.0.1"
}