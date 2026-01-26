package main

import (
	"context"
	"log"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smctf/internal/cache"
	"smctf/internal/config"
	"smctf/internal/db"
	httpserver "smctf/internal/http"
	"smctf/internal/repo"
	"smctf/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	database, err := db.New(cfg.DB, cfg.AppEnv)
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}
	if err := database.PingContext(ctx); err != nil {
		log.Fatalf("db ping error: %v", err)
	}

	redisClient := cache.New(cfg.Redis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(ctx, database); err != nil {
			log.Fatalf("auto migrate error: %v", err)
		}
	}

	userRepo := repo.NewUserRepo(database)
	challengeRepo := repo.NewChallengeRepo(database)
	submissionRepo := repo.NewSubmissionRepo(database)

	authSvc := service.NewAuthService(cfg, userRepo, redisClient)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, redisClient)

	router := httpserver.NewRouter(cfg, authSvc, ctfSvc, userRepo, redisClient)

	srv := &nethttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := redisClient.Close(); err != nil {
		log.Printf("redis close error: %v", err)
	}
	if err := database.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}
}
