package app

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	httphandler "treblesurf-backend/internal/api"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/logging"
)

// App wires the application and exposes the HTTP handler.
type App struct {
	config    *config.Config
	container *httphandler.Container
	router    *gin.Engine
}

// New builds the application from config.
func New(cfg *config.Config) (*App, error) {
	logging.Init(cfg)

	container, err := httphandler.NewContainer(cfg)
	if err != nil {
		return nil, err
	}

	router := httphandler.SetupRouter(cfg, container)

	return &App{
		config:    cfg,
		container: container,
		router:    router,
	}, nil
}

// Handler returns the HTTP handler for servers.
func (a *App) Handler() http.Handler {
	return a.router
}

// GinEngine exposes the Gin engine for Lambda adapters.
func (a *App) GinEngine() *gin.Engine {
	return a.router
}

// Shutdown allows cleanup when running as a server.
func (a *App) Shutdown(_ context.Context) error {
	return nil
}
