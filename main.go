package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hnatekmarorg/lmproxy/config"
	"github.com/hnatekmarorg/lmproxy/proxy"
)

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	} else {
		slog.Error("Please provide path to config.yaml")
		os.Exit(1)
	}

	var err error
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Couldn't load config", "error", err)
		os.Exit(1)
	}

	proxyServer := proxy.NewProxy(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyServer.Handler)

	host := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("Proxy server starting", "host", host)

	server := &http.Server{
		Addr:    host,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		slog.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
