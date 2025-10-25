# Treble Surf Backend

A comprehensive backend API for surf forecasting, real-time conditions, and surf spot management built with Go and AWS services.

## 🏄‍♂️ Overview

Treble Surf Backend is a robust API service that provides surf forecasting, real-time weather conditions, buoy data, surf reports, and live streaming capabilities for surf spots around the world. The application is designed to serve surfers with accurate, up-to-date information about surf conditions and enable community-driven surf reporting.

## 🏗️ Architecture

- **Language**: Go 1.24.0
- **Framework**: Gin (HTTP web framework)
- **Database**: DynamoDB (NoSQL)
- **Storage**: AWS S3 (images, snapshots)
- **Authentication**: Google OAuth with JWT
- **Deployment**: AWS Lambda with API Gateway
- **Real-time**: WebSocket support
- **Streaming**: AWS MediaLive integration

## 🚀 Features

### Core Functionality

- **Surf Forecasting**: Weather and wave predictions for surf spots
- **AI Swell Predictions**: Machine learning-powered swell predictions for enhanced accuracy
- **Real-time Conditions**: Current weather and ocean conditions
- **Buoy Data**: Live ocean buoy measurements and historical data
- **Surf Reports**: Community-driven surf condition reports with image and video uploads
- **Live Streaming**: Real-time surf spot streaming with snapshot capture
- **Location Management**: Surf spots, regions, and coordinate systems
- **User Management**: Authentication, sessions, and user preferences

### Technical Features

- **RESTful API**: Comprehensive HTTP endpoints
- **WebSocket Support**: Real-time data streaming
- **Image & Video Processing**: Surf report media handling with S3 integration
- **API Key Management**: Secure access control for streaming services
- **Multi-environment Support**: Development and production configurations
- **CORS Support**: Cross-origin resource sharing for web and mobile applications
- **Presigned URLs**: Secure direct-to-S3 uploads for media files
- **Session Management**: Advanced session tracking and termination

## 📁 Project Structure

```
treblesurf-backend/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   └── websocket/         # WebSocket server
├── internal/               # Internal application code
│   ├── api/               # API setup and routing
│   ├── auth/              # Authentication and authorization
│   ├── controller/        # HTTP request handlers
│   ├── middleware/        # HTTP middleware
│   ├── model/             # Data models and structures
│   ├── service/           # Business logic services
│   ├── storage/           # Data storage interfaces
│   ├── validation/        # Input validation
│   └── websocket/         # WebSocket handling
├── local/                  # Local development setup
├── models/                 # AWS service models
├── pkg/                    # Public packages
└── scripts/                # Deployment and utility scripts
```

## 🔌 API Endpoints

### Authentication

- `POST /auth/google` - Google OAuth authentication
- `GET /auth/validate` - Validate authentication token
- `POST /auth/logout` - User logout
- `GET /auth/csrf` - Get CSRF token (protected)
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

### User Management (Protected)

- `GET /sessions` - Get user sessions
- `GET /getTheme` - Get user theme preference
- `PUT /setTheme` - Set user theme preference
- `DELETE /deleteMyAccount` - Delete user account
- `DELETE /sessions/:sessionId` - Terminate specific session

### Surf Reports (Protected)

- `POST /submitSurfReport` - Submit a new surf report (legacy base64)
- `POST /submitSurfReportWithS3Image` - Submit report with S3 image key
- `POST /submitSurfReportWithIOSValidation` - Submit report with iOS validation
- `GET /generateImageUploadURL` - Generate presigned upload URL for images
- `GET /generateVideoUploadURL` - Generate presigned upload URL for videos
- `DELETE /deleteUploadedMedia` - Delete uploaded media files
- `GET /getTodaySpotReports` - Get today's surf reports for a spot
- `GET /getReportImage` - Get image for a surf report
- `GET /getReportVideo` - Get video for a surf report
- `GET /generateVideoViewURL` - Generate presigned view URL for videos

