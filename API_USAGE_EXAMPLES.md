# API Usage Examples

## New Image Upload Workflow (Presigned URLs)

### 1. Generate Upload URL

**Request:**

```http
GET /generateImageUploadURL?country=Ireland&region=Donegal&spot=Rossnowlagh
```

**Response:**

```json
{
  "uploadUrl": "https://treblesurf-images.s3.amazonaws.com/surf-reports/Ireland_Donegal_Rossnowlagh/2024-01-15T14:30:00Z/user@example.com.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "imageKey": "surf-reports/Ireland_Donegal_Rossnowlagh/2024-01-15T14:30:00Z/user@example.com.jpg",
  "expiresAt": "2024-01-15T14:45:00Z"
}
```

### 2. Upload Image to S3

**Request:**

```http
PUT <uploadUrl>
Content-Type: image/jpeg
Body: <binary_image_data>
```

**Notes:**

- Use the `uploadUrl` from step 1
- Set appropriate `Content-Type` header
- Upload the raw image bytes
- No authentication needed (presigned URL handles this)

### 3. Submit Report with Image Key

**Request:**

```http
POST /submitSurfReportWithS3Image
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "country": "Ireland",
  "region": "Donegal",
  "spot": "Rossnowlagh",
  "surfSize": "3-4ft",
  "windAmount": "Light",
  "windDirection": "Offshore",
  "consistency": "Good",
  "quality": "Good",
  "messiness": "Clean",
  "imageKey": "surf-reports/Ireland_Donegal_Rossnowlagh/2024-01-15T14:30:00Z/user@example.com.jpg",
  "date": "2024-01-15 14:30:00"
}
```

**Response:**

```json
{
  "message": "Report submitted successfully"
}
```

## Legacy Workflow (Base64 - Still Supported)

### Submit Report with Base64 Image

**Request:**

```http
POST /submitSurfReport
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "country": "Ireland",
  "region": "Donegal",
  "spot": "Rossnowlagh",
  "surfSize": "3-4ft",
  "windAmount": "Light",
  "windDirection": "Offshore",
  "consistency": "Good",
  "quality": "Good",
  "messiness": "Clean",
  "imageData": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...",
  "date": "2024-01-15 14:30:00"
}
```

## iOS App Implementation Example

```swift
class SurfReportService {

    func uploadImageAndSubmitReport(country: String, region: String, spot: String, image: UIImage, reportData: [String: Any]) async throws {

        // Step 1: Generate upload URL
        let uploadResponse = try await generateUploadURL(country: country, region: region, spot: spot)

        // Step 2: Upload image to S3
        try await uploadImageToS3(uploadURL: uploadResponse.uploadURL, image: image)

        // Step 3: Submit report with image key
        var finalReportData = reportData
        finalReportData["imageKey"] = uploadResponse.imageKey

        try await submitReportWithS3Image(reportData: finalReportData)
    }

    private func generateUploadURL(country: String, region: String, spot: String) async throws -> PresignedUploadResponse {
        // Implementation to call /generateImageUploadURL
    }

    private func uploadImageToS3(uploadURL: String, image: UIImage) async throws {
        // Implementation to upload image using presigned URL
    }

    private func submitReportWithS3Image(reportData: [String: Any]) async throws {
        // Implementation to call /submitSurfReportWithS3Image
    }
}
```

## Error Handling

### Image Validation Failures

If an image fails Rekognition validation:

- The image is automatically deleted from S3
- The report submission fails with an error
- No orphaned files are left in S3

### Network Failures During Upload

If the image upload fails:

- The report submission will fail when trying to retrieve the image
- The image key will be cleaned up automatically

### Database Failures

If the report fails to save to the database:

- The uploaded image is automatically deleted from S3
- No orphaned files are left

## Benefits of New Workflow

1. **Performance**: No base64 encoding/decoding overhead
2. **Memory**: Reduced memory usage on both client and server
3. **Reliability**: Better error handling and automatic cleanup
4. **Scalability**: Direct S3 uploads reduce server load
5. **User Experience**: iOS app can upload images in background
6. **Caching**: Predictable image keys enable better caching strategies
