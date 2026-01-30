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

func NewRouter(cfg config.Config, authSvc *service.AuthService, ctfSvc *service.CTFService, userRepo *repo.UserRepo, teamSvc *service.TeamService, redis *redis.Client, logger *logging.Logger) *gin.Engine {
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
	r.Use(middleware.CORS(cfg.AppEnv != "production", nil))

	h := handlers.New(cfg, authSvc, ctfSvc, userRepo, teamSvc, redis)

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
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
		auth.GET("/me/solved", h.MeSolved)
		auth.GET("/me/solved/team", h.MeSolvedTeam)
		auth.PUT("/me", h.UpdateMe)
		auth.POST("/challenges/:id/submit", h.SubmitFlag)

		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.JWT), middleware.RequireRole("admin"))
		admin.POST("/challenges", h.CreateChallenge)
		admin.PUT("/challenges/:id", h.UpdateChallenge)
		admin.DELETE("/challenges/:id", h.DeleteChallenge)
		admin.POST("/registration-keys", h.CreateRegistrationKeys)
		admin.GET("/registration-keys", h.ListRegistrationKeys)
		admin.POST("/teams", h.CreateTeam)
	}

	attachFrontendRoutes(r)

	return r
}
