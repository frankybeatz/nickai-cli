// Command nickai-node runs the Nick Node server — an always-on gRPC service
// for persistent strategy execution, price streaming, backtest offloading,
// and alert dispatch.
//
// Usage:
//
//	nickai-node [flags]
//	  --addr   listen address (default "127.0.0.1:9400")
//	  --debug  enable debug logging to ~/.nickai/debug.log
//	  --token  shared auth token (or NICKAI_NODE_TOKEN env var)
//
// Build:
//
//	go build -o nickai-node ./cmd/node/
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/node"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:9400", "listen address (host:port)")
	debug := flag.Bool("debug", false, "enable debug logging")
	token := flag.String("token", "", "shared auth token (or NICKAI_NODE_TOKEN env var)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("nickai-node v%s\n", version)
		return
	}

	logging.Init(*debug)
	node.Version = version
	if *token == "" {
		*token = os.Getenv("NICKAI_NODE_TOKEN")
	}
	if !isLoopbackAddr(*addr) && *token == "" {
		fmt.Fprintln(os.Stderr, "Error: --token (or NICKAI_NODE_TOKEN) is required for non-loopback bind addresses")
		os.Exit(1)
	}

	srv := node.NewServer(*token)

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

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
