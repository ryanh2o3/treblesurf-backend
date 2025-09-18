# iOS ML Validation and Video Upload Integration

This document outlines the new backend functionality for iOS machine learning validation and video uploads in the Treble Surf backend.

## Overview

The backend now supports:

- iOS client-side image validation using Vision framework
- Video upload and retrieval for surf reports
- Mixed media reports (both image and video)
- Backward compatibility with existing web and Android clients

## New API Endpoints

### 1. Submit iOS-Validated Surf Report

**Endpoint:** `POST /api/submitSurfReportWithIOSValidation`

**Authentication:** Required (JWT token)

**Description:** Submit a surf report that has been validated using iOS Vision framework. This endpoint bypasses server-side image validation.

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
  "videoKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4",
  "iosValidated": true,
  "date": "2024-01-15 14:30:00"
}
```

**Key Fields:**

- `iosValidated`: **Required** - Must be `true` for this endpoint
- `imageKey`: Optional - S3 key for pre-uploaded image
- `videoKey`: Optional - S3 key for pre-uploaded video
- At least one of `imageKey` or `videoKey` must be provided

**Response:**

```json
{
  "message": "Report submitted successfully"
}
```

**Error Responses:**

- `400` - Missing required fields, invalid iOS validation flag, or no media provided
- `401` - Authentication required
- `500` - Server error

### 2. Generate Video Upload URL

**Endpoint:** `GET /api/generateVideoUploadURL`

**Authentication:** Required (JWT token)

**Description:** Generate a presigned URL for uploading a video to S3.

**Query Parameters:**

- `country` (required): Country name
- `region` (required): Region name
- `spot` (required): Surf spot name

**Example Request:**

```
GET /api/generateVideoUploadURL?country=Ireland&region=Donegal&spot=Bundoran
```

**Response:**

```json
{
  "uploadUrl": "https://s3.amazonaws.com/treblesurf-images/surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4?X-Amz-Algorithm=...",
  "videoKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4",
  "expiresAt": "2024-01-15T14:45:00Z"
}
```

**Video Upload Process:**

1. Call this endpoint to get presigned URL
2. Upload video file directly to S3 using the presigned URL
3. Use the returned `videoKey` in your surf report submission

### 3. Get Report Video

**Endpoint:** `GET /api/getReportVideo`

**Authentication:** Required (JWT token)

**Description:** Retrieve a video associated with a surf report.

**Query Parameters:**

- `key` (required): S3 video key

**Example Request:**

```
GET /api/getReportVideo?key=surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4
```

**Response:**

```json
{
  "videoData": "base64-encoded-video-data",
  "contentType": "video/mp4"
}
```

## Updated Response Format

All surf report endpoints now include additional fields for video and iOS validation support:

**Surf Report Response:**

```json
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
  "ImageKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.jpg",
  "VideoKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4",
  "MediaType": "both",
  "IOSValidated": true
}
```

**New Fields:**

- `VideoKey`: S3 key for associated video (empty string if no video)
- `MediaType`: "image", "video", or "both" indicating what media is attached
- `IOSValidated`: Boolean indicating if report was validated on iOS

## iOS App Integration Workflow

### Complete Video Upload Flow

1. **Generate Video Upload URL**

   ```swift
   // GET /api/generateVideoUploadURL?country=Ireland&region=Donegal&spot=Bundoran
   let response = await generateVideoUploadURL(country: "Ireland", region: "Donegal", spot: "Bundoran")
   ```

2. **Upload Video to S3**

   ```swift
   // Upload video file using the presigned URL
   let success = await uploadVideoToS3(url: response.uploadUrl, videoData: videoData)
   ```

3. **Validate with Vision Framework**

   ```swift
   // Use iOS Vision framework to validate the video/image
   let isValid = await validateWithVision(videoData: videoData)
   ```

4. **Submit Validated Report**
   ```swift
   // POST /api/submitSurfReportWithIOSValidation
   let report = SurfReportWithIOSValidation(
       country: "Ireland",
       region: "Donegal",
       spot: "Bundoran",
       surfSize: "chest-shoulder",
       windAmount: "light",
       windDirection: "offshore",
       consistency: "consistent",
       quality: "good",
       messiness: "clean",
       imageKey: imageKey, // Optional
       videoKey: response.videoKey,
       iosValidated: true,
       date: "2024-01-15 14:30:00"
   )
   await submitIOSValidatedReport(report)
   ```

### Mixed Media Support

You can submit reports with both image and video:

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
  "videoKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4",
  "iosValidated": true,
  "date": "2024-01-15 14:30:00"
}
```

