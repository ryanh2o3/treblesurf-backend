# API Error Handling Documentation

This document outlines the enhanced error handling system for the Treble Surf Backend API, specifically for surf report submissions and image validation.

## Overview

All API endpoints now return structured error responses with three key fields:

- `error`: A short, machine-readable error identifier
- `message`: A human-readable description of what went wrong
- `help`: Actionable guidance for the user to resolve the issue

## Error Response Format

```json
{
  "error": "Error Type",
  "message": "Detailed description of the error",
  "help": "Specific guidance on how to fix the issue"
}
```

## HTTP Status Codes

- `400 Bad Request`: Client-side errors (validation failures, missing data)
- `401 Unauthorized`: Authentication required
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side errors

## Surf Report Submission Errors

### 1. Submit Current Surf Report (`POST /api/reports/submit`)

#### Missing Required Fields

```json
{
  "error": "Missing required fields",
  "message": "Country, region, and spot are required",
  "help": "Please provide all required location information."
}
```

**Frontend Action**: Highlight missing fields and show the help message.

#### Invalid Surf Size

```json
{
  "error": "Invalid surf size",
  "message": "Surf size 'huge' is not valid",
  "help": "Valid surf sizes are: flat, 0-0.5, 0.5-1, 1-1.5, 1.5-2.5, 2.5+"
}
```

**Frontend Action**: Show validation error next to the surf size field with the help message.

#### Invalid Wind Amount

```json
{
  "error": "Invalid wind amount",
  "message": "Wind amount 'hurricane' is not valid",
  "help": "Valid wind amounts are: calm, light, moderate, strong"
}
```

**Frontend Action**: Show validation error next to the wind amount field with the help message.

#### Invalid Wind Direction

```json
{
  "error": "Invalid wind direction",
  "message": "Wind direction 'north' is not valid",
  "help": "Valid wind directions are: glassy, offshore, cross, onshore"
}
```

**Frontend Action**: Show validation error next to the wind direction field with the help message.

#### Invalid Consistency

```json
{
  "error": "Invalid consistency",
  "message": "Consistency 'random' is not valid",
  "help": "Valid consistency values are: lulls, consistent, relentless"
}
```

**Frontend Action**: Show validation error next to the consistency field with the help message.

#### Invalid Quality

```json
{
  "error": "Invalid quality",
  "message": "Quality 'amazing' is not valid",
  "help": "Valid quality values are: glassy, clean, messy, okay"
}
```

**Frontend Action**: Show validation error next to the quality field with the help message.

#### Invalid Messiness

```json
{
  "error": "Invalid messiness",
  "message": "Messiness 'perfect' is not valid",
  "help": "Valid messiness values are: glassy, clean, messy, okay"
}
```

**Frontend Action**: Show validation error next to the messiness field with the help message.

#### Image Validation Failure

```json
{
  "error": "Image validation failed",
  "message": "image does not appear to be surf-related. Detected: Car, Road, Building. Please upload a photo showing the ocean, waves, beach, or coastline",
  "help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. The image should be clear and focused on surf conditions."
}
```

**Frontend Action**:

- Show the error message prominently
- Display the help text as guidance
- Optionally show the detected labels from the message
- Provide a retry option or image selection guidance

#### Invalid Image Data

```json
{
  "error": "Invalid image data",
  "message": "The image data provided is not in a valid format",
  "help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) and that the image data is properly encoded."
}
```

**Frontend Action**:

- Show file format requirements
- Provide guidance on supported image types
- Offer to let user select a different image

#### Authentication Required

```json
{
  "error": "Authentication required",
  "message": "You must be logged in to submit a surf report",
  "help": "Please log in and try again."
}
```

**Frontend Action**:

- Redirect to login page
- Show the help message
- Store the form data for after login

#### User Information Error

```json
{
  "error": "User information error",
  "message": "Unable to retrieve your user profile",
  "help": "Please try again in a moment. If the problem persists, contact support."
}
```

**Frontend Action**:

- Show retry button
- Display the help message
- Optionally show contact support link

#### Invalid Request Format

```json
{
  "error": "Invalid request format",
  "message": "The request data is not in the correct format",
  "help": "Please ensure you're sending valid JSON data with all required fields."
}
```

**Frontend Action**:

- This is typically a frontend bug - log the error
- Show generic error message to user
- Check form validation

### 2. Submit Surf Report with S3 Image (`POST /api/reports/submit-s3`)

#### S3 Image Validation Failure

```json
{
  "error": "S3 Image validation failed",
  "message": "S3 image validation failed: image does not appear to be surf-related. Detected: Car, Road, Building. Please upload a photo showing the ocean, waves, beach, or coastline",
  "help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. The image should be clear and focused on surf conditions."
}
```

**Frontend Action**:

- Show the error message prominently
- Display the help text as guidance
- The S3 image will be automatically cleaned up
- Provide option to upload a new image

#### Image Not Found

```json
{
  "error": "Image not found",
  "message": "The uploaded image could not be found or accessed",
  "help": "Please try uploading your image again. If the problem persists, contact support."
}
```

**Frontend Action**:

- Show the error message
- Provide option to re-upload the image
- Show contact support option if problem persists

### 3. Generate Image Upload URL (`GET /api/reports/upload-url`)

#### Missing Required Parameters

```json
{
  "error": "Missing required parameters",
  "message": "Country, region, and spot parameters are required",
  "help": "Please provide all required location parameters in your request."
}
```

**Frontend Action**:

- Ensure all location fields are filled before requesting upload URL
- Show validation errors for missing fields

#### Failed to Generate Upload URL

