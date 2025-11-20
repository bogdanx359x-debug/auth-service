package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"authservice/internal/config"
	"authservice/internal/db"
	"authservice/internal/server"
	"authservice/internal/user"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	if err := user.EnsureSchema(ctx, pool); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	userRepo := user.NewRepository(pool)
	authService := user.NewService(cfg.JWTSecret, cfg.TokenTTL, userRepo)

	srv := server.New(cfg, authService)

	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
