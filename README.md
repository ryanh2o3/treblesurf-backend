# Treble Surf Backend

Backend API for surf forecasting, real-time conditions, and surf spot management. Built with Go and deployed on AWS. Recent refactors allow for future easy expansion to support other database typed by new repository implementations, they should work with services directly

## Overview

This API provides surf forecasting, weather conditions, buoy data, surf reports, and live streaming for surf spots. It's designed to give surfers accurate, up-to-date information about surf conditions and enable community-driven reporting.

## Tech Stack

- Go 1.24.0
- Gin web framework
- DynamoDB for data storage
- AWS S3 for images and snapshots
- Google OAuth for authentication
- AWS Lambda with API Gateway for hosting
- WebSocket support for real-time updates
- AWS Kinesis Video Streams for streaming

## What It Does

The API handles:

- Surf forecasts and weather predictions for specific spots
- AI-powered swell predictions (calculated in the buoy ingestions - a simple algorithm, not really ai lol)
- Real-time weather and ocean conditions
- Live ocean buoy data and historical measurements
- Community surf reports with photos and videos
- Live streaming from surf spots with snapshot capture
- Location management (spots, regions, coordinates)
- User accounts, authentication, and preferences

## Project Structure

```
treblesurf-backend/
├── cmd/                    # Application entry points
│   ├── api/               # API Lambda entry point
│   ├── server/            # HTTP server entry point
│   └── websocket/         # WebSocket server
├── internal/               # Internal application code
│   ├── api/               # API setup and routing
│   ├── auth/              # Authentication and authorization
│   ├── controller/        # HTTP request handlers
│   ├── logging/           # Structured logging
│   ├── model/             # Data models and structures
│   ├── service/           # Business logic services
│   ├── storage/           # Data storage interfaces
│   ├── validation/        # Input validation
│   └── websocket/         # WebSocket handling
├── local/                  # Local development setup
└── scripts/                # Deployment and utility scripts
```

## API Endpoints

### Authentication

- `POST /auth/google` - Authenticate with Google OAuth
- `GET /auth/validate` - Validate authentication token
- `POST /auth/logout` - Log out
- `GET /auth/csrf` - Get CSRF token (requires auth)
- `POST /auth/dev-session` - Create development session (dev only)
- `GET /ws-token` - Get WebSocket authentication token

### Public Endpoints

- `GET /regions` - List available surf regions
- `GET /spots` - List available surf spots
- `GET /location` - Get coordinates for a location
- `GET /forecast` - Get surf forecast for a specific spot
- `GET /listSpotsForecast` - Get forecasts for multiple spots
- `GET /regionForecast` - Get forecast for an entire region
- `GET /currentConditions` - Get current weather conditions
- `GET /beforeAfterTide` - Get tide information (before/after)
- `GET /tideExtremes` - Get daily tide extremes
- `GET /locationInfo` - Get detailed location information

### Buoy Data

- `GET /getLiveBuoyData` - Get real-time buoy data
- `GET /getSingleBuoyData` - Get data from a specific buoy
- `GET /getLast24BuoyData` - Get last 24 hours of buoy data
- `GET /getBuoyDataRange` - Get buoy data for a date range
- `GET /getMultipleBuoyData` - Get data from multiple buoys
- `GET /buoyLocationInfo` - Get buoy location information
- `GET /regionBuoys` - Get buoys for a specific region
- `GET /individualBuoyLocation` - Get specific buoy location

### Swell Predictions

- `GET /swellPrediction` - Get AI swell prediction for a specific spot
- `GET /listSpotsSwellPrediction` - Get swell predictions for multiple spots
- `GET /regionSwellPrediction` - Get swell predictions for an entire region
- `GET /swellPredictionRange` - Get swell prediction range for a spot
- `GET /recentSwellPredictions` - Get recent swell predictions
- `GET /swellPredictionStatus` - Get swell prediction service status
- `GET /closestAIPrediction` - Get closest AI prediction for a spot

### User Management (Requires Authentication)

- `GET /sessions` - Get user sessions
- `GET /getTheme` - Get user theme preference
- `PUT /setTheme` - Set user theme preference
- `DELETE /deleteMyAccount` - Delete user account
- `DELETE /sessions/:sessionId` - Terminate specific session

