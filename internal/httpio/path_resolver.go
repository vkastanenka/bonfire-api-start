package httpio

import "unicode"

// resolveUnmarshalPath maps internal Go sub-struct names to their external
// camelCase JSON pathways to avoid leaking raw Go struct types to the client.
func resolveUnmarshalPath(structName, fieldName string) string {
	if structName == "" {
		return fieldName
	}

	// Map your internal Go struct types to their parent JSON keys.
	// As Bonfire grows, add your nested sub-structs here.
	switch structName {
	case "ProfileInfo", "ProfileData":
		return "profileInfo." + fieldName
	case "UserSettings", "Settings":
		return "settings." + fieldName
	case "ChannelPermissions":
		return "permissions." + fieldName
	default:
		// Fallback: lowercase the internal struct name as a sensible default
		// if you forget to add a explicit mapping later.
		if len(structName) > 0 {
			runes := []rune(structName)
			runes[0] = unicode.ToLower(runes[0]) // Native, clean rune translation
			return string(runes) + "." + fieldName
		}
		return fieldName
	}
}
