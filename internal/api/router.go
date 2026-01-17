// Package httphandler provides HTTP API routing configuration, route registration,
// and middleware setup for the Treble Surf backend API.
package httphandler

import (
	"log"
	"net/http"
	"time"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures a Gin router with all application routes
func SetupRouter(cfg *config.Config, container *Container) *gin.Engine {
	r := gin.Default()

	isLocal := cfg.IsDevelopment()

	// Apply CORS middleware before registering routes
	if isLocal {
		// Add CORS middleware for development environment
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}))
	} else {
		// Production CORS configuration for iOS PWA and other clients
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"}, // Allow all origins in production
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token", "User-Agent"},
			ExposeHeaders:    []string{"Content-Length", "X-CSRF-Token"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour, // Cache preflight for 12 hours
		}))
	}

	// Register routes - conditionally with /api prefix for local dev
	// In production, Lambda handler strips /api before routing
	setupRoutes(r, cfg, container)

	return r
}

func setupRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	log.Print(cfg.Env)
	isLocal := cfg.IsDevelopment()

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

	setupPublicRoutes(routeGroup)
	setupAuthRoutes(routeGroup, cfg)
	setupLocationAndForecastRoutes(routeGroup, container)
	setupSwellPredictionRoutes(routeGroup, container)
	setupProtectedRoutes(routeGroup, cfg, container)
	setupAPIKeyRoutes(routeGroup, cfg, container)
	setupAdminRoutes(routeGroup, cfg, container)
}

// setupPublicRoutes configures public authentication routes.
func setupPublicRoutes(r gin.IRouter) {
	r.POST("/auth/google", auth.GoogleAuthHandler)
	r.GET("/auth/validate", auth.ValidateTokenHandler)
	r.POST("/auth/logout", auth.LogoutHandler)
}

// setupAuthRoutes configures authenticated auth-related routes.
func setupAuthRoutes(r gin.IRouter, cfg *config.Config) {
	isLocal := cfg.IsDevelopment()

	// CSRF token refresh endpoint (requires authentication)
	csrfRoutes := r.Group("/auth")
	if isLocal {
		csrfRoutes.Use(DevAuthMiddleware())
	} else {
		csrfRoutes.Use(auth.Middleware())
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

			if err := auth.CreateDevSession(req.Email, c); err != nil {
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
		log.Print("using dev middleware")
		authorized.Use(DevAuthMiddleware())
	} else {
		log.Print("using production auth middleware")
		authorized.Use(auth.Middleware())
	}

	// Routes that modify data (require CSRF in production)
	webModifyGroup := authorized.Group("/")
	if !isLocal {
		log.Print("using production CSRF middleware")
		webModifyGroup.Use(auth.CSRFMiddleware())
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
	g.DELETE("/sessions/:sessionId", auth.TerminateSessionHandler)
}

// setupUserRoutes configures user-related read routes.
func setupUserRoutes(g *gin.RouterGroup, container *Container) {
	g.GET("/sessions", auth.GetUserSessionsHandler)
	g.GET("/getTheme", container.UserController.GetUserTheme)
	g.GET("/getTodaySpotReports", container.ReportController.RetrieveTodaysSurfReports)
	g.GET("/getAllSpotReports", container.ReportController.GetAllSpotSurfReports)
	g.GET("/getSurfReportsWithSimilarBuoyData", container.ReportController.GetSurfReportsWithSimilarBuoyData)
	g.GET("/getSurfReportsWithMatchingConditions", container.ReportController.GetSurfReportsWithMatchingConditions)
	g.GET("getReportImage", container.ReportController.GetReportImage)
	g.GET("/getReportVideo", container.ReportController.GetReportVideo)
	g.GET("/generateVideoViewURL", container.ReportController.GenerateVideoViewURL)
	g.GET("/ws-token", auth.GetWebSocketTokenHandler)
	g.GET("/streamUrl", container.StreamController.GetStreamPlaybackURL)
	g.GET("/latestSnapshot", container.SnapshotController.GetLatestSnapshotHandler)
	g.POST("/requestStream", container.StreamController.RequestStreamHandler)
}

// setupAPIKeyRoutes configures routes that require API key authentication.
func setupAPIKeyRoutes(r gin.IRouter, cfg *config.Config, container *Container) {
	isLocal := cfg.IsDevelopment()

	apiKeyRoutes := r.Group("/")
	if isLocal {
		apiKeyRoutes.Use(DevAuthMiddleware())
	} else {
		apiKeyRoutes.Use(APIKeyAuthMiddleware("stream"))
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
		adminRoutes.Use(DevAdminAuthMiddleware())
	} else {
		log.Print("using production admin middleware")
		adminRoutes.Use(auth.Middleware(), AdminMiddleware())
	}

	adminRoutes.POST("/api-keys", container.APIKeyController.CreateAPIKeyHandler)
	adminRoutes.GET("/api-keys", container.APIKeyController.ListAPIKeysHandler)
	adminRoutes.DELETE("/api-keys/:keyID", container.APIKeyController.RevokeAPIKeyHandler)
}
