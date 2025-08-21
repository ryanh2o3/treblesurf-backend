# Frontend Quick Reference - Error Handling

## 🚀 What Changed

- **Before**: String-based error matching, duplicate messages, inconsistent responses
- **After**: Type-safe error handling, consistent format, clear user guidance

## 📋 Error Response Format

```json
{
  "error": "Error Type",
  "message": "What went wrong",
  "help": "How to fix it"
}
```

## 🔑 Key Error Types to Handle

### Image Validation Errors

| Error Type                | HTTP Status | User Message                                      |
| ------------------------- | ----------- | ------------------------------------------------- |
| `Image not surf-related`  | 400         | "Please upload a photo showing ocean/waves/beach" |
| `Invalid image data`      | 400         | "Please ensure valid image format (JPEG/PNG)"     |
| `Image upload failed`     | 500         | "Upload failed, please try again"                 |
| `Image validation failed` | 400         | "Image validation failed, check surf content"     |
| `Image not found`         | 400         | "Image not found, please re-upload"               |

## 💻 Implementation Example

```typescript
const handleApiError = (response: Response) => {
  if (!response.ok) {
    const errorData = await response.json();

    switch (errorData.error) {
      case "Image not surf-related":
        showError(errorData.message, errorData.help);
        break;
      case "Invalid image data":
        showError(errorData.message, errorData.help);
        break;
      default:
        showError("An error occurred", "Please try again");
    }
  }
};
```

## 🧪 Test These Scenarios

1. **Non-surf image** → Should get "Image not surf-related"
2. **Invalid base64** → Should get "Invalid image data"
3. **Valid surf image** → Should succeed

## 📱 User Experience Tips

- Show error message prominently
- Display help text below error
- Provide clear next steps
- Use consistent error styling

## 🆘 Need Help?

- Check `API_ERROR_HANDLING.md` for full details
- Contact backend team with specific error details
- Test with the examples above
