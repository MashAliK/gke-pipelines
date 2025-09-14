package main

import (
	"fmt"
	"os"
	"context"
	"os/signal"
	"syscall"
	"errors"

	"github.com/MashAliK/gke-pipelines/internal/cli"

	"k8s.io/klog/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		signal.Stop(make(chan os.Signal))
		cancel()
		klog.Flush()
		fmt.Fprintf(os.Stderr, "\nReceived signal, shutting down gracefully... (press Ctrl+C again to force)\n")
	}()

	if err := cli.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
		}
		
		if errors.Is(err, context.Canceled) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
