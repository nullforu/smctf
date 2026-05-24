package middleware

import (
	"log/slog"
	"net/http"

	"smctf/internal/logging"

	"github.com/gin-gonic/gin"
)

func RecoveryLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var log *slog.Logger
		if logger != nil {
			log = logger.Logger
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				if log != nil {
					log.Error("panic recovered",
						slog.Any("error", recovered),
						slog.String("path", ctx.Request.URL.Path),
						slog.String("method", ctx.Request.Method),
					)
				}

				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		ctx.Next()
	}
}
