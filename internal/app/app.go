package app

import (
	"context"
	"log/slog"
	"net/http"
	"treblesurf-backend/internal/observability"

	"github.com/gin-gonic/gin"

	httphandler "treblesurf-backend/internal/api"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/logging"
	"treblesurf-backend/internal/service"
	"treblesurf-backend/internal/websocket"
)

// App wires the application and exposes the HTTP handler.
type App struct {
	telemetry *observability.Runtime
	config    *config.Config
	container *httphandler.Container
	router    *gin.Engine
}

// New builds the application from config.
func New(cfg *config.Config) (*App, error) {
	logging.Init(cfg)
	telemetry, err := observability.Start(context.Background(), cfg.Observability)
	if err != nil {
		slog.Warn("trace exporter initialization failed; continuing without trace export")
		telemetry = &observability.Runtime{}
	}

	container, err := httphandler.NewContainer(cfg)
	if err != nil {
		if shutdownErr := telemetry.Shutdown(context.Background()); shutdownErr != nil {
			slog.Warn("telemetry shutdown failed")
		}
		return nil, err
	}

	router := httphandler.SetupRouter(cfg, container)

	return &App{
		telemetry: telemetry,
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
func (a *App) Shutdown(ctx context.Context) error {
	if a.container != nil {
		a.container.Close()
	}
	return a.telemetry.Shutdown(ctx)
}

func (a *App) FlushTelemetry() { a.telemetry.Flush() }

// WebSocketApp provides the WebSocket handler built from the shared container.
type WebSocketApp struct {
	config    *config.Config
	container *httphandler.Container
	handler   *websocket.Handler
}

// NewWebSocket builds a WebSocket application from config, reusing the shared container.
func NewWebSocket(cfg *config.Config) (*WebSocketApp, error) {
	logging.Init(cfg)

	container, err := httphandler.NewContainer(cfg)
	if err != nil {
		return nil, err
	}

	handler := websocket.NewHandler(container.WebSocketService)

	return &WebSocketApp{
		config:    cfg,
		container: container,
		handler:   handler,
	}, nil
}

// Handler returns the WebSocket handler for Lambda.
func (a *WebSocketApp) Handler() *websocket.Handler {
	return a.handler
}

// WebSocketService returns the WebSocket service for direct access.
func (a *WebSocketApp) WebSocketService() *service.WebSocketService {
	return a.container.WebSocketService
}