This will result in `MediaType: "both"` in the database and responses.

## Video Specifications

### Supported Formats

- MP4 (recommended)
- MOV
- AVI

### File Size Limits

- Maximum: 100MB per video
- Recommended: Under 50MB for better upload performance

### Upload Timeout

- Presigned URLs expire after 15 minutes
- Upload should be completed within this timeframe

## Error Handling

### Common Error Responses

**Missing iOS Validation:**

```json
{
  "error": "iOS validation required",
  "message": "This endpoint requires iOS validation to be set to true",
  "help": "Please use the iOS app to validate your surf report before submission."
}
```

**No Media Provided:**

```json
{
  "error": "No media provided",
  "message": "At least one image or video must be provided",
  "help": "Please provide either an imageKey or videoKey (or both)."
}
```

**Video Upload Failed:**

```json
{
  "error": "Video upload failed",
  "message": "Failed to upload the video to storage",
  "help": "Please try again in a moment. If the problem persists, contact support."
}
```

**Video Not Found:**

```json
{
  "error": "Video not found",
  "message": "The requested video could not be found or accessed",
  "help": "The video may have been deleted or the video key may be incorrect."
}
```

## Backward Compatibility

### Existing Endpoints

All existing endpoints continue to work unchanged:

- `POST /api/submitSurfReport` - Legacy image upload with validation
- `POST /api/submitSurfReportWithS3Image` - Pre-uploaded image with validation
- `GET /api/generateImageUploadURL` - Image upload URL generation
- `GET /api/getReportImage` - Image retrieval

### Response Compatibility

- Legacy reports get default values: `VideoKey: ""`, `MediaType: "image"`, `IOSValidated: false`
- Existing clients receive new fields with sensible defaults
- No breaking changes to existing functionality

## Testing Checklist

### iOS App Validation

**Video Upload Flow:**

- [ ] Generate video upload URL successfully
- [ ] Upload video to S3 using presigned URL
- [ ] Verify video exists in S3 after upload
- [ ] Submit report with video key
- [ ] Retrieve video using getReportVideo endpoint

**Image + Video Flow:**

- [ ] Upload both image and video
- [ ] Submit report with both imageKey and videoKey
- [ ] Verify MediaType is set to "both"
- [ ] Verify both media types are retrievable

**iOS Validation:**

- [ ] Submit report with iosValidated: true
- [ ] Verify server skips image validation
- [ ] Submit report with iosValidated: false (should fail)
- [ ] Verify proper error handling

**Error Scenarios:**

- [ ] Missing required fields
- [ ] Invalid video format
- [ ] Video too large
- [ ] Expired presigned URL
- [ ] Network failures

**Backward Compatibility:**

- [ ] Existing image upload still works
- [ ] Legacy reports include new fields
- [ ] Web/Android clients unaffected

## Database Schema

### New DynamoDB Fields

**SurfReports Table:**

- `VideoKey` (String, optional): S3 key for associated video
- `MediaType` (String): "image", "video", or "both"
- `IOSValidated` (Boolean): Indicates iOS validation

**Example Item:**

```json
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
  "ImageKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.jpg",
  "VideoKey": "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_user-uuid.mp4",
  "MediaType": "both",
  "IOSValidated": true
}
```

## Security Considerations

- Presigned URLs expire after 15 minutes
- iOS validation trusts the presigned URL upload process (no additional S3 verification)
- iOS validation flag is required for the new endpoint
- All uploads require authentication
- S3 bucket policies remain unchanged

**Note:** For iOS-validated reports, the server trusts that files uploaded via presigned URLs exist and are accessible, avoiding additional S3 permission requirements.

## Performance Notes

- Video uploads use direct S3 upload (no server processing)
- Base64 encoding for video retrieval (consider streaming for large files)
- WebSocket broadcasting includes video information
- Database queries include new optional fields

## Support

For issues or questions regarding the iOS integration:

1. Check error messages for specific guidance
2. Verify authentication tokens are valid
3. Ensure video files meet format and size requirements
4. Confirm presigned URLs haven't expired

The implementation maintains full backward compatibility while adding robust video and iOS validation support for enhanced surf reporting capabilities.
