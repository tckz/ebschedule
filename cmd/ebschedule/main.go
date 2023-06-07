package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/tckz/ebschedule"
)

var version = "dev"
var myName = filepath.Base(os.Args[0])

func main() {
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return fmt.Errorf("config.LoadDefaultConfig: %w", err)
	}

	return ebschedule.NewCommand(
		myName, version,
		scheduler.NewFromConfig(cfg),
		os.Stdout,
	).ExecuteContext(ctx)
}
