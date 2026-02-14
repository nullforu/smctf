package main

import (
	"context"
	"io"
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
	"smctf/internal/logging"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/stack"
	"smctf/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		log.Fatalf("logging init error: %v", err)
	}

	defer func() {
		if err := logger.Close(); err != nil {
			log.Printf("log close error: %v", err)
		}
	}()

	log.SetOutput(io.MultiWriter(os.Stdout, logger))
	log.Printf("config loaded:\n%s", config.FormatForLog(cfg))

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
	teamRepo := repo.NewTeamRepo(database)
	registrationKeyRepo := repo.NewRegistrationKeyRepo(database)
	challengeRepo := repo.NewChallengeRepo(database)
	submissionRepo := repo.NewSubmissionRepo(database)
	scoreRepo := repo.NewScoreboardRepo(database)
	appConfigRepo := repo.NewAppConfigRepo(database)
	stackRepo := repo.NewStackRepo(database)

	var fileStore storage.ChallengeFileStore
	if cfg.S3.Enabled {
		store, err := storage.NewS3ChallengeFileStore(ctx, cfg.S3)
		if err != nil {
			log.Fatalf("s3 init error: %v", err)
		}
		fileStore = store
	}

	authSvc := service.NewAuthService(cfg, database, userRepo, registrationKeyRepo, teamRepo, redisClient)
	teamSvc := service.NewTeamService(teamRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, redisClient, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo)
	stackClient := stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, stackClient, redisClient)

	if cfg, _, _, err := appConfigSvc.Get(ctx); err != nil {
		log.Printf("app config load warning: %v", err)
	} else if cfg.CTFStartAt == "" && cfg.CTFEndAt == "" {
		log.Printf("warning: ctf_start_at/ctf_end_at not configured; competition will always be active")
	}

	router := httpserver.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userRepo, scoreRepo, teamSvc, stackSvc, redisClient, logger)
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
