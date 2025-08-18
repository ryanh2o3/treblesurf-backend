package utils

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}



func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
    merged := make(map[string]interface{})
    for _, m := range maps {
        for k, v := range m {
            merged[k] = v
        }
    }
    return merged
}