### Streaming Services (API Key Required)

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

## 🗄️ Data Models

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

### ReportImage

```go
type ReportImage struct {
    Key         string    `json:"key" dynamodbav:"key"`
    ReportID    string    `json:"report_id" dynamodbav:"report_id"`
    ImageData   []byte    `json:"image_data" dynamodbav:"image_data"`
    ContentType string    `json:"content_type" dynamodbav:"content_type"`
    UploadedAt  time.Time `json:"uploaded_at" dynamodbav:"uploaded_at"`
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

### Buoy

```go
type Buoy struct {
    RegionBuoy string  `json:"region_buoy"`
    Latitude   float64 `json:"latitude" dynamodbav:"latitude"`
    Longitude  float64 `json:"longitude" dynamodbav:"longitude"`
    Name       string  `json:"name"`
}
```

### BuoyLocation

```go
type BuoyLocation struct {
    Name      string  `json:"name" dynamodbav:"name"`
    Country   string  `json:"country" dynamodbav:"country"`
    Region    string  `json:"region" dynamodbav:"region"`
    Spot      string  `json:"spot" dynamodbav:"spot"`
    Latitude  float64 `json:"latitude" dynamodbav:"latitude"`
    Longitude float64 `json:"longitude" dynamodbav:"longitude"`
}
```

### Weather

```go
type Weather struct {
    Location      string    `json:"location" dynamodbav:"location"`
    Country       string    `json:"country" dynamodbav:"country"`
    Region        string    `json:"region" dynamodbav:"region"`
    Spot          string    `json:"spot" dynamodbav:"spot"`
    Timestamp     time.Time `json:"timestamp" dynamodbav:"timestamp"`
    Temperature   float64   `json:"temperature" dynamodbav:"temperature"`
    Humidity      float64   `json:"humidity" dynamodbav:"humidity"`
    Pressure      float64   `json:"pressure" dynamodbav:"pressure"`
    WindSpeed     float64   `json:"wind_speed" dynamodbav:"wind_speed"`
    WindDirection float64   `json:"wind_direction" dynamodbav:"wind_direction"`
    Visibility    float64   `json:"visibility" dynamodbav:"visibility"`
    Conditions    string    `json:"conditions" dynamodbav:"conditions"`
}
```

### Tide

```go
type Tide struct {
    Location    string    `json:"location" dynamodbav:"location"`
    Date        time.Time `json:"date" dynamodbav:"date"`
    Time        time.Time `json:"time" dynamodbav:"time"`
    Height      float64   `json:"height" dynamodbav:"height"`
    Type        string    `json:"type" dynamodbav:"type"` // high, low
    Description string    `json:"description" dynamodbav:"description"`
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

### StreamCredentials

```go
type StreamCredentials struct {
    AccessKeyID     string `json:"access_key_id"`
    SecretAccessKey string `json:"secret_access_key"`
    SessionToken    string `json:"session_token"`
    Region          string `json:"region"`
    StreamName      string `json:"stream_name"`
}
```

### Snapshot

```go
type Snapshot struct {
    ID        string    `json:"id" dynamodbav:"id"`
    StreamID  string    `json:"stream_id" dynamodbav:"stream_id"`
    ImageKey  string    `json:"image_key" dynamodbav:"image_key"`
    Timestamp time.Time `json:"timestamp" dynamodbav:"timestamp"`
    CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}
```

### APIKey

```go
type APIKey struct {
    KeyID       string    `json:"key_id" dynamodbav:"key_id"`
    KeyValue    string    `json:"key_value" dynamodbav:"key_value"`
    Description string    `json:"description" dynamodbav:"description"`
    CreatedBy   string    `json:"created_by" dynamodbav:"created_by"`
    CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
    ExpiresAt   time.Time `json:"expires_at" dynamodbav:"expires_at"`
    Scopes      []string  `json:"scopes" dynamodbav:"scopes"`
}
```

### Location

```go
type Location struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}
```

### LocationInfo

```go
type LocationInfo struct {
    BeachDirection      int     `json:"BeachDirection"`
    Elevation           int     `json:"Elevation"`
    IdealSwellDirection string  `json:"IdealSwellDirection"`
    Image               string  `json:"Image"`
    Latitude            float64 `json:"Latitude"`
    Longitude           float64 `json:"Longitude"`
    Type                string  `json:"Type"`
    CountryRegionSpot   string  `json:"country_region_spot"`
    ImageString         string  `json:"ImageString"`
}
```

## 🛠️ Setup & Development

### Prerequisites

- Go 1.24.0 or higher
- Docker and Docker Compose
- AWS CLI (for production deployment)

### Local Development

1. **Clone the repository**

   ```bash
   git clone <repository-url>
   cd treblesurf-backend
   ```

2. **Start local services**

   ```bash
   docker-compose up -d
   ```

   This starts:

   - DynamoDB Local (port 8000)
   - LocalStack (port 4566) for S3, IAM, STS, Rekognition

3. **Set environment variables**

   ```bash
   export GO_ENV=development
   export AWS_ENDPOINT_URL=http://localhost:4566
   export DYNAMODB_ENDPOINT=http://localhost:8000
   ```

4. **Run the application**

   ```bash
   # Run API server
   go run cmd/api/main.go

   # Run WebSocket server
   go run cmd/websocket/main.go
   ```

### Environment Configuration

The application supports different environments:

- **Development**: Uses mock authentication and local services
- **Production**: Uses real AWS services and authentication

Key environment variables:

- `GO_ENV`: Set to "development" for local development
- `AWS_ENDPOINT_URL`: AWS service endpoint (local for dev)
- `DYNAMODB_ENDPOINT`: DynamoDB endpoint (local for dev)

## 🚀 Deployment

### AWS Lambda Deployment

The application is designed to run on AWS Lambda with API Gateway:

1. **Build the application**

   ```bash
   GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/api/main.go
   ```

2. **Deploy using the provided script**
   ```bash
   ./scripts/deploy.sh
   ```

### Docker Deployment

For container-based deployment:

```bash
docker build -t treblesurf-backend .
docker run -p 8080:8080 treblesurf-backend
```

## 🔐 Authentication & Security

### Authentication Flow

1. **Google OAuth Integration**: Users authenticate via Google OAuth with support for both web and iOS client IDs
2. **Session Management**: JWT tokens and session cookies are issued for session management
3. **Protected Endpoints**: Most endpoints require valid authentication via middleware
4. **API Key Authentication**: Streaming services require API key authentication
5. **Development Mode**: Mock authentication available for local development

### Security Features

- **JWT Token Validation**: Secure token validation with configurable expiration
- **CSRF Protection**: CSRF tokens required for state-changing operations (production only)
- **Session Management**: Comprehensive session tracking with IP address and user agent logging
- **API Key Management**: Secure API key generation and management for streaming services
- **Role-based Access Control**: Admin endpoints require elevated privileges
- **CORS Configuration**: Environment-specific CORS policies for web and mobile clients
- **Secure Cookies**: HTTP-only and secure cookie settings for production
- **Session Termination**: Users can terminate individual sessions or all sessions

### Authentication Endpoints

- `POST /auth/google` - Google OAuth authentication with automatic user creation
- `GET /auth/validate` - Token validation with user data retrieval
- `POST /auth/logout` - Session termination
- `GET /auth/csrf` - CSRF token refresh (protected)
- `POST /auth/dev-session` - Development session creation (dev only)
- `GET /ws-token` - WebSocket authentication token

### Environment-Specific Behavior

- **Development**: Mock authentication, relaxed CORS, no CSRF requirements
- **Production**: Full Google OAuth validation, strict CORS, CSRF protection enabled

## 📊 Data Sources

- **Weather Data**: External weather APIs for forecasts
- **Buoy Data**: NOAA and other oceanographic data sources
- **Tide Information**: Tide prediction services
- **AI Swell Predictions**: Machine learning models for enhanced swell forecasting
- **User Reports**: Community-submitted surf conditions
- **Live Streams**: User-generated content from surf spots

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📝 License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).

