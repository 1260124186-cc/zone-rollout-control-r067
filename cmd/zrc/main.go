package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solo-0014-zone-rollout-control/internal/allocation"
	"solo-0014-zone-rollout-control/internal/assessment"
	"solo-0014-zone-rollout-control/internal/fleet"
	"solo-0014-zone-rollout-control/internal/httpapi"
	"solo-0014-zone-rollout-control/internal/plan"
	"solo-0014-zone-rollout-control/internal/platform"
	"solo-0014-zone-rollout-control/internal/probe"
	"solo-0014-zone-rollout-control/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		address        string
		snapshotPath   string
		shutdownWindow time.Duration
	)
	flag.StringVar(&address, "addr", "127.0.0.1:8090", "HTTP listen address")
	flag.StringVar(&snapshotPath, "state", "data/zrc-state.json", "state snapshot path")
	flag.DurationVar(&shutdownWindow, "shutdown-timeout", 10*time.Second, "graceful shutdown window")
	flag.Parse()
	if shutdownWindow <= 0 {
		return errors.New("shutdown-timeout must be positive")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	lock, err := platform.LockFile(snapshotPath + ".lock")
	if err != nil {
		return fmt.Errorf("lock snapshot: %w", err)
	}
	defer lock.Unlock()

	stateStore, err := store.Open(snapshotPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	clock := platform.SystemClock{}
	ids := platform.CryptoIDGenerator{}
	plans := plan.New(stateStore, clock, ids)
	nodes := fleet.New(stateStore, clock, ids)
	allocations := allocation.New(stateStore, clock, ids)
	probes := probe.New(stateStore, clock, ids, allocations)
	assessments := assessment.New(stateStore, clock, ids)
	api := httpapi.New(plans, nodes, allocations, probes, assessments, ids, logger)

	server := &http.Server{
		Addr:              address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	rootContext, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go expireFunction(rootContext, allocations, logger)

	serverResult := make(chan error, 1)
	go func() {
		logger.Info("server listening", "address", address, "state_path", snapshotPath)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverResult <- err
			return
		}
		serverResult <- nil
	}()

	select {
	case err := <-serverResult:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-rootContext.Done():
		logger.Info("shutdown requested")
	}

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), shutdownWindow)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverResult; err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	logger.Info("server stopped")
	return nil
}

func expireFunction(ctx context.Context, service *allocation.Service, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := service.ExpireDue(ctx)
			if err != nil {
				logger.Error("lease expiry sweep failed", "error", err)
				continue
			}
			if count > 0 {
				logger.Info("expired leases", "count", count)
			}
		}
	}
}
