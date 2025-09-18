# Media Cleanup Backend Implementation

## Overview

This document describes the implementation of media cleanup functionality for the Treble Surf backend. The system allows users to delete unused uploaded media (images and videos) from S3 storage, preventing storage waste and reducing costs.

## Implementation Details

### 1. New Endpoint

**Endpoint:** `DELETE /api/deleteUploadedMedia`

**Purpose:** Delete unused uploaded media from S3 storage.

**Authentication:** Requires user authentication (session cookie + CSRF token)

**Request Parameters:**

- `key` (query parameter): The S3 key of the media to delete
- `type` (query parameter): Either "image" or "video"

**Example Request:**

```
DELETE /api/deleteUploadedMedia?key=surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4&type=video
```

**Response:**

```json
{
  "message": "Media deleted successfully",
  "mediaKey": "surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4",
  "mediaType": "video"
}
```

### 2. Security Features

#### Access Control

- **User Authentication**: Requires valid session cookie
- **CSRF Protection**: Requires valid CSRF token for state-changing operations
- **Ownership Verification**: Users can only delete media they uploaded
- **Path Traversal Prevention**: Validates media key format to prevent directory traversal attacks

#### Input Validation

- **Required Parameters**: Both `key` and `type` parameters are mandatory
- **Media Type Validation**: Only accepts "image" or "video" as valid types
- **Key Format Validation**: Ensures media keys follow the expected pattern
- **File Extension Validation**: Validates against allowed file extensions

#### Media Key Format

Media keys must follow the pattern: `surf-reports/Country_Region_Spot/Timestamp_UUID.ext`

**Valid Extensions:**

- Images: `.jpg`, `.jpeg`, `.png`
- Videos: `.mp4`, `.mov`, `.avi`

### 3. Error Handling

The endpoint provides comprehensive error handling with helpful messages:

#### 400 Bad Request

- Missing required parameters
- Invalid media type
- Invalid media key format

#### 401 Unauthorized

- No valid session found
- Authentication required

#### 403 Forbidden

- User doesn't have permission to delete the media
- Access denied (ownership verification failed)

#### 404 Not Found

- Media file not found in S3
- Media may have already been deleted

#### 500 Internal Server Error

- User information retrieval failed
- S3 deletion failed
- General server errors

### 4. Implementation Files

#### Controller (`internal/controller/report_controller.go`)

- `DeleteUploadedMedia()`: Main handler function
- `isValidMediaKey()`: Validates media key format
- `canUserAccessMedia()`: Verifies user ownership

#### Service (`internal/service/report_service.go`)

- `DeleteMediaFromS3()`: Handles S3 deletion operations

#### Router (`internal/api/router.go`)

- Added DELETE route to `webModifyGroup` (requires authentication + CSRF)

### 5. Usage Examples

#### Frontend Integration (iOS Swift)

```swift
// Cleanup unused uploads when user cancels form
func cleanupUnusedUploads() {
    let mediaKeys = [uploadedImageKey, uploadedVideoKey, uploadedVideoThumbnailKey]

    for (key, type) in mediaKeys {
        guard let key = key, let type = type else { continue }

        Task {
            await deleteMedia(key: key, type: type)
        }
    }
}

private func deleteMedia(key: String, type: String) async {
    guard let url = URL(string: "\(baseURL)/api/deleteUploadedMedia?key=\(key)&type=\(type)") else { return }

    var request = URLRequest(url: url)
    request.httpMethod = "DELETE"
    request.setValue(csrfToken, forHTTPHeaderField: "X-CSRF-Token")

    do {
        let (_, response) = try await URLSession.shared.data(for: request)
        if let httpResponse = response as? HTTPURLResponse {
            print("Media deletion status: \(httpResponse.statusCode)")
        }
    } catch {
        print("Failed to delete media: \(error)")
    }
}
```

#### cURL Testing

```bash
# Delete an image
curl -X DELETE "https://your-api.com/api/deleteUploadedMedia?key=surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.jpg&type=image" \
  -H "Cookie: session_id=your-session-id" \
  -H "X-CSRF-Token: your-csrf-token"

# Delete a video
curl -X DELETE "https://your-api.com/api/deleteUploadedMedia?key=surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4&type=video" \
  -H "Cookie: session_id=your-session-id" \
  -H "X-CSRF-Token: your-csrf-token"
```

### 6. Logging

The implementation includes comprehensive logging:

```
=== Delete Uploaded Media Request ===
User-Agent: TrebleSurf/1.0 (iOS 17.0)
Method: DELETE
Content-Type: application/json
Deleting video media: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4 for user: user@example.com
🗑️ [CLEANUP] Deleting media: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4
✅ [CLEANUP] Successfully deleted media: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4
Successfully deleted video media: surf-reports/Ireland_Donegal_Ballyhiernan/2025-09-18T20:31:30Z_27ebc05e-625a-4c05-add5-5e6ef33f8b8e.mp4 for user: user@example.com
```

### 7. Benefits

#### Cost Reduction

- Prevents storage waste from unused uploads
- Reduces S3 storage costs
- Eliminates orphaned media files

#### User Experience

- Immediate cleanup when user cancels form
- Background cleanup without blocking UI
- Clear error messages for troubleshooting

#### Security

- Robust access control and ownership verification
- Protection against path traversal attacks
- CSRF protection for state-changing operations

### 8. Testing

#### Unit Tests

Test the validation functions:

- `isValidMediaKey()` with various inputs
- `canUserAccessMedia()` with different user UUIDs
- Error handling for invalid inputs

#### Integration Tests

Test the full endpoint:

- Valid deletion requests
- Authentication failures
- Authorization failures
- S3 deletion failures
- Network timeouts

#### Manual Testing

Use the provided cURL examples to test:

- Different media types (image/video)
- Various media key formats
- Authentication scenarios
- Error conditions

### 9. Future Enhancements

#### Backend Flag System

Consider implementing a database-backed cleanup system:

- Track uploaded media with confirmation flags
- Automatic cleanup of unconfirmed media after 24-48 hours
- More robust handling of edge cases (app crashes, network issues)

#### Batch Operations

Add support for batch media deletion:

- Delete multiple media files in one request
- Reduce API calls for bulk cleanup operations

#### Analytics

Track cleanup operations:

- Monitor storage savings
- Track cleanup frequency
- Identify patterns in unused uploads

## Conclusion

The media cleanup implementation provides a secure, efficient way to manage unused uploaded media. It integrates seamlessly with the existing authentication and authorization systems while providing comprehensive error handling and logging. The implementation follows Go best practices and maintains consistency with the existing codebase architecture.