```json
{
  "error": "Failed to generate upload URL",
  "message": "Unable to create a secure upload link for your image",
  "help": "Please try again in a moment. If the problem persists, contact support."
}
```

**Frontend Action**:

- Show retry button
- Display the help message
- Show contact support option if problem persists

### 4. Get Report Image (`GET /api/reports/image`)

#### Missing Image Key

```json
{
  "error": "Missing image key",
  "message": "Image key parameter is required",
  "help": "Please provide the image key in your request."
}
```

**Frontend Action**:

- This is typically a frontend bug - log the error
- Show generic error message to user

#### Image Not Found

```json
{
  "error": "Image not found",
  "message": "The requested image could not be found or accessed",
  "help": "The image may have been deleted or the image key may be incorrect."
}
```

**Frontend Action**:

- Show placeholder image or error icon
- Display the help message
- Optionally show "Image not available" message

### 5. Retrieve Today's Surf Reports (`GET /api/reports/today`)

#### Missing Required Parameters

```json
{
  "error": "Missing required parameters",
  "message": "Country, region, and spot parameters are required",
  "help": "Please provide all required location parameters in your request."
}
```

**Frontend Action**:

- Ensure all location fields are filled before making the request
- Show validation errors for missing fields

#### Failed to Retrieve Reports

```json
{
  "error": "Failed to retrieve reports",
  "message": "Unable to fetch surf reports from the database",
  "help": "Please try again in a moment. If the problem persists, contact support."
}
```

**Frontend Action**:

- Show retry button
- Display the help message
- Show contact support option if problem persists

## Frontend Implementation Guidelines

### 1. Error Display

```typescript
interface ApiError {
  error: string;
  message: string;
  help: string;
}

function displayError(error: ApiError) {
  // Show error banner/notification
  showErrorBanner({
    title: error.error,
    message: error.message,
    help: error.help,
  });

  // Log error for debugging
  console.error("API Error:", error);
}
```

### 2. Field-Specific Validation

```typescript
function handleFieldValidationError(error: ApiError, fieldName: string) {
  // Extract field name from error message if possible
  const field = extractFieldFromError(error.message);

  // Show validation error next to the specific field
  showFieldError(field || fieldName, error.help);

  // Optionally highlight the field
  highlightField(field || fieldName);
}
```

### 3. Retry Logic

```typescript
async function submitWithRetry(data: any, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await submitReport(data);
    } catch (error) {
      if (i === maxRetries - 1) throw error;

      // Wait before retrying
      await new Promise((resolve) => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}
```

### 4. User Guidance

```typescript
function showUserGuidance(error: ApiError) {
  // Show help text prominently
  showHelpText(error.help);

  // Provide actionable buttons based on error type
  if (error.error.includes("validation")) {
    showValidationHelp();
  } else if (error.error.includes("authentication")) {
    showLoginPrompt();
  } else if (error.error.includes("image")) {
    showImageGuidance();
  }
}
```

### 5. Image Upload Handling

```typescript
async function handleImageUpload(file: File) {
  try {
    // Validate file type and size
    if (!isValidImageFile(file)) {
      throw new Error("Invalid image file");
    }

    // Upload to S3
    const uploadUrl = await getUploadUrl();
    await uploadToS3(uploadUrl, file);

    // Submit report with image key
    await submitReportWithS3Image(reportData);
  } catch (error) {
    // Handle specific image errors
    if (error.error?.includes("validation")) {
      showImageValidationError(error);
    } else if (error.error?.includes("upload")) {
      showUploadError(error);
    }
  }
}
```

## Error Recovery Strategies

### 1. Image Validation Failures

- **Automatic cleanup**: S3 images are automatically deleted when validation fails
- **User guidance**: Clear instructions on what types of images are acceptable
- **Retry option**: Allow users to upload a different image immediately

### 2. Authentication Issues

- **Redirect to login**: Automatically redirect unauthenticated users
- **Form preservation**: Store form data for after login
- **Clear messaging**: Explain why authentication is required

### 3. Network/Server Issues

- **Retry mechanism**: Implement exponential backoff for retries
- **User feedback**: Show progress indicators and retry buttons
- **Fallback options**: Provide alternative submission methods if possible

### 4. Validation Errors

- **Field highlighting**: Clearly indicate which fields have errors
- **Inline help**: Show validation rules and examples
- **Progressive validation**: Validate fields as user types

## Testing Error Scenarios

### 1. Image Validation Testing

- Upload non-surf images (cars, buildings, etc.)
- Upload corrupted image files
- Test with various image formats (JPEG, PNG, GIF)
- Verify cleanup of invalid S3 images

### 2. Input Validation Testing

- Submit forms with missing required fields
- Test invalid values for each field type
- Verify error messages are field-specific
- Test boundary conditions

### 3. Authentication Testing

- Submit requests without authentication
- Test with expired/invalid tokens
- Verify proper redirect behavior
- Test form data preservation

### 4. Network Error Testing

- Simulate network timeouts
- Test with slow connections
- Verify retry mechanisms work
- Test error recovery flows

## Support and Debugging

### 1. Error Logging

- Log all API errors with full context
- Include user information (when available)
- Log request/response data for debugging
- Monitor error patterns and frequencies

### 2. User Support

- Provide clear contact information
- Include error codes in support requests
- Offer alternative submission methods
- Maintain user-friendly error messages

### 3. Monitoring

- Track error rates by endpoint
- Monitor image validation success rates
- Alert on unusual error patterns
- Track user experience metrics

This enhanced error handling system provides a much better user experience by giving clear, actionable feedback when things go wrong, while maintaining detailed logging for debugging and support purposes.
