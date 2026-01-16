package store

import "strings"

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to mask Redis URL for logging (hides passwords)
func maskRedisURL(url string) string {
	// Hide password in Redis URL for logging
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			// Mask the auth part
			authHost := parts[0]
			hostPart := parts[1]

			// Check if auth contains password
			if strings.Contains(authHost, ":") {
				authParts := strings.Split(authHost, ":")
				if len(authParts) >= 2 {
					// Keep username, mask password
					authParts[1] = "***"
					authHost = strings.Join(authParts, ":")
				}
			}

			return authHost + "@" + hostPart
		}
	}
	return url
}
