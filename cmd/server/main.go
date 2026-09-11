// Command server runs the 5/3/1 tracker.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	// Embedding the timezone database keeps dates correct in a scratch image,
	// which has no /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/server"
)

// version is stamped at build time and used to bust static asset caches.
var version = "dev"

// healthcheck probes the running server from inside the container. The
// scratch image has no shell and no curl, so the binary doubles as its own
// health probe for the compose healthcheck.
func healthcheck() int {
	addr := envOr("ADDR", ":3000")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1" + announceAddr(addr) + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	password := os.Getenv("AUTH_PASSWORD")
	if password == "" {
		return errors.New("AUTH_PASSWORD is required; set it in the environment or compose file")
	}

	dataDir := envOr("DATA_DIR", "/data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir %s: %w", dataDir, err)
	}
	dbPath := filepath.Join(dataDir, "531.db")

	conn, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Info("database ready", "path", dbPath)

	addr := envOr("ADDR", ":3000")
	srv := &http.Server{
		Addr:              announceAddr(addr),
		Handler:           server.New(conn, log, server.Config{Password: password, Version: version}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-stop:
		log.Info("shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func announceAddr(addr string) string {
	// Accept either ":3000" or a bare port number.
	if _, err := strconv.Atoi(addr); err == nil {
		return ":" + addr
	}
	return addr
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