### Surf Reports (Requires Authentication)

- `POST /submitSurfReport` - Submit a new surf report (legacy base64 method)
- `POST /submitSurfReportWithS3Image` - Submit report with S3 image key
- `POST /submitSurfReportWithIOSValidation` - Submit report with iOS validation
- `GET /generateImageUploadURL` - Generate presigned upload URL for images
- `GET /generateVideoUploadURL` - Generate presigned upload URL for videos
- `DELETE /deleteUploadedMedia` - Delete uploaded media files
- `GET /getTodaySpotReports` - Get today's surf reports for a spot
- `GET /getAllSpotReports` - Get all surf reports for a spot
- `GET /getSurfReportsWithSimilarBuoyData` - Get reports with similar buoy conditions
- `GET /getSurfReportsWithMatchingConditions` - Get reports with matching conditions
- `GET /getReportImage` - Get image for a surf report
- `GET /getReportVideo` - Get video for a surf report
- `GET /generateVideoViewURL` - Generate presigned view URL for videos

### Streaming Services (Requires API Key)

- `GET /streamUrl` - Get stream playback URL
- `GET /latestSnapshot` - Get latest stream snapshot
- `POST /requestStream` - Request a new stream
- `POST /streaming-credentials` - Get streaming credentials
- `GET /check-streaming-requested` - Check stream request status
- `POST /upload-snapshot` - Upload stream snapshot

### Admin Endpoints (Admin Only)

- `POST /admin/api-keys` - Create new API key
- `GET /admin/api-keys` - List all API keys
- `DELETE /admin/api-keys/:keyID` - Revoke API key

## Data Models

### User

```go
type User struct {
    UUID       string `json:"uuid" dynamodbav:"uuid"`
    Email      string `json:"email" dynamodbav:"email"`
    Name       string `json:"name" dynamodbav:"name"`
    Picture    string `json:"picture" dynamodbav:"picture"`
    FamilyName string `json:"family_name" dynamodbav:"family_name"`
    GivenName  string `json:"given_name" dynamodbav:"given_name"`
    CreatedAt  string `json:"created_at" dynamodbav:"created_at"`
    LastLogin  string `json:"last_login" dynamodbav:"last_login"`
    Theme      string `json:"theme" dynamodbav:"theme"`
    Role       string `json:"role,omitempty" dynamodbav:"role,omitempty"`
}
```

### UserSession

```go
type UserSession struct {
    Email       string    `json:"email"`
    SessionID   string    `json:"session_id"`
    UserAgent   string    `json:"user_agent"`
    IPAddress   string    `json:"ip_address"`
    CreatedAt   time.Time `json:"created_at"`
    LastActive  time.Time `json:"last_active"`
    CSRFToken   string    `json:"csrf_token"`
}
```

### Forecast

```go
type Forecast struct {
    Location          string    `json:"location" dynamodbav:"location"`
    Country           string    `json:"country" dynamodbav:"country"`
    Region            string    `json:"region" dynamodbav:"region"`
    Spot              string    `json:"spot" dynamodbav:"spot"`
    Date              time.Time `json:"date" dynamodbav:"date"`
    DateForecastedFor string    `json:"dateForecastedFor" dynamodbav:"dateForecastedFor"`
    Hour              int       `json:"hour" dynamodbav:"hour"`
    Temperature       float64   `json:"temperature" dynamodbav:"temperature"`
    WindSpeed         float64   `json:"wind_speed" dynamodbav:"wind_speed"`
    WindDirection     float64   `json:"wind_direction" dynamodbav:"wind_direction"`
    WaveHeight        float64   `json:"wave_height" dynamodbav:"wave_height"`
    WavePeriod        float64   `json:"wave_period" dynamodbav:"wave_period"`
    MaxPeriod         float64   `json:"max_period" dynamodbav:"max_period"`
    WaveDirection     float64   `json:"wave_direction" dynamodbav:"wave_direction"`
    Conditions        string    `json:"conditions" dynamodbav:"conditions"`
}
```

### Surf Report

