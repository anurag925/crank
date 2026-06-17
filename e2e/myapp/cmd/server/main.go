package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "myapp/docs"
	"myapp/internal/adapters/eventbus"
	"myapp/internal/adapters/http/web"
	"myapp/internal/adapters/http/web/middleware"
	"myapp/internal/adapters/persistence/gorm"
	"myapp/internal/adapters/uow"
	userapp "myapp/internal/application/user"
	"myapp/internal/config"
	"myapp/internal/domain/user"
	"myapp/internal/model"
	"myapp/internal/validator"
	"myapp/pkg/logging"
)

// @title           myapp API
// @version         1.0
// @description     Production-ready backend service scaffolded by crank.
//
// @host            localhost:8080
// @BasePath        /
//
// @accept          json
// @produce         json
//
// @securityDefinitions.apikey BearerAuth
// @in   header
// @name Authorization
// @description Enter "Bearer {token}" to authenticate.

func main() {
	cfg := config.Load()

	level := parseLevel(cfg.Logging.Level)
	logger := logging.New(level, cfg.Logging.AddSource)
	slog.SetDefault(logger)

	logger.Info("starting application",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
		"log_level", cfg.Logging.Level,
	)

	gormDB, err := gorm.NewDB(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, dbErr := gormDB.DB()
		if dbErr != nil {
			return
		}
		_ = sqlDB.Close()
	}()

	// ---- Composition root: explicit DDD wiring ----
	bus := eventbus.NewInMemory()

	userRepo := gorm.NewUserRepository(gormDB)

	uow := uow.NewInMemoryUoW(bus)

	userCmd := userapp.NewCommandHandler(userRepo, uow)
	userQry := userapp.NewQueryHandler(userRepo)
	userHandler := web.NewUserHandler(userCmd, userQry)

	e := web.NewServer(logger)
	e.HideBanner = true
	e.HidePort = true
	e.Use(echomw.Recover())
	e.Use(middleware.RequestLogger())

	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())
	e.GET("/health", web.Health)

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		reqLog := logging.FromContext(c.Request().Context())

		if ve, ok := err.(*validator.ValidationError); ok {
			reqLog.Warn("validation failed", "errors", ve.Errors)
			_ = c.JSON(ve.HTTPStatus, model.APIError{Error: ve.Message, Details: ve.Errors})
			return
		}
		if he, ok := err.(*echo.HTTPError); ok {
			if msg, ok := he.Message.(string); ok {
				_ = c.JSON(he.Code, model.APIError{Error: msg})
				return
			}
			_ = c.JSON(he.Code, echo.Map{"error": he.Message})
			return
		}
		reqLog.Error("unhandled error", "error", err)
		_ = c.JSON(http.StatusInternalServerError, model.APIError{Error: "internal server error"})
	}

	web.Mount(e, web.MountConfig{UserHandler: userHandler})

	addr := cfg.App.Host + ":" + strconv.Itoa(cfg.App.Port)
	logger.Info("server listening", "addr", addr)

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

var _ = user.ErrUserNotFound
