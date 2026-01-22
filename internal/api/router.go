// Package httphandler provides HTTP API routing configuration, route registration,
// and middleware setup for the Treble Surf backend API.
package httphandler

import (
	"log/slog"
	"net/http"
	"time"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures a Gin router with all application routes.
func SetupRouter(cfg *config.Config, container *Container) *gin.Engine {
	r := gin.Default()

	// Apply CORS middleware
	r.Use(buildCORSMiddleware(cfg))

	// Apply rate limiting in production
	if !cfg.IsDevelopment() {
		r.Use(RateLimitMiddlewareWithLimiter(container.rateLimiter))
	}

	// Register routes
	setupRoutes(r, cfg, container)

	return r
}

// buildCORSMiddleware creates the CORS middleware configuration.
func buildCORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token", "User-Agent"},
		ExposeHeaders:    []string{"Content-Length", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if cfg.IsDevelopment() {
		corsConfig.AllowOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
	} else if len(cfg.Security.AllowedOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.Security.AllowedOrigins
	} else {
		// Fallback: restrict to treblesurf domains
		corsConfig.AllowOrigins = []string{
			"https://treblesurf.com",
			"https://www.treblesurf.com",
		}
	}

	return cors.New(corsConfig)
}

func setupRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	slog.Info("environment", slog.String("env", string(cfg.Env)))
	isLocal := cfg.IsDevelopment()

	// Health check endpoint for container orchestration
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Apply iOS headers to all API routes
	r.Use(iOSHeadersMiddleware())

	// In production, Lambda handler strips /api before routing, so routes are registered without /api
	// In local development, routes need /api prefix since there's no Lambda handler
	var routeGroup gin.IRouter
	if isLocal {
		routeGroup = r.Group("/api")
	} else {
		routeGroup = r
	}

	setupPublicRoutes(routeGroup, container.AuthService)
	setupAuthRoutes(routeGroup, cfg, container.AuthService)
	setupLocationAndForecastRoutes(routeGroup, container)
	setupSwellPredictionRoutes(routeGroup, container)
	setupProtectedRoutes(routeGroup, cfg, container)
	setupAPIKeyRoutes(routeGroup, cfg, container)
	setupAdminRoutes(routeGroup, cfg, container)
}

// setupPublicRoutes configures public authentication routes.
func setupPublicRoutes(r gin.IRouter, authService *auth.Service) {
	r.POST("/auth/google", authService.GoogleAuthHandler)
	r.GET("/auth/validate", authService.ValidateTokenHandler)
	r.POST("/auth/logout", authService.LogoutHandler)
}

// setupAuthRoutes configures authenticated auth-related routes.
func setupAuthRoutes(r gin.IRouter, cfg *config.Config, authService *auth.Service) {
	isLocal := cfg.IsDevelopment()

	// CSRF token refresh endpoint (requires authentication)
	csrfRoutes := r.Group("/auth")
	if isLocal {
		csrfRoutes.Use(DevAuthMiddleware(authService))
	} else {
		csrfRoutes.Use(authService.Middleware())
	}
	csrfRoutes.GET("/csrf", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "CSRF token available"})
	})

	// Development-only endpoint for iOS simulator
	if isLocal {
		r.POST("/auth/dev-session", func(c *gin.Context) {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}

			if err := authService.CreateDevSession(req.Email, c); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Development session created"})
		})
	}
}

// setupLocationAndForecastRoutes configures location, forecast, and buoy data routes.
func setupLocationAndForecastRoutes(r gin.IRouter, container *Container) {
	// Location routes
	r.GET("/regions", container.LocationController.GetRegions)
	r.GET("/spots", container.LocationController.GetSpots)
	r.GET("/location", container.LocationController.GetCoordinates)
	r.GET("/locationInfo", container.LocationController.GetLocationInfo)

	// Forecast routes
	r.GET("/listSpotsForecast", container.ForecastController.GetListSpotsForecast)
	r.GET("/regionForecast", container.ForecastController.GetRegionForecast)
	r.GET("/currentConditions", container.ForecastController.GetCurrentWeather)
	r.GET("/beforeAfterTide", container.ForecastController.GetBeforeAfterTides)
	r.GET("/tideExtremes", container.ForecastController.GetDayTides)
	r.GET("/forecast", container.ForecastController.GetSpotForecast)

	// Buoy data routes
	r.GET("/getLiveBuoyData", container.BuoyController.GetLiveBuoyData)
	r.GET("/getSingleBuoyData", container.BuoyController.GetSingleBuoyData)
	r.GET("/getLast24BuoyData", container.BuoyController.GetLast24HoursBuoyData)
	r.GET("/getBuoyDataRange", container.BuoyController.GetBuoyDataRange)
	r.GET("/getMultipleBuoyData", container.BuoyController.GetMultipleBuoyData)
	r.GET("/buoyLocationInfo", container.BuoyController.BuoyLocationInfo)
	r.GET("/regionBuoys", container.BuoyController.GetRegionBuoys)
	r.GET("/individualBuoyLocation", container.BuoyController.IndividualBuoyLocationInfo)
}

