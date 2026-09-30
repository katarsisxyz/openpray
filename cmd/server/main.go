// Command server runs the prayer times HTTP API.
//
//	PORT=8080 go run ./cmd/server
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // load time zones even on systems/images without tzdata (e.g. distroless)

	"github.com/katarsisxyz/openpray/internal/handler"
	"github.com/katarsisxyz/openpray/internal/repository"
	"github.com/katarsisxyz/openpray/internal/router"
	"github.com/katarsisxyz/openpray/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Wiring: repository -> service -> handler -> router.
	methodRepo := repository.NewMethodRepository()
	cityRepo, err := repository.NewCityRepository()
	if err != nil {
		slog.Error("load cities", "err", err)
		os.Exit(1)
	}
	prayerHandler := handler.NewPrayerHandler(service.NewPrayerService(methodRepo, cityRepo))
	cityHandler := handler.NewCityHandler(service.NewCityService(cityRepo))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router.New(prayerHandler, cityHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
}
