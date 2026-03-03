// Command nickai-node runs the Nick Node server — an always-on gRPC service
// for persistent strategy execution, price streaming, backtest offloading,
// and alert dispatch.
//
// Usage:
//
//	nickai-node [flags]
//	  --addr   listen address (default "0.0.0.0:9400")
//	  --debug  enable debug logging to ~/.nickai/debug.log
//
// Build:
//
//	go build -o nickai-node ./cmd/node/
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/node"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	addr := flag.String("addr", "0.0.0.0:9400", "listen address (host:port)")
	debug := flag.Bool("debug", false, "enable debug logging")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("nickai-node v%s\n", version)
		return
	}

	logging.Init(*debug)
	node.Version = version

	srv := node.NewServer()

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logging.Info("received signal, shutting down", "signal", sig)
		srv.Stop()
	}()

	fmt.Printf("Nick Node v%s starting on %s\n", version, *addr)
	if err := srv.Start(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
