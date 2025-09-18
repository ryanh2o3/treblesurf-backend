# Video View URL Backend Implementation

## Overview

This implementation adds a new endpoint `/api/generateVideoViewURL` that generates presigned URLs for viewing videos stored in S3. This approach is more efficient and secure for video playback compared to fetching raw video data.

## Implementation Details

### New Endpoint

**Endpoint:** `GET /api/generateVideoViewURL?key={videoKey}`

**Authentication:** Requires user authentication (session cookie + CSRF token)

**Request Example:**

```
GET /api/generateVideoViewURL?key=surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4
```

**Response Example:**

```json
{
  "viewURL": "https://your-s3-bucket.s3.amazonaws.com/surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4?AWSAccessKeyId=...&Signature=...&Expires=...",
  "expiresAt": "2025-09-18T21:31:30Z"
}
```

### Security Features

1. **User Authentication**: Requires valid user session
2. **Access Control**: Users can only view videos from their own surf reports
3. **Video Key Validation**: Extracts UUID from video key to verify ownership
4. **URL Expiration**: Presigned URLs expire after 1 hour
5. **Video Existence Check**: Verifies video exists in S3 before generating URL

### Error Handling

- **400 Bad Request**: Missing video key parameter
- **401 Unauthorized**: User not authenticated or user not found
- **403 Forbidden**: User doesn't have permission to view the video
- **404 Not Found**: Video doesn't exist or is not accessible
- **500 Internal Server Error**: Server error during URL generation

### Files Modified

1. **internal/storage/s3.go**: Added `GeneratePresignedViewURL` method
2. **internal/model/report.go**: Added `VideoViewURLResponse` struct
3. **internal/service/report_service.go**: Added `GenerateVideoViewURL` method and `canUserAccessVideo` helper
4. **internal/controller/report_controller.go**: Added `GenerateVideoViewURL` controller function
5. **internal/api/router.go**: Added route for the new endpoint

### Usage

The iOS app should:

1. Call `/api/generateVideoViewURL?key={videoKey}` when user taps play button
2. Receive presigned URL and expiration time
3. Use `AVPlayer` to stream video directly from S3
4. Handle URL expiration gracefully by requesting a new URL

### Benefits

- **Performance**: No need to download entire video file
- **Security**: URLs expire automatically and include access control
- **Scalability**: S3 handles video delivery, reducing server load
- **Bandwidth**: Reduced server bandwidth usage

### Testing

Test the endpoint with:

```bash
curl -X GET "https://your-api.com/api/generateVideoViewURL?key=test-video-key" \
  -H "Cookie: session_id=your-session-id" \
  -H "X-CSRF-Token: your-csrf-token"
```

Expected response:

```json
{
  "viewURL": "https://s3.amazonaws.com/...",
  "expiresAt": "2025-09-18T21:31:30Z"
}
```
