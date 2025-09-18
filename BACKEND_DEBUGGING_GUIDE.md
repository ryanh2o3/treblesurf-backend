# Backend Debugging Guide for iOS Video + Image Upload

## Issue Description

When uploading both image (thumbnail) and video from iOS app, only the video gets saved in the backend database. The image key is not being stored properly.

## Root Cause Analysis

The issue appears to be in the iOS validation flow where both `imageKey` and `videoKey` are provided, but the backend may not be processing both correctly.

## Changes Made

### 1. Added Debugging Logs

Added comprehensive logging to track the image and video processing:

```go
// In SubmitSurfReportWithIOSValidation function
log.Printf("iOS validated report with image: %s", report.ImageKey)
log.Printf("Set s3KeyReport to: %s", s3KeyReport)
log.Printf("iOS validated report with video: %s", report.VideoKey)
log.Printf("Set videoKeyReport to: %s", videoKeyReport)
log.Printf("WebSocket message - ImageKey: %s, VideoKey: %s, MediaType: %s", s3KeyReport, videoKeyReport, mediaType)
```

### 2. Enhanced Error Handling

Added logging when no image or video keys are provided to help identify the issue.

## Testing Steps

### 1. Test with Both Image and Video

Submit a report with both `imageKey` and `videoKey`:

```json
POST /api/submitSurfReportWithIOSValidation
{
  "country": "Ireland",
  "region": "Donegal",
  "spot": "Ballyhiernan",
  "imageKey": "surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:09Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.jpg",
  "videoKey": "surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:13Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4",
  "iosValidated": true,
  "surfSize": "chest-shoulder",
  "windAmount": "light",
  "windDirection": "offshore",
  "consistency": "consistent",
  "quality": "good",
  "messiness": "clean",
  "date": "2024-01-15 14:30:00"
}
```

### 2. Check Server Logs

Look for these log messages in the server output:

```
2025/09/18 20:04:49 iOS validated report with image: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:09Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.jpg
2025/09/18 20:04:49 Set s3KeyReport to: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:09Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.jpg
2025/09/18 20:04:49 iOS validated report with video: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:13Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4
2025/09/18 20:04:49 Set videoKeyReport to: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:13Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4
2025/09/18 20:04:49 WebSocket message - ImageKey: surf-reports/..., VideoKey: surf-reports/..., MediaType: both
```

### 3. Verify Database Storage

Check the DynamoDB table `SurfReports` to ensure both keys are stored:

```json
{
  "country_region_spot": "Ireland_Donegal_Ballyhiernan",
  "dateReported": "2025-09-18T20:04:49Z_user-uuid",
  "ImageKey": "surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:09Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.jpg",
  "VideoKey": "surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:04:13Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4",
  "MediaType": "both",
  "IOSValidated": true
  // ... other fields
}
```

### 4. Test Report Retrieval

Retrieve the report to verify both keys are returned:

```bash
GET /api/getTodaySpotReports?country=Ireland&region=Donegal&spot=Ballyhiernan
```

Expected response should include both `ImageKey` and `VideoKey`.

## Expected Behavior

### When Both Image and Video Are Provided:

- `MediaType` should be `"both"`
- `ImageKey` should contain the thumbnail S3 key
- `VideoKey` should contain the video S3 key
- Both should be stored in DynamoDB
- Both should be returned in API responses

### When Only Image Is Provided:

- `MediaType` should be `"image"`
- `ImageKey` should contain the image S3 key
- `VideoKey` should be empty string

### When Only Video Is Provided:

- `MediaType` should be `"video"`
- `VideoKey` should contain the video S3 key
- `ImageKey` should be empty string

## Troubleshooting

### If Image Key Is Still Missing:

1. **Check iOS App**: Ensure the iOS app is sending both `imageKey` and `videoKey` in the request
2. **Check Logs**: Look for "No image key provided" message
3. **Check Request**: Verify the JSON payload contains both keys
4. **Check Database**: Query DynamoDB directly to see what's actually stored

### If MediaType Is Wrong:

1. **Check Logic**: The media type determination logic should work correctly
2. **Check Logs**: Look for the WebSocket message log to see what values are being used

## Code Flow

1. **Request Received**: iOS sends both `imageKey` and `videoKey`
2. **Image Processing**: If `imageKey` exists, store it in `item["ImageKey"]` and set `s3KeyReport`
3. **Video Processing**: If `videoKey` exists, store it in `item["VideoKey"]` and set `videoKeyReport`
4. **Media Type**: Determine based on which keys are present
5. **Database Storage**: Store the complete item in DynamoDB
6. **WebSocket**: Broadcast with both keys

## Next Steps

1. **Deploy the updated code** with debugging logs
2. **Test with iOS app** using both image and video
3. **Check server logs** for the debugging output
4. **Verify database storage** contains both keys
5. **Test report retrieval** to ensure both keys are returned

If the issue persists after these changes, the problem may be in the iOS app not sending the `imageKey` properly, or there may be an issue with the DynamoDB storage logic that needs further investigation.
