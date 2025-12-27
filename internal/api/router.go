package api

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/controller"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures a Gin router with all application routes
func SetupRouter(container *Container) *gin.Engine {
	r := gin.Default()

	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment

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

	// Register routes once on the root router
	// The middleware in setupMiddleware handles /api prefix stripping,
	// so routes work with or without the /api prefix
	setupRoutes(r, container)

	return r
}

func setupRoutes(r gin.IRouter, container *Container) {
	log.Print(os.Getenv("GO_ENV"))
	
	setupMiddleware(r)
	setupPublicRoutes(r)
	setupAuthRoutes(r)
	setupLocationAndForecastRoutes(r, container)
	setupSwellPredictionRoutes(r, container)
	setupProtectedRoutes(r, container)
	setupAPIKeyRoutes(r, container)
	setupAdminRoutes(r, container)
}

// setupMiddleware configures middleware for all routes.
func setupMiddleware(r gin.IRouter) {
	// Apply iOS headers to all API routes
	r.Use(iOSHeadersMiddleware())
	
	// Strip /api prefix if present
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			newPath := strings.TrimPrefix(path, "/api")
			c.Request.URL.Path = newPath
		}
		c.Next()
	})
}

// setupPublicRoutes configures public authentication routes.
func setupPublicRoutes(r gin.IRouter) {
	r.POST("/auth/google", auth.GoogleAuthHandler)
	r.GET("/auth/validate", auth.ValidateTokenHandler)
	r.POST("/auth/logout", auth.LogoutHandler)
}

// setupAuthRoutes configures authenticated auth-related routes.
func setupAuthRoutes(r gin.IRouter) {
	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
	
	// CSRF token refresh endpoint (requires authentication)
	csrfRoutes := r.Group("/auth")
	if isLocal {
		csrfRoutes.Use(DevAuthMiddleware())
	} else {
		csrfRoutes.Use(auth.AuthMiddleware())
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
	r.GET("/regions", controller.GetRegions)
	r.GET("/spots", controller.GetSpots)
	r.GET("/location", controller.GetCoordinates)
	r.GET("/locationInfo", controller.GetLocationInfo)
	
	// Forecast routes
	r.GET("/listSpotsForecast", container.ForecastController.GetListSpotsForecast)
	r.GET("/regionForecast", container.ForecastController.GetRegionForecast)
	r.GET("/currentConditions", container.ForecastController.GetCurrentWeather)
	r.GET("/beforeAfterTide", container.ForecastController.GetBeforeAfterTides)
	r.GET("/tideExtremes", container.ForecastController.GetDayTides)
	r.GET("/forecast", container.ForecastController.GetSpotForecast)
	
	// Buoy data routes
	r.GET("/getLiveBuoyData", controller.GetLiveBuoyData)
	r.GET("/getSingleBuoyData", controller.GetSingleBuoyData)
	r.GET("/getLast24BuoyData", controller.GetLast24HoursBuoyData)
	r.GET("/getBuoyDataRange", controller.GetBuoyDataRange)
	r.GET("/getMultipleBuoyData", controller.GetMultipleBuoyData)
	r.GET("/buoyLocationInfo", controller.BuoyLocationInfo)
	r.GET("/regionBuoys", controller.GetRegionBuoys)
	r.GET("/individualBuoyLocation", controller.IndividualBuoyLocationInfo)
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
func setupProtectedRoutes(r gin.IRouter, container *Container) {
	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
	
	// Create authenticated route group
	authorized := r.Group("/")
	if isLocal {
		log.Print("using dev middleware")
		authorized.Use(DevAuthMiddleware())
	} else {
		log.Print("using production auth middleware")
		authorized.Use(auth.AuthMiddleware())
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
	g.POST("/submitSurfReport", controller.SubmitCurrentSurfReport)
	g.POST("/submitSurfReportWithS3Image", controller.SubmitSurfReportWithS3Image)
	g.POST("/submitSurfReportWithIOSValidation", controller.SubmitSurfReportWithIOSValidation)
	g.GET("/generateImageUploadURL", controller.GenerateImageUploadURL)
	g.GET("/generateVideoUploadURL", controller.GenerateVideoUploadURL)
	g.DELETE("/deleteUploadedMedia", controller.DeleteUploadedMedia)
	g.DELETE("/deleteMyAccount", controller.DeleteMyAccount)
	g.PUT("/setTheme", controller.SetUserTheme)
	g.DELETE("/sessions/:sessionId", auth.TerminateSessionHandler)
}

// setupUserRoutes configures user-related read routes.
func setupUserRoutes(g *gin.RouterGroup, container *Container) {
	g.GET("/sessions", auth.GetUserSessionsHandler)
	g.GET("/getTheme", controller.GetUserTheme)
	g.GET("/getTodaySpotReports", controller.RetrieveTodaysSurfReports)
	g.GET("/getAllSpotReports", controller.GetAllSpotSurfReports)
	g.GET("/getSurfReportsWithSimilarBuoyData", controller.GetSurfReportsWithSimilarBuoyData)
	g.GET("/getSurfReportsWithMatchingConditions", controller.GetSurfReportsWithMatchingConditions)
	g.GET("getReportImage", controller.GetReportImage)
	g.GET("/getReportVideo", controller.GetReportVideo)
	g.GET("/generateVideoViewURL", controller.GenerateVideoViewURL)
	g.GET("/ws-token", auth.GetWebSocketTokenHandler)
	g.GET("/streamUrl", controller.GetStreamPlaybackURL)
	g.GET("/latestSnapshot", controller.GetLatestSnapshotHandler)
	g.POST("/requestStream", controller.RequestStreamHandler)
}

// setupAPIKeyRoutes configures routes that require API key authentication.
func setupAPIKeyRoutes(r gin.IRouter, container *Container) {
	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
	
	apiKeyRoutes := r.Group("/")
	if isLocal {
		apiKeyRoutes.Use(DevAuthMiddleware())
	} else {
		apiKeyRoutes.Use(APIKeyAuthMiddleware("stream"))
	}
	
	apiKeyRoutes.POST("/streaming-credentials", controller.GetStreamingCredentials)
	apiKeyRoutes.GET("/check-streaming-requested", controller.CheckStreamRequestHandler)
	apiKeyRoutes.POST("/upload-snapshot", controller.UploadSnapshotHandler)
}

// setupAdminRoutes configures admin-only routes.
func setupAdminRoutes(r gin.IRouter, container *Container) {
	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
	
	adminRoutes := r.Group("/admin")
	if isLocal {
		adminRoutes.Use(DevAdminAuthMiddleware())
	} else {
		log.Print("using production admin middleware")
		adminRoutes.Use(auth.AuthMiddleware(), AdminMiddleware())
	}
	
	adminRoutes.POST("/api-keys", controller.CreateAPIKeyHandler)
	adminRoutes.GET("/api-keys", controller.ListAPIKeysHandler)
	adminRoutes.DELETE("/api-keys/:keyID", controller.RevokeAPIKeyHandler)
}