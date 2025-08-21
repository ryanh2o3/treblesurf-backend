# API Error Handling Documentation

## Overview

The backend has been updated with a new, simplified error handling system for image validation and surf report submission. This eliminates duplicate error messages and provides consistent, type-safe error responses.

## Key Changes

### 1. Custom Error Types

All image-related errors now use specific error types instead of generic error messages:

- `ErrImageNotSurfRelated` - Image doesn't show surf conditions
- `ErrImageAnalysisFailed` - AWS Rekognition analysis failed
- `ErrImageUploadFailed` - S3 upload failed
- `ErrInvalidImageData` - Malformed image data
- `ErrImageValidationFailed` - General validation failure
- `ErrImageRetrievalFailed` - S3 retrieval failed

### 2. Consistent Error Response Format

All error responses now follow this structure:

```json
{
  "error": "Short error title",
  "message": "Detailed error description",
  "help": "User-friendly guidance on how to resolve the issue"
}
```

## API Endpoints & Error Handling

### Submit Surf Report (`POST /submitSurfReport`)

**Success Response:**

```json
{
  "message": "Report submitted successfully"
}
```

**Error Responses:**

#### Image Not Surf-Related

```json
{
  "error": "Image not surf-related",
  "message": "The image does not appear to show surf conditions",
  "help": "Please upload a photo that clearly shows the ocean, waves, beach, or coastline."
}
```

**HTTP Status:** `400 Bad Request`

#### Invalid Image Data

```json
{
  "error": "Invalid image data",
  "message": "The image data provided is not in a valid format",
  "help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) and that the image data is properly encoded."
}
```

**HTTP Status:** `400 Bad Request`

#### Image Upload Failed

```json
{
  "error": "Image upload failed",
  "message": "Failed to upload the image to storage",
  "help": "Please try again in a moment. If the problem persists, contact support."
}
```

**HTTP Status:** `500 Internal Server Error`

#### Image Validation Failed (Generic)

```json
{
  "error": "Image validation failed",
  "message": "[Specific error message from backend]",
  "help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. The image should be clear and focused on surf conditions."
}
```

**HTTP Status:** `400 Bad Request`

### Submit Surf Report with S3 Image (`POST /submitSurfReportWithS3Image`)

**Success Response:**

```json
{
  "message": "Report submitted successfully"
}
```

**Error Responses:**

#### Image Not Found

```json
{
  "error": "Image not found",
  "message": "The uploaded image could not be found or accessed",
  "help": "Please try uploading your image again. If the problem persists, contact support."
}
```

**HTTP Status:** `400 Bad Request`

#### All other image-related errors follow the same pattern as above

## Frontend Implementation Guide

### 1. Error Handling Pattern

```typescript
// Example error handling for surf report submission
const submitReport = async (reportData: ReportData) => {
  try {
    const response = await fetch("/api/submitSurfReport", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(reportData),
    });

    if (!response.ok) {
      const errorData = await response.json();

      // Handle specific error types
      switch (errorData.error) {
        case "Image not surf-related":
          // Show user-friendly message about surf-related content
          showError(errorData.message, errorData.help);
          break;

        case "Invalid image data":
          // Show message about image format/encoding
          showError(errorData.message, errorData.help);
          break;

        case "Image upload failed":
          // Show retry message with support contact
          showError(errorData.message, errorData.help);
          break;

        case "Image validation failed":
          // Show general validation guidance
          showError(errorData.message, errorData.help);
          break;

        default:
          // Handle unexpected errors
          showError(
            "An unexpected error occurred",
            "Please try again or contact support."
          );
      }
      return;
    }

    const result = await response.json();
    showSuccess(result.message);
  } catch (error) {
    // Handle network/other errors
    showError("Network error", "Please check your connection and try again.");
  }
};
```

### 2. User Experience Improvements

#### Before (Old System)

- Generic error messages
- Duplicate error text
- Inconsistent error handling
- String-based error matching

#### After (New System)

- Specific, actionable error messages
- Consistent error format
- Type-safe error handling
- Clear user guidance

### 3. Error Message Display

```typescript
interface ErrorDisplay {
  showError: (message: string, help?: string) => void;
  showSuccess: (message: string) => void;
}

const showError = (message: string, help?: string) => {
  // Display error message prominently
  // Show help text below if provided
  // Provide clear next steps for user
};

const showSuccess = (message: string) => {
  // Display success message
  // Clear form if applicable
  // Redirect or show next steps
};
```

## Image Validation Requirements

### What Constitutes a Valid Surf Image

The backend uses AWS Rekognition to validate that images contain surf-related content. Valid images should show:

- Ocean/Sea
- Water
- Sea Waves
- Beach
- Coast

### Image Format Requirements

- **Supported formats:** JPEG, PNG
- **Encoding:** Base64 (with or without data URI prefix)
- **Size:** Reasonable file sizes (backend handles compression)

### Development vs Production

- **Development:** All images are automatically accepted (bypasses Rekognition)
- **Production:** Full image validation using AWS Rekognition

## Testing Error Scenarios

### 1. Test Invalid Image Data

```typescript
// Send malformed base64 data
const invalidImageData = "invalid-base64-string";
const reportData = {
  ...validReportData,
  imageData: invalidImageData,
};
// Should return "Invalid image data" error
```

### 2. Test Non-Surf Image

```typescript
// Send image of something other than surf
const nonSurfImage = "base64-encoded-image-of-cat";
const reportData = {
  ...validReportData,
  imageData: nonSurfImage,
};
// Should return "Image not surf-related" error
```

### 3. Test Valid Surf Image

```typescript
// Send image of ocean/waves
const surfImage = "base64-encoded-image-of-ocean";
const reportData = {
  ...validReportData,
  imageData: surfImage,
};
// Should succeed
```

## Migration Notes

### Breaking Changes

- Error response format is now consistent across all endpoints
- Error messages are more specific and actionable
- HTTP status codes may differ for some error conditions

### Backward Compatibility

- All existing API endpoints remain the same
- Request format is unchanged
- Only error response format has been improved

## Support & Troubleshooting

### Common Issues

1. **Image validation failures** - Ensure images clearly show surf conditions
2. **Upload failures** - Check image format and size
3. **Network errors** - Verify API endpoint accessibility

### Debugging

- Check browser network tab for exact error responses
- Verify image data format and encoding
- Ensure proper authentication headers are sent

### Getting Help

- Check this documentation first
- Review error response details
- Contact backend team with specific error details
