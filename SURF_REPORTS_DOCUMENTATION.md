# Surf Reports API Documentation

This document provides comprehensive documentation on how surf reports are handled from API calls and how image validation works in the Treble Surf backend system.

## Table of Contents

1. [Overview](#overview)
2. [API Endpoints](#api-endpoints)
3. [Data Models](#data-models)
4. [Image Upload Workflows](#image-upload-workflows)
5. [Image Validation System](#image-validation-system)
6. [Error Handling](#error-handling)
7. [Validation Rules](#validation-rules)
8. [Database Storage](#database-storage)
9. [WebSocket Integration](#websocket-integration)
10. [Development vs Production](#development-vs-production)

## Overview

The surf reports system allows authenticated users to submit surf condition reports for specific surf spots. The system supports two image upload methods and includes comprehensive image validation using AWS Rekognition to ensure images are surf-related.

## API Endpoints

### 1. Submit Surf Report with Base64 Image

**Endpoint:** `POST /submitSurfReport`

**Authentication:** Required (JWT token)

**Description:** Submit a surf report with image data embedded as base64 in the request body.

**Request Body:**

```json
{
  "country": "Ireland",
  "region": "Donegal",
  "spot": "Bundoran",
  "surfSize": "chest-shoulder",
  "windAmount": "light",
  "windDirection": "offshore",
  "consistency": "consistent",
  "quality": "good",
  "messiness": "clean",
  "imageData": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...",
  "date": "2024-01-15 14:30:00"
}
```

**Response:**

```json
{
  "message": "Report submitted successfully"
}
```

### 2. Submit Surf Report with S3 Image

**Endpoint:** `POST /submitSurfReportWithS3Image`

**Authentication:** Required (JWT token)

**Description:** Submit a surf report with a pre-uploaded S3 image key.

**Request Body:**

```json
{
  "country": "Ireland",
  "region": "Donegal",
  "spot": "Bundoran",
  "surfSize": "chest-shoulder",
  "windAmount": "light",
  "windDirection": "offshore",
  "consistency": "consistent",
  "quality": "good",
  "messiness": "clean",
  "imageKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.jpg",
  "date": "2024-01-15 14:30:00"
}
```

### 3. Generate Image Upload URL

**Endpoint:** `GET /generateImageUploadURL`

**Authentication:** Required (JWT token)

**Description:** Generate a presigned URL for uploading an image to S3.

**Query Parameters:**

- `country` (required): Country name
- `region` (required): Region name
- `spot` (required): Surf spot name

**Response:**

```json
{
  "uploadUrl": "https://s3.amazonaws.com/bucket/surf-reports/...",
  "imageKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.jpg",
  "expiresAt": "2024-01-15T14:45:00Z"
}
```

### 4. Retrieve Today's Surf Reports

**Endpoint:** `GET /getTodaySpotReports`

**Authentication:** Required (JWT token)

**Description:** Get surf reports for a specific spot.

**Query Parameters:**

- `country` (required): Country name
- `region` (required): Region name
- `spot` (required): Surf spot name

**Response:**

```json
[
  {
    "country_region_spot": "Ireland_Donegal_Bundoran",
    "dateReported": "2024-01-15T14:30:00Z_user-uuid",
    "SurfSize": "chest-shoulder",
    "WindAmount": "light",
    "WindDirection": "offshore",
    "Consistency": "consistent",
    "Quality": "good",
    "Messiness": "clean",
    "Reporter": "John Doe",
    "Time": "2024-01-15T14:30:00Z",
    "reportedBy": "user-uuid",
    "ImageKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.jpg"
  }
]
```

### 5. Get Report Image

**Endpoint:** `GET /getReportImage`

**Authentication:** Required (JWT token)

**Description:** Retrieve an image associated with a surf report.

**Query Parameters:**

- `key` (required): S3 image key

**Response:**

```json
{
  "imageData": "base64-encoded-image-data",
  "contentType": "image/jpeg"
}
```

## Data Models

### ReportWithImage

Used for base64 image uploads:

```go
type ReportWithImage struct {
    Country       string `json:"country"`
    Region        string `json:"region"`
    Spot          string `json:"spot"`
    SurfSize      string `json:"surfSize"`
    WindAmount    string `json:"windAmount"`
    WindDirection string `json:"windDirection"`
    Consistency   string `json:"consistency"`
    Quality       string `json:"quality"`
    Messiness     string `json:"messiness"`
    ImageData     string `json:"imageData"` // Base64 encoded
    Date          string `json:"date"`
}
```

### ReportWithS3Image

Used for S3 pre-uploaded images:

```go
type ReportWithS3Image struct {
    Country       string `json:"country"`
    Region        string `json:"region"`
    Spot          string `json:"spot"`
    SurfSize      string `json:"surfSize"`
    WindAmount    string `json:"windAmount"`
    WindDirection string `json:"windDirection"`
    Consistency   string `json:"consistency"`
    Quality       string `json:"quality"`
    Messiness     string `json:"messiness"`
    ImageKey      string `json:"imageKey"` // S3 key
    Date          string `json:"date"`
}
```

## Image Upload Workflows

### Workflow 1: Base64 Image Upload

1. **Client** sends POST request to `/submitSurfReport` with base64 image data
2. **Controller** validates request format and required fields
3. **Service** extracts and decodes base64 image data
4. **Service** validates image using AWS Rekognition
5. **Service** uploads image to S3 with generated key
6. **Service** stores report data in DynamoDB
7. **Service** broadcasts update via WebSocket
8. **Controller** returns success response

### Workflow 2: S3 Pre-upload

1. **Client** requests presigned URL from `/generateImageUploadURL`
2. **Service** generates S3 key and presigned URL (15-minute expiry)
3. **Client** uploads image directly to S3 using presigned URL
4. **Client** sends POST request to `/submitSurfReportWithS3Image` with S3 key
5. **Service** retrieves image from S3 for validation
6. **Service** validates image using AWS Rekognition
7. **Service** stores report data in DynamoDB
8. **Service** broadcasts update via WebSocket
9. **Controller** returns success response

## Image Validation System

### AWS Rekognition Integration

The system uses AWS Rekognition to validate that uploaded images are surf-related. The validation process:

1. **Image Analysis:** Rekognition analyzes the image and detects labels
2. **Confidence Threshold:** Only labels with 90%+ confidence are considered
3. **Valid Labels:** Images must contain at least one of:
   - "Sea"
   - "Water"
   - "Sea Waves"
   - "Beach"
   - "Coast"

### Validation Logic

```go
func (s *ReportService) validateImageWithRekognition(imageData []byte) (bool, error) {
    if os.Getenv("GO_ENV") == "development" {
        // In development, always return true to allow all images
        return true, nil
    }

    input := &rekognition.DetectLabelsInput{
        Image: &rekognition.Image{
            Bytes: imageData,
        },
        MinConfidence: aws.Float64(90.0),
    }

    result, err := s.rekognitionClient.DetectLabels(input)
    if err != nil {
        return false, model.NewImageValidationError(err, "image analysis failed")
    }

    validLabels := []string{"Sea", "Water", "Sea Waves", "Beach", "Coast"}

    for _, label := range result.Labels {
        for _, validLabel := range validLabels {
            if strings.EqualFold(*label.Name, validLabel) {
                return true, nil
            }
        }
    }

    return false, model.ErrImageNotSurfRelated
}
```

### Image Processing

- **Base64 Decoding:** Handles both raw base64 and data URI formats
- **EXIF Extraction:** Extracts timestamp from image EXIF data if no date provided
- **Format Support:** JPEG and PNG formats supported
- **S3 Storage:** Images stored with structured keys: `surf-reports/{country_region_spot}/{timestamp}_{user-uuid}.jpg`

## Error Handling

### Image Validation Errors

```go
var (
    ErrImageNotSurfRelated = errors.New("image does not appear to be surf-related")
    ErrImageAnalysisFailed = errors.New("image analysis failed")
    ErrImageUploadFailed   = errors.New("image upload failed")
    ErrInvalidImageData    = errors.New("invalid image data")
    ErrImageValidationFailed = errors.New("image validation failed")
    ErrImageRetrievalFailed = errors.New("failed to retrieve pre-uploaded image")
)
```

### Error Response Format

All API errors follow a consistent format:

```json
{
  "error": "Error type",
  "message": "Human-readable error message",
  "help": "Actionable guidance for the user"
}
```

### Common Error Scenarios

1. **Invalid Image Data:**

   ```json
   {
     "error": "Invalid image data",
     "message": "The image data provided is not in a valid format",
     "help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) and that the image data is properly encoded."
   }
   ```

2. **Image Not Surf-Related:**

   ```json
   {
     "error": "Image not surf-related",
     "message": "The image does not appear to show surf conditions",
     "help": "Please upload a photo that clearly shows the ocean, waves, beach, or coastline."
   }
   ```

3. **Missing Required Fields:**
   ```json
   {
     "error": "Missing required fields",
     "message": "Country, region, and spot are required",
     "help": "Please provide all required location information."
   }
   ```

## Validation Rules

### Surf Size Validation

```go
validSizes := []string{
    "flat",
    "knee-waist",
    "chest-shoulder",
    "head-high",
    "overhead",
    "double-overhead"
}
```

### Wind Amount Validation

```go
validAmounts := []string{
    "light",
    "moderate",
    "strong",
    "very-strong"
}
```

### Wind Direction Validation

```go
validDirections := []string{
    "onshore",
    "offshore",
    "cross-shore",
    "no-wind"
}
```

### Consistency Validation

```go
validDifficulties := []string{
    "setty",
    "consistent",
    "inconsistent",
    "sporadic"
}
```

### Quality Validation

```go
validConditions := []string{
    "mushy",
    "average",
    "okay",
    "good",
    "excellent"
}
```

### Messiness Validation

```go
validMessiness := []string{
    "clean",
    "slight-chop",
    "choppy",
    "messy"
}
```

## Database Storage

### DynamoDB Table: SurfReports

**Primary Key:**

- Partition Key: `country_region_spot` (e.g., "Ireland_Donegal_Bundoran")
- Sort Key: `dateReported` (e.g., "2024-01-15T14:30:00Z_user-uuid")

**Attributes:**

- `SurfSize`: Wave height description
- `WindAmount`: Wind strength
- `WindDirection`: Wind direction relative to shore
- `Consistency`: Wave consistency
- `Quality`: Overall surf quality
- `Messiness`: Water surface condition
- `UserEmail`: Reporter's email (stored but not returned to clients)
- `Reporter`: Reporter's display name
- `Time`: Report timestamp
- `reportedBy`: User UUID
- `ImageKey`: S3 key for associated image (optional)

### S3 Storage

**Bucket Structure:**

```
treblesurf-images/
└── surf-reports/
    └── {country_region_spot}/
        └── {timestamp}_{user-uuid}.jpg
```

**Example:**

```
treblesurf-images/surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_abc123.jpg
```

## WebSocket Integration

When a new surf report is submitted, the system broadcasts updates to connected WebSocket clients:

```go
message := map[string]interface{}{
    "action": "new_report",
    "data": map[string]interface{}{
        "country": report.Country,
        "region":  report.Region,
        "spot":    report.Spot,
        "time":    currentTime.String(),
        "reporter": userName,
    },
}
```

## Development vs Production

### Development Mode

- **Image Validation:** Bypassed - all images are accepted
- **Authentication:** Uses mock authentication middleware
- **CSRF Protection:** Disabled
- **Environment Variable:** `GO_ENV=development`

### Production Mode

- **Image Validation:** Full AWS Rekognition validation enabled
- **Authentication:** JWT token validation required
- **CSRF Protection:** Enabled for web requests
- **Environment Variable:** `GO_ENV=production`

## Security Considerations

1. **Authentication Required:** All surf report endpoints require valid JWT authentication
2. **Input Validation:** Comprehensive validation of all input fields
3. **Image Validation:** AWS Rekognition prevents non-surf images
4. **S3 Security:** Presigned URLs expire after 15 minutes
5. **Data Privacy:** User emails are not returned in API responses
6. **CSRF Protection:** Web requests protected against CSRF attacks

## Performance Considerations

1. **Image Processing:** Base64 images are processed in memory
2. **S3 Upload:** Direct S3 uploads reduce server load
3. **Rekognition:** 90% confidence threshold balances accuracy and performance
4. **Database Queries:** Optimized DynamoDB queries with proper indexing
5. **WebSocket Broadcasting:** Asynchronous message broadcasting

## Monitoring and Logging

The system includes comprehensive logging for:

- Request/response details
- Image validation results
- S3 upload operations
- Database operations
- Error conditions
- WebSocket message broadcasting

All logs include relevant context such as user email, image keys, and error details for effective debugging and monitoring.
