package api

import (
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures a Gin router with all application routes
func SetupRouter() *gin.Engine {
    r := gin.Default()

	isLocal := os.Getenv("GO_ENV")

	setupRoutes(r)

	if isLocal == "development" {
        // Add CORS middleware for development environment
        r.Use(cors.New(cors.Config{
            AllowOrigins:     []string{"http://localhost:5173"},
            AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
            AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
            ExposeHeaders:    []string{"Content-Length"},
            AllowCredentials: true,
        }))
        
        apiGroup := r.Group("/api")
        setupRoutes(apiGroup)
    }
    // Enhanced logging middleware
	return r
}

func setupRoutes(r gin.IRouter) {
    
    

	log.Print(os.Getenv("GO_ENV"))
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			newPath := strings.TrimPrefix(path, "/api")
			c.Request.URL.Path = newPath
		}
		c.Next()
	})
    

    if err := InitSessionService(); err != nil {
        log.Printf("Failed to initialize session service: %v", err)
    }

    r.POST("/auth/google", GoogleAuthHandler)
    r.GET("/auth/validate", ValidateTokenHandler)
    r.POST("/auth/logout", LogoutHandler)

    r.GET("/regions", GetRegions)                      // Expects ?country=
    r.GET("/spots", GetSpots)                         // Expects ?region= &country=
    r.GET("/location", GetCoordinates)                // Expects ?country= &region= &spot=
    r.GET("/listSpotsForecast", GetListSpotsForecast) // Expects ?country= &region= &spots=
    r.GET("/regionForecast", GetRegionForecast)       // Expects ?country= &region=
    r.GET("/currentConditions", GetCurrentWeather)     // Expects ?country= &region= &spot=
    r.GET("/beforeAfterTide", GetBeforeAfterTides)    // Expects ?locationName=
    r.GET("/tideExtremes", GetDayTides)               // Expects ?locationName= &start=
    r.GET("/getLiveBuoyData", GetLiveBuoyData)
    r.GET("/getSingleBuoyData", GetSingleBuoyData)    // Expects ?buoyName=
    r.GET("/getLast24BuoyData", GetLast24HoursBuoyData) // Expects ?buoyName=
    r.GET("/getMultipleBuoyData", GetMultipleBuoyData)      // Expects ?buoys=
    r.GET("/buoyLocationInfo", BuoyLocationInfo)
    r.GET("/individualBuoyLocation", IndividualBuoyLocationInfo) // Expects ?buoyName=
    r.GET("/locationInfo", GetLocationInfo)   
    r.GET("/forecast", GetSpotForecast)               // Expects ?country= &region= &spot=


	isLocal := os.Getenv("GO_ENV") == "development"

    // Protected routes (auth required)
	authorized := r.Group("/")
    if isLocal {
		log.Print("using dev middleware")
        // In local development, use a mock auth middleware that automatically authenticates
        authorized.Use(DevAuthMiddleware())
    } else {
        // In production, use the real auth middleware
        authorized.Use(AuthMiddleware())
    }    // authorized.Use(ClientTypeMiddleware()) // First detect client type
  
	// webModifyGroup.Use(func(c *gin.Context) {
	// // Skip this middleware for app clients
	// isAppClient, _ := c.Get("isAppClient")
	// if isAppClient.(bool) {
	//     c.Next()
	//     return
	// }
	// // Apply CSRF only for web clients
	// CSRFMiddleware()(c)
	// })
	webModifyGroup := authorized.Group("/")
    if !isLocal {
        // Only apply CSRF in production, skip in local dev
        webModifyGroup.Use(CSRFMiddleware())
    }
		webModifyGroup.POST("/submitSurfReport", SubmitCurrentSurfReport)
		webModifyGroup.DELETE("/deleteMyAccount", DeleteMyAccount)
		webModifyGroup.PUT("/setTheme", SetUserTheme)
		webModifyGroup.DELETE("/sessions/:sessionId", TerminateSessionHandler)

	
	authorized.GET("/sessions", GetUserSessionsHandler)
	authorized.GET("/getTheme", GetUserTheme)
	authorized.GET("/getTodaySpotReports", RetrieveTodaysSurfReports) // Expects ?country= &region= &spot=
	authorized.GET("getReportImage", GetReportImage) // Expects ?key=
	authorized.GET("/ws-token", GetWebSocketTokenHandler)
	authorized.GET("/streamUrl", GetStreamPlaybackURL)
	authorized.GET("/latestSnapshot", GetLatestSnapshotHandler) // For web users to view snapshots
	authorized.POST("/requestStream", RequestStreamHandler)



    

apiKeyRoutes := r.Group("/")
    if isLocal {
        // In local dev, use mock auth
        apiKeyRoutes.Use(DevAuthMiddleware())
    } else {
        // In production, require API key
        apiKeyRoutes.Use(APIKeyAuthMiddleware("stream"))
    }    
        apiKeyRoutes.GET("/streaming-credentials", GetStreamingCredentials)
        apiKeyRoutes.GET("/check-streaming-requested", CheckStreamRequestHandler)
        apiKeyRoutes.POST("/upload-snapshot", UploadSnapshotHandler) // Add this line

    

    adminRoutes := r.Group("/admin")
    if isLocal {
        // In local dev, use mock auth that gives admin privileges
        adminRoutes.Use(DevAdminAuthMiddleware())
    } else {
        // In production, use real auth with admin check
        adminRoutes.Use(AuthMiddleware(), AdminMiddleware())
    }
    
    // Admin routes
    adminRoutes.POST("/api-keys", CreateAPIKeyHandler)
    adminRoutes.GET("/api-keys", ListAPIKeysHandler)
    adminRoutes.DELETE("/api-keys/:keyID", RevokeAPIKeyHandler)

}
