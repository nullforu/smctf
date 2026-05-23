package http

import (
	nethttp "net/http"

	"smctf/internal/config"
	"smctf/internal/http/handlers"
	"smctf/internal/http/middleware"
	"smctf/internal/logging"
	"smctf/internal/models"
	"smctf/internal/realtime"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, authSvc *service.AuthService, ctfSvc *service.CTFService, appConfigSvc *service.AppConfigService, userSvc *service.UserService, scoreSvc *service.ScoreboardService, divisionSvc *service.DivisionService, teamSvc *service.TeamService, stackSvc *service.StackService, redis *redis.Client, logger *logging.Logger, sse *realtime.SSEHub) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RecoveryLogger(logger))
	r.Use(middleware.RequestLogger(cfg.Logging, logger))
	r.Use(middleware.CORS(cfg.AppEnv == "local", cfg.CORS.AllowedOrigins))
	r.Use(middleware.CSRF())

	h := handlers.New(cfg, authSvc, ctfSvc, appConfigSvc, userSvc, scoreSvc, divisionSvc, teamSvc, stackSvc, redis)
	sseHandler := handlers.NewSSEHandler(sse)

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api")
	{
		api.GET("/config", h.GetConfig)

		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/logout", h.Logout)

		api.GET("/challenges", h.ListChallenges)
		api.GET("/scoreboard/stream", sseHandler.ScoreboardStream)
		api.GET("/leaderboard", h.Leaderboard)
		api.GET("/leaderboard/teams", h.TeamLeaderboard)
		api.GET("/timeline", h.Timeline)
		api.GET("/timeline/teams", h.TeamTimeline)
		api.GET("/divisions", h.ListDivisions)
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
		auth.GET("/stacks", h.ListStacks)
		auth.GET("/challenges/:id/stack", h.GetStack)

		unblocked := auth.Group("")
		unblocked.Use(middleware.RequireActiveUser(userSvc))
		unblocked.PUT("/me", h.UpdateMe)
		unblocked.POST("/challenges/:id/submit", h.SubmitFlag)
		unblocked.POST("/challenges/:id/file/download", h.RequestChallengeFileDownload)
		unblocked.POST("/challenges/:id/stack", h.CreateStack)
		unblocked.DELETE("/challenges/:id/stack", h.DeleteStack)

		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.JWT), middleware.RequireActiveUser(userSvc), middleware.RequireRole(models.AdminRole))
		admin.PUT("/config", h.AdminUpdateConfig)
		admin.POST("/challenges", h.CreateChallenge)
		admin.GET("/challenges/:id", h.AdminGetChallenge)
		admin.PUT("/challenges/:id", h.UpdateChallenge)
		admin.DELETE("/challenges/:id", h.DeleteChallenge)
		admin.POST("/challenges/:id/file/upload", h.RequestChallengeFileUpload)
		admin.DELETE("/challenges/:id/file", h.DeleteChallengeFile)
		admin.POST("/registration-keys", h.CreateRegistrationKeys)
		admin.GET("/registration-keys", h.ListRegistrationKeys)
		admin.POST("/divisions", h.CreateDivision)
		admin.POST("/teams", h.CreateTeam)
		admin.GET("/stacks", h.AdminListStacks)
		admin.GET("/stacks/:stack_id", h.AdminGetStack)
		admin.DELETE("/stacks/:stack_id", h.AdminDeleteStack)
		admin.GET("/report", h.AdminReport)
		admin.POST("/users/:id/team", h.AdminMoveUserTeam)
		admin.POST("/users/:id/block", h.AdminBlockUser)
		admin.POST("/users/:id/unblock", h.AdminUnblockUser)
	}

	return r
}
