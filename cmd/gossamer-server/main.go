package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/egidinas/gossamer/internal/api"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8095", "HTTP listen address")
	root := flag.String("root", ".", "repository or fixture root")
	webDir := flag.String("web-dir", "", "optional built web/dist directory to serve with the API")
	allowRemote := flag.Bool("allow-remote", false, "allow binding command-authority demo endpoints to non-loopback interfaces")
	flag.Parse()
	if err := validateListenAddress(*addr, *allowRemote); err != nil {
		log.Fatal(err)
	}

	var handler *api.Server
	if *webDir != "" {
		handler = api.NewWithStatic(*root, *webDir)
		log.Printf("gossamer demo listening on http://%s with web assets from %s", *addr, *webDir)
	} else {
		handler = api.New(*root)
		log.Printf("gossamer API listening on http://%s", *addr)
	}
	defer func() {
		if err := handler.Close(); err != nil {
			log.Printf("close gossamer API: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown gossamer API: %v", err)
			_ = server.Close()
		}
	}
}

func validateListenAddress(addr string, allowRemote bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if strings.TrimSpace(host) == "" {
		if allowRemote {
			return nil
		}
		return fmt.Errorf("listen address %q binds all interfaces; pass -allow-remote to expose command-authority demo endpoints", addr)
	}
	if allowRemote {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("resolve listen host %q: %w", host, err)
		}
		for _, candidate := range ips {
			if !candidate.IsLoopback() {
				return fmt.Errorf("listen host %q resolves to non-loopback address %s; pass -allow-remote to expose command-authority demo endpoints", host, candidate)
			}
		}
		return nil
	}
	if !ip.IsLoopback() {
		return fmt.Errorf("listen address %q is not loopback; pass -allow-remote to expose command-authority demo endpoints", addr)
	}
	return nil
}
