package http

import (
	"io"
	nethttp "net/http"
	"os"

	"smctf/internal/config"
	"smctf/internal/http/handlers"
	"smctf/internal/http/middleware"
	"smctf/internal/logging"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, authSvc *service.AuthService, ctfSvc *service.CTFService, appConfigSvc *service.AppConfigService, userRepo *repo.UserRepo, scoreRepo *repo.ScoreboardRepo, teamSvc *service.TeamService, stackSvc *service.StackService, redis *redis.Client, logger *logging.Logger) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	if logger != nil {
		gin.DefaultWriter = io.MultiWriter(os.Stdout, logger)
		gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, logger)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(cfg.Logging, logger))
	r.Use(middleware.CORS(cfg.AppEnv != "production", cfg.CORS.AllowedOrigins))

	h := handlers.New(cfg, authSvc, ctfSvc, appConfigSvc, userRepo, scoreRepo, teamSvc, stackSvc, redis)

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.GET("/config", h.GetConfig)

		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/logout", h.Logout)

		api.GET("/challenges", h.ListChallenges)
		api.GET("/leaderboard", h.Leaderboard)
		api.GET("/leaderboard/teams", h.TeamLeaderboard)
		api.GET("/timeline", h.Timeline)
		api.GET("/timeline/teams", h.TeamTimeline)
		api.GET("/teams", h.ListTeams)
		api.GET("/teams/:id", h.GetTeam)
		api.GET("/teams/:id/members", h.ListTeamMembers)
		api.GET("/teams/:id/solved", h.ListTeamSolved)
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id", h.GetUser)
		api.GET("/users/:id/solved", h.GetUserSolved)

		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWT))
		auth.GET("/me", h.Me)
		auth.PUT("/me", h.UpdateMe)
		auth.POST("/challenges/:id/submit", h.SubmitFlag)
		auth.POST("/challenges/:id/file/download", h.RequestChallengeFileDownload)
		auth.GET("/stacks", h.ListStacks)
		auth.POST("/challenges/:id/stack", h.CreateStack)
		auth.GET("/challenges/:id/stack", h.GetStack)
		auth.DELETE("/challenges/:id/stack", h.DeleteStack)

		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.JWT), middleware.RequireRole("admin"))
		admin.PUT("/config", h.AdminUpdateConfig)
		admin.POST("/challenges", h.CreateChallenge)
		admin.GET("/challenges/:id", h.AdminGetChallenge)
		admin.PUT("/challenges/:id", h.UpdateChallenge)
		admin.DELETE("/challenges/:id", h.DeleteChallenge)
		admin.POST("/challenges/:id/file/upload", h.RequestChallengeFileUpload)
		admin.DELETE("/challenges/:id/file", h.DeleteChallengeFile)
		admin.POST("/registration-keys", h.CreateRegistrationKeys)
		admin.GET("/registration-keys", h.ListRegistrationKeys)
		admin.POST("/teams", h.CreateTeam)
	}

	return r
}
