package cmd

import "strings"

// normalizePluginName removes special characters and normalizes for matching
func normalizePluginName(name string) string {
	// Remove special characters like ®, ™, ©, etc.
	result := strings.Builder{}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}

	// Replace spaces with dashes and convert to lowercase
	normalized := strings.ReplaceAll(result.String(), " ", "-")
	return strings.ToLower(normalized)
}
