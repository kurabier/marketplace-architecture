// Catalog Service — сервис управления товарным каталогом маркетплейса.
//
// На этом этапе (ДЗ: C4 + инициализация сервисов) бизнес-логики нет:
// сервис только поднимается и отдаёт health-check.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const serviceName = "catalog-service"

func main() {
	// Флаг -healthcheck нужен для HEALTHCHECK в Docker: в distroless-образе
	// нет curl/wget, поэтому бинарник сам умеет проверить себя.
	healthcheck := flag.Bool("healthcheck", false, "probe /health of a running instance and exit")
	flag.Parse()

	addr := ":" + envOr("PORT", "8080")

	if *healthcheck {
		os.Exit(probe("http://localhost" + addr + "/health"))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	srv := &http.Server{
		Addr:              addr,
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting", "service", serviceName, "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown: docker stop шлёт SIGTERM и ждёт 10 секунд.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("stopped", "service", serviceName)
}

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	return mux
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok", Service: serviceName})
}

func probe(url string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
