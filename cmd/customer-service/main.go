package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Archiit19/customer-service-go/internal/config"
	"github.com/Archiit19/customer-service-go/internal/customer"
	dbpkg "github.com/Archiit19/customer-service-go/internal/db"
	httph "github.com/Archiit19/customer-service-go/internal/http"
	"github.com/Archiit19/customer-service-go/internal/logger"
)

func main() {
	ctx := context.Background()
	cfg, warnings := config.Load()
	logg, err := logger.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logg.Sync()
	}()
	for _, msg := range warnings {
		logg.Warn(ctx, "configuration warning", logger.String("detail", msg))
	}
	logg.Info(ctx, "configuration loaded", logger.String("port", cfg.AppPort), logger.String("log_level", cfg.LogLevel))
	pool, err := dbpkg.NewPool(ctx, cfg, logg)
	if err != nil {
		logg.Error(ctx, "database pool initialization failed", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()
	logg.Info(ctx, "database pool initialized")
	repo := customer.NewPGRepository(pool, logg)
	svc := customer.NewService(repo, logg)
	router := httph.NewRouter(svc, logg)
	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logg.Info(ctx, "server listening", logger.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Error(ctx, "server listen failed", logger.Err(err))
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logg.Info(ctx, "shutdown signal received")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logg.Error(ctxShutdown, "server shutdown error", logger.Err(err))
	} else {
		logg.Info(ctxShutdown, "server shutdown complete")
	}
}
