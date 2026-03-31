package main

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smctf/internal/bootstrap"
	"smctf/internal/cache"
	"smctf/internal/config"
	"smctf/internal/db"
	httpserver "smctf/internal/http"
	"smctf/internal/logging"
	"smctf/internal/realtime"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/stack"
	"smctf/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		boot := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		boot.Error("config error", slog.Any("error", err))
		os.Exit(1)
	}

	logger, err := logging.New(cfg.Logging, logging.Options{
		Service:   "smctf",
		Env:       cfg.AppEnv,
		AddSource: false,
	})
	if err != nil {
		boot := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		boot.Error("logging init error", slog.Any("error", err))
		os.Exit(1)
	}

	slog.SetDefault(logger.Logger)

	defer func() {
		if err := logger.Close(); err != nil {
			logger.Error("log close error", slog.Any("error", err))
		}
	}()

	logger.Info("config loaded", slog.Any("config", config.FormatForLog(cfg)))

	ctx := context.Background()
	database, err := db.New(cfg.DB, cfg.AppEnv)
	if err != nil {
		logger.Error("db init error", slog.Any("error", err))
		os.Exit(1)
	}

	if err := database.PingContext(ctx); err != nil {
		logger.Error("db ping error", slog.Any("error", err))
		os.Exit(1)
	}

	redisClient := cache.New(cfg.Redis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("redis ping error", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(ctx, database); err != nil {
			logger.Error("auto migrate error", slog.Any("error", err))
			os.Exit(1)
		}
	}

	userRepo := repo.NewUserRepo(database)
	divisionRepo := repo.NewDivisionRepo(database)
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
			logger.Error("s3 init error", slog.Any("error", err))
			os.Exit(1)
		}
		fileStore = store
	}

	authSvc := service.NewAuthService(cfg, database, userRepo, registrationKeyRepo, teamRepo, redisClient)
	userSvc := service.NewUserService(userRepo, teamRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	divisionSvc := service.NewDivisionService(divisionRepo)
	teamSvc := service.NewTeamService(teamRepo, divisionRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, redisClient, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, redisClient, cfg.Cache.AppConfigTTL)

	var stackClient stack.API
	var stackClientCloser func() error
	if cfg.Stack.ProvisionerUseGRPC {
		client, err := stack.NewGRPCClient(cfg.Stack.ProvisionerGRPCAddr, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
		if err != nil {
			logger.Error("grpc stack client init error", slog.Any("error", err))
			os.Exit(1)
		}

		stackClient = client
		stackClientCloser = client.Close
	} else {
		stackClient = stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	}
	if stackClientCloser != nil {
		defer func() {
			if err := stackClientCloser(); err != nil {
				logger.Warn("stack client close error", slog.Any("error", err))
			}
		}()
	}

	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, stackClient, redisClient)

	bootstrap.BootstrapAdmin(ctx, cfg, database, userRepo, teamRepo, divisionRepo, logger)

	if cfg, _, _, err := appConfigSvc.Get(ctx); err != nil {
		logger.Warn("app config load warning", slog.Any("error", err))
	} else if cfg.CTFStartAt == "" && cfg.CTFEndAt == "" {
		logger.Warn("ctf window not configured; competition always active")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sseHub := realtime.NewSSEHub()
	leaderboardBus := realtime.NewScoreboardBus(redisClient, cfg, scoreSvc, divisionSvc, logger, sseHub)
	leaderboardBus.Start(ctx)

	router := httpserver.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userSvc, scoreSvc, divisionSvc, teamSvc, stackSvc, redisClient, logger, sseHub)
	srv := &nethttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.Any("error", err))
	}

	if err := redisClient.Close(); err != nil {
		logger.Error("redis close error", slog.Any("error", err))
	}

	if err := database.Close(); err != nil {
		logger.Error("db close error", slog.Any("error", err))
	}
}