// setupSwellPredictionRoutes configures swell prediction routes.
func setupSwellPredictionRoutes(r gin.IRouter, container *Container) {
	r.GET("/swellPrediction", container.SwellPredictionController.GetSpotSwellPrediction)
	r.GET("/listSpotsSwellPrediction", container.SwellPredictionController.GetListSpotsSwellPrediction)
	r.GET("/regionSwellPrediction", container.SwellPredictionController.GetRegionSwellPrediction)
	r.GET("/swellPredictionRange", container.SwellPredictionController.GetSpotSwellPredictionRange)
	r.GET("/recentSwellPredictions", container.SwellPredictionController.GetRecentSwellPredictions)
	r.GET("/swellPredictionStatus", container.SwellPredictionController.GetSwellPredictionStatus)
	r.GET("/closestAIPrediction", container.SwellPredictionController.GetClosestAIPredictionForSpot)
}

// setupProtectedRoutes configures authenticated routes that require user authentication.
func setupProtectedRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	isLocal := cfg.IsDevelopment()

	// Create authenticated route group
	authorized := r.Group("/")
	if isLocal {
		slog.Info("using dev middleware")
		authorized.Use(DevAuthMiddleware(container.AuthService))
	} else {
		slog.Info("using production auth middleware")
		authorized.Use(container.AuthService.Middleware())
	}

	// Routes that modify data (require CSRF in production)
	webModifyGroup := authorized.Group("/")
	if !isLocal {
		slog.Info("using production CSRF middleware")
		webModifyGroup.Use(container.AuthService.CSRFMiddleware())
	}

	setupReportModificationRoutes(webModifyGroup, container)
	setupUserRoutes(authorized, container)
}

// setupReportModificationRoutes configures routes that modify surf reports.
func setupReportModificationRoutes(g *gin.RouterGroup, container *Container) {
	g.POST("/submitSurfReport", container.ReportController.SubmitCurrentSurfReport)
	g.POST("/submitSurfReportWithS3Image", container.ReportController.SubmitSurfReportWithS3Image)
	g.POST("/submitSurfReportWithIOSValidation", container.ReportController.SubmitSurfReportWithIOSValidation)
	g.GET("/generateImageUploadURL", container.ReportController.GenerateImageUploadURL)
	g.GET("/generateVideoUploadURL", container.ReportController.GenerateVideoUploadURL)
	g.DELETE("/deleteUploadedMedia", container.ReportController.DeleteUploadedMedia)
	g.DELETE("/deleteMyAccount", container.UserController.DeleteMyAccount)
	g.PUT("/setTheme", container.UserController.SetUserTheme)
	g.DELETE("/sessions/:sessionId", container.AuthService.TerminateSessionHandler)
}

// setupUserRoutes configures user-related read routes.
func setupUserRoutes(g *gin.RouterGroup, container *Container) {
	g.GET("/sessions", container.AuthService.GetUserSessionsHandler)
	g.GET("/user/preferences", container.UserController.GetUserPreferences)
	g.GET("/getTheme", container.UserController.GetUserTheme)
	g.GET("/getTodaySpotReports", container.ReportController.RetrieveTodaysSurfReports)
	g.GET("/getAllSpotReports", container.ReportController.GetAllSpotSurfReports)
	g.GET("/getSurfReportsWithSimilarBuoyData", container.ReportController.GetSurfReportsWithSimilarBuoyData)
	g.GET("/getSurfReportsWithMatchingConditions", container.ReportController.GetSurfReportsWithMatchingConditions)
	g.GET("/getReportImage", container.ReportController.GetReportImage)
	g.GET("/getReportVideo", container.ReportController.GetReportVideo)
	g.GET("/generateVideoViewURL", container.ReportController.GenerateVideoViewURL)
	g.GET("/ws-token", container.AuthService.GetWebSocketTokenHandler)
	g.GET("/streamUrl", container.StreamController.GetStreamPlaybackURL)
	g.GET("/latestSnapshot", container.SnapshotController.GetLatestSnapshotHandler)
	g.POST("/requestStream", container.StreamController.RequestStreamHandler)
}

// setupAPIKeyRoutes configures routes that require API key authentication.
func setupAPIKeyRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	isLocal := cfg.IsDevelopment()

	apiKeyRoutes := r.Group("/")
	if isLocal {
		apiKeyRoutes.Use(DevAuthMiddleware(container.AuthService))
	} else {
		apiKeyRoutes.Use(APIKeyAuthMiddleware(container.APIKeyService, "stream"))
	}

	apiKeyRoutes.POST("/streaming-credentials", container.StreamController.GetStreamingCredentials)
	apiKeyRoutes.GET("/check-streaming-requested", container.StreamController.CheckStreamRequestHandler)
	apiKeyRoutes.POST("/upload-snapshot", container.SnapshotController.UploadSnapshotHandler)
}

// setupAdminRoutes configures admin-only routes.
func setupAdminRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	isLocal := cfg.IsDevelopment()

	adminRoutes := r.Group("/admin")
	if isLocal {
		adminRoutes.Use(DevAdminAuthMiddleware(container.AuthService))
	} else {
		slog.Info("using production admin middleware")
		adminRoutes.Use(container.AuthService.Middleware(), AdminMiddlewareWithConfig(cfg))
	}

	adminRoutes.POST("/api-keys", container.APIKeyController.CreateAPIKeyHandler)
	adminRoutes.GET("/api-keys", container.APIKeyController.ListAPIKeysHandler)
	adminRoutes.DELETE("/api-keys/:keyID", container.APIKeyController.RevokeAPIKeyHandler)
}