```go
type SurfReport struct {
    ID              string    `json:"id" dynamodbav:"id"`
    UserEmail      string    `json:"user_email" dynamodbav:"user_email"`
    Country        string    `json:"country" dynamodbav:"country"`
    Region         string    `json:"region" dynamodbav:"region"`
    Spot           string    `json:"spot" dynamodbav:"spot"`
    Timestamp      time.Time `json:"timestamp" dynamodbav:"timestamp"`
    SwellSize      string    `json:"swell_size" dynamodbav:"swell_size"`
    WindAmount     string    `json:"wind_amount" dynamodbav:"wind_amount"`
    WindDirection  string    `json:"wind_direction" dynamodbav:"wind_direction"`
    SurfConditions string    `json:"surf_conditions" dynamodbav:"surf_conditions"`
    SurfDifficulty string    `json:"surf_difficulty" dynamodbav:"surf_difficulty"`
    ImageKey       string    `json:"image_key,omitempty" dynamodbav:"image_key,omitempty"`
    Notes          string    `json:"notes,omitempty" dynamodbav:"notes,omitempty"`
    CreatedAt      time.Time `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt      time.Time `json:"updated_at" dynamodbav:"updated_at"`
}
```

### Buoy Data

```go
type BuoyData struct {
    BuoyName      string    `json:"buoy_name" dynamodbav:"buoy_name"`
    Timestamp     time.Time `json:"timestamp" dynamodbav:"timestamp"`
    WaveHeight    float64   `json:"wave_height" dynamodbav:"wave_height"`
    WavePeriod    float64   `json:"wave_period" dynamodbav:"wave_period"`
    MaxPeriod     float64   `json:"max_period" dynamodbav:"max_period"`
    WaveDirection float64  `json:"wave_direction" dynamodbav:"wave_direction"`
    WindSpeed     float64   `json:"wind_speed" dynamodbav:"wind_speed"`
    WindDirection float64  `json:"wind_direction" dynamodbav:"wind_direction"`
    Temperature   float64   `json:"temperature" dynamodbav:"temperature"`
    Pressure      float64   `json:"pressure" dynamodbav:"pressure"`
}
```

### Stream

```go
type Stream struct {
    ID           string     `json:"id" dynamodbav:"id"`
    UserEmail    string     `json:"user_email" dynamodbav:"user_email"`
    Country      string     `json:"country" dynamodbav:"country"`
    Region       string     `json:"region" dynamodbav:"region"`
    Spot         string     `json:"spot" dynamodbav:"spot"`
    Status       string     `json:"status" dynamodbav:"status"` // requested, active, completed, cancelled
    RequestedAt  time.Time  `json:"requested_at" dynamodbav:"requested_at"`
    StartedAt    *time.Time `json:"started_at,omitempty" dynamodbav:"started_at,omitempty"`
    CompletedAt  *time.Time `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
    StreamURL    string     `json:"stream_url,omitempty" dynamodbav:"stream_url,omitempty"`
    PlaybackURL  string     `json:"playback_url,omitempty" dynamodbav:"playback_url,omitempty"`
    CreatedAt    time.Time  `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at" dynamodbav:"updated_at"`
}
```

## Setup & Development

### Prerequisites

- Go 1.24.0 or higher
- Docker and Docker Compose
- AWS CLI (for production deployment)

### Local Development

1. Clone the repository

   ```bash
   git clone <repository-url>
   cd treblesurf-backend
   ```

2. Start local services

   ```bash
   docker-compose up -d
   ```

   This starts:

   - DynamoDB Local (port 8000)
   - LocalStack (port 4566) for S3, IAM, STS, Rekognition

3. Set environment variables

   ```bash
   export GO_ENV=development
   export AWS_ENDPOINT_URL=http://localhost:4566
   export DYNAMODB_ENDPOINT=http://localhost:8000
   ```

4. Run the application

   ```bash
   # Run API server
   go run cmd/api/main.go

   # Run WebSocket server
   go run cmd/websocket/main.go
   ```

### Environment Configuration

The application supports different environments:

- Development: Uses mock authentication and local services
- Production: Uses real AWS services and authentication

Key environment variables:

- `GO_ENV`: Set to "development" for local development
- `AWS_ENDPOINT_URL`: AWS service endpoint (local for dev)
- `DYNAMODB_ENDPOINT`: DynamoDB endpoint (local for dev)

## Deployment

### AWS Lambda Deployment

The application runs on AWS Lambda with API Gateway:

1. Build the application

   ```bash
   GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/api/main.go
   ```

2. Deploy using the provided script
   ```bash
   ./scripts/deploy.sh
   ```

### Docker Deployment

For container-based deployment:

```bash
docker build -t treblesurf-backend .
docker run -p 8080:8080 treblesurf-backend
```

## Authentication & Security

### Authentication Flow

1. Users authenticate via Google OAuth (supports both web and iOS client IDs)
2. JWT tokens and session cookies are issued for session management
3. Most endpoints require valid authentication via middleware
4. Streaming services require API key authentication
5. Development mode has mock authentication available for local testing

### Security Features

- JWT token validation with configurable expiration
- CSRF protection for state-changing operations (production only)
- Session tracking with IP address and user agent logging
- API key generation and management for streaming services
- Role-based access control for admin endpoints
- CORS configuration for web and mobile clients
- Secure cookies with HTTP-only and secure settings in production
- Users can terminate individual sessions or all sessions

### Environment-Specific Behavior

- Development: Mock authentication, relaxed CORS, no CSRF requirements
- Production: Full Google OAuth validation, strict CORS, CSRF protection enabled

## Data Sources

- Weather data from external weather APIs
- Buoy data from NOAA and other oceanographic sources
- Tide information from tide prediction services
- AI swell predictions from machine learning models
- User reports from community-submitted surf conditions
- Live streams from user-generated content at surf spots

## Media Upload Workflow

The backend supports multiple methods for submitting surf reports with media:

### Legacy Method (Base64)

Submit reports with base64-encoded image data using `/submitSurfReport`.

### Presigned URL Method (Recommended)

Use presigned URLs to upload media to S3, then submit reports with the media key.

#### Image Upload Workflow:

1. Generate upload URL: Call `GET /generateImageUploadURL?country={country}&region={region}&spot={spot}`

   - Returns: `{ "uploadUrl": "...", "imageKey": "...", "expiresAt": "..." }`

2. Upload image: Use the `uploadUrl` to upload the image directly to S3

   - The `imageKey` is predictable and calculable
   - URL expires in 15 minutes

3. Submit report: Call `POST /submitSurfReportWithS3Image` with the report data including the `imageKey`
   - Backend retrieves image from S3
   - Validates image using Rekognition
   - If validation fails, image is automatically deleted from S3
   - If validation passes, report is stored with image reference

#### Video Upload Workflow:

1. Generate upload URL: Call `GET /generateVideoUploadURL?country={country}&region={region}&spot={spot}`

   - Returns: `{ "uploadUrl": "...", "videoKey": "...", "expiresAt": "..." }`

2. Upload video: Use the `uploadUrl` to upload the video directly to S3

3. Submit report: Call `POST /submitSurfReportWithIOSValidation` with the report data including the `videoKey`

#### Benefits:

- Better performance (no base64 encoding/decoding)
- Reduced memory usage
- Better error handling and cleanup
- iOS app can upload media in background
- Predictable media keys for caching
- Support for both images and videos

#### Error Handling:

- Invalid media files are automatically cleaned up from S3
- Orphaned media can be cleaned up using the cleanup methods
- Database failures trigger media cleanup to prevent orphaned files

### iOS Validation Method

Special endpoint for iOS apps with enhanced validation: `POST /submitSurfReportWithIOSValidation`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).

### AGPL-3.0 License Terms

- Copyleft: Any derivative work must also be licensed under AGPL-3.0
- Network Use: If you run a modified version on a server and let other users communicate with it, you must make the source code available
- Commercial Use: Allowed, but you must comply with the copyleft provisions
- Distribution: You must provide the complete source code when distributing

The complete license text is available at: https://www.gnu.org/licenses/agpl-3.0.en.html

**Important**: If you modify this software and run it on a server accessible to users over a network, you must provide users with access to the modified source code under the same license terms.

## Support

For support and questions:

- Create an issue in the repository
- Contact the development team
- Check the documentation for common issues

## API Versioning

The current API version is v1. All endpoints are prefixed with `/api` in production environments.
