package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Archiit19/customer-service-go/internal/config"
	"github.com/Archiit19/customer-service-go/internal/customer"
	dbpkg "github.com/Archiit19/customer-service-go/internal/db"
	httph "github.com/Archiit19/customer-service-go/internal/http"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := dbpkg.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	repo := customer.NewPGRepository(pool)

	svc := customer.NewService(repo)

	router := httph.NewRouter(svc)

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("Customer Service listening on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped gracefully")
}