### AGPL-3.0 License Terms

- **Copyleft**: Any derivative work must also be licensed under AGPL-3.0
- **Network Use**: If you run a modified version on a server and let other users communicate with it, you must make the source code available
- **Commercial Use**: Allowed, but you must comply with the copyleft provisions
- **Distribution**: You must provide the complete source code when distributing

### Full License Text

The complete license text is available at: https://www.gnu.org/licenses/agpl-3.0.en.html

**Important**: If you modify this software and run it on a server accessible to users over a network, you must provide users with access to the modified source code under the same license terms.

## 🆘 Support

For support and questions:

- Create an issue in the repository
- Contact the development team
- Check the documentation for common issues

## 🔄 API Versioning

The current API version is v1. All endpoints are prefixed with `/api` in production environments.

## 📈 Performance

- **Response Time**: Optimized for sub-100ms responses
- **Scalability**: Designed for AWS Lambda auto-scaling
- **Caching**: Implemented where appropriate for frequently accessed data
- **Rate Limiting**: Applied to prevent abuse

---

Built with ❤️ for the surfing community

## Media Upload Workflow

The backend supports multiple methods for submitting surf reports with media:

### 1. Legacy Method (Base64)

Submit reports with base64-encoded image data using the existing `/submitSurfReport` endpoint.

