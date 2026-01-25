package http

import (
	nethttp "net/http"

	"smctf/internal/config"
	"smctf/internal/http/handlers"
	"smctf/internal/http/middleware"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, authSvc *service.AuthService, ctfSvc *service.CTFService, userRepo *repo.UserRepo) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AppEnv != "production", nil))

	h := handlers.New(cfg, authSvc, ctfSvc, userRepo)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/logout", h.Logout)

		api.GET("/challenges", h.ListChallenges)
		api.GET("/scoreboard", h.Scoreboard)
		api.GET("/scoreboard/timeline", h.ScoreboardTimeline)

		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWT))
		auth.GET("/me", h.Me)
		auth.GET("/me/solved", h.MeSolved)
		auth.POST("/challenges/:id/submit", h.SubmitFlag)

		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.JWT), middleware.RequireRole("admin"))
		admin.POST("/challenges", h.CreateChallenge)
	}

	attachFrontendRoutes(r)

	return r
}
