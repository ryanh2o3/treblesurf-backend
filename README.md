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
- **Real-time Conditions**: Current weather and ocean conditions
- **Buoy Data**: Live ocean buoy measurements and historical data
- **Surf Reports**: Community-driven surf condition reports with image uploads
- **Live Streaming**: Real-time surf spot streaming with snapshot capture
- **Location Management**: Surf spots, regions, and coordinate systems
- **User Management**: Authentication, sessions, and user preferences

### Technical Features

- **RESTful API**: Comprehensive HTTP endpoints
- **WebSocket Support**: Real-time data streaming
- **Image Processing**: Surf report and snapshot image handling
- **API Key Management**: Secure access control for streaming services
- **Multi-environment Support**: Development and production configurations
- **CORS Support**: Cross-origin resource sharing for web applications

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
- `GET /individualBuoyLocation` - Get specific buoy location

### User Management (Protected)

- `GET /sessions` - Get user sessions
- `GET /getTheme` - Get user theme preference
- `PUT /setTheme` - Set user theme preference
- `DELETE /deleteMyAccount` - Delete user account
- `DELETE /sessions/:sessionId` - Terminate specific session

### Surf Reports (Protected)

- `POST /submitSurfReport` - Submit a new surf report
- `GET /getTodaySpotReports` - Get today's surf reports for a spot
- `GET /getReportImage` - Get image for a surf report

### Streaming Services (Protected)

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

### Forecast

```go
type Forecast struct {
    Location          string    `json:"location"`
    Country           string    `json:"country"`
    Region            string    `json:"region"`
    Spot              string    `json:"spot"`
    Date              time.Time `json:"date"`
    DateForecastedFor string    `json:"dateForecastedFor"`
    Hour              int       `json:"hour"`
    Temperature       float64   `json:"temperature"`
    WindSpeed         float64   `json:"wind_speed"`
    WindDirection     float64   `json:"wind_direction"`
    WaveHeight        float64   `json:"wave_height"`
    WavePeriod        float64   `json:"wave_period"`
    WaveDirection     float64   `json:"wave_direction"`
    Conditions        string    `json:"conditions"`
}
```

### Surf Report

```go
type SurfReport struct {
    ID              string    `json:"id"`
    UserEmail      string    `json:"user_email"`
    Country        string    `json:"country"`
    Region         string    `json:"region"`
    Spot           string    `json:"spot"`
    Timestamp      time.Time `json:"timestamp"`
    SwellSize      string    `json:"swell_size"`
    WindAmount     string    `json:"wind_amount"`
    WindDirection  string    `json:"wind_direction"`
    SurfConditions string    `json:"surf_conditions"`
    SurfDifficulty string    `json:"surf_difficulty"`
    ImageKey       string    `json:"image_key,omitempty"`
    Notes          string    `json:"notes,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

### Buoy Data

```go
type BuoyData struct {
    BuoyName      string    `json:"buoy_name"`
    Timestamp     time.Time `json:"timestamp"`
    WaveHeight    float64   `json:"wave_height"`
    WavePeriod    float64   `json:"wave_period"`
    WaveDirection float64   `json:"wave_direction"`
    WindSpeed     float64   `json:"wind_speed"`
    WindDirection float64   `json:"wind_direction"`
    Temperature   float64   `json:"temperature"`
    Pressure      float64   `json:"pressure"`
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

1. Users authenticate via Google OAuth
2. JWT tokens are issued for session management
3. Protected endpoints require valid authentication
4. API keys are used for streaming service access

### Security Features

- JWT token validation
- CSRF protection (production only)
- API key management for streaming services
- Session management and termination
- Role-based access control (admin endpoints)

## 📊 Data Sources

- **Weather Data**: External weather APIs for forecasts
- **Buoy Data**: NOAA and other oceanographic data sources
- **Tide Information**: Tide prediction services
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

## Image Upload Workflow

The backend now supports two methods for submitting surf reports with images:

### 1. Legacy Method (Base64)
Submit reports with base64-encoded image data using the existing `/submitSurfReport` endpoint.

### 2. New Presigned URL Method (Recommended for iOS)
Use presigned URLs to pre-upload images to S3, then submit reports with the image key.

#### Workflow:
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

#### Benefits:
- Better performance (no base64 encoding/decoding)
- Reduced memory usage
- Better error handling and cleanup
- iOS app can upload images in background
- Predictable image keys for caching

#### Error Handling:
- Invalid images are automatically cleaned up from S3
- Orphaned images can be cleaned up using the cleanup methods
- Database failures trigger image cleanup to prevent orphaned files

## API Endpoints

### Surf Reports
- `POST /submitSurfReport` - Submit report with base64 image (legacy)
- `POST /submitSurfReportWithS3Image` - Submit report with S3 image key (new)
- `GET /generateImageUploadURL` - Generate presigned upload URL
- `GET /getTodaySpotReports` - Get today's reports for a spot
- `GET /getReportImage` - Get report image by key

## Development

To run locally:
```bash
cd local
./scripts/setup.sh
go run cmd/server.go
```

The backend will start with mock services for local development.