### 2. Presigned URL Method (Recommended)

Use presigned URLs to pre-upload media to S3, then submit reports with the media key.

#### Image Upload Workflow:

1. **Generate Upload URL**: Call `GET /generateImageUploadURL?country={country}&region={region}&spot={spot}`

   - Returns: `{ "uploadUrl": "...", "imageKey": "...", "expiresAt": "..." }`

2. **Upload Image**: Use the `uploadUrl` to upload the image directly to S3

   - The `imageKey` is predictable and calculable
   - URL expires in 15 minutes

3. **Submit Report**: Call `POST /submitSurfReportWithS3Image` with the report data including the `imageKey`
   - Backend retrieves image from S3
   - Validates image using Rekognition
   - If validation fails, image is automatically deleted from S3
   - If validation passes, report is stored with image reference

#### Video Upload Workflow:

1. **Generate Upload URL**: Call `GET /generateVideoUploadURL?country={country}&region={region}&spot={spot}`

   - Returns: `{ "uploadUrl": "...", "videoKey": "...", "expiresAt": "..." }`

2. **Upload Video**: Use the `uploadUrl` to upload the video directly to S3

3. **Submit Report**: Call `POST /submitSurfReportWithIOSValidation` with the report data including the `videoKey`

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

### 3. iOS Validation Method

Special endpoint for iOS apps with enhanced validation: `POST /submitSurfReportWithIOSValidation`

## Media Management Endpoints

### Surf Reports

- `POST /submitSurfReport` - Submit report with base64 image (legacy)
- `POST /submitSurfReportWithS3Image` - Submit report with S3 image key
- `POST /submitSurfReportWithIOSValidation` - Submit report with iOS validation
- `GET /generateImageUploadURL` - Generate presigned upload URL for images
- `GET /generateVideoUploadURL` - Generate presigned upload URL for videos
- `DELETE /deleteUploadedMedia` - Delete uploaded media files
- `GET /getTodaySpotReports` - Get today's reports for a spot
- `GET /getReportImage` - Get report image by key
- `GET /getReportVideo` - Get report video by key
- `GET /generateVideoViewURL` - Generate presigned view URL for videos

## Development

To run locally:

```bash
cd local
./scripts/setup.sh
go run cmd/server.go
```

The backend will start with mock services for local development.
