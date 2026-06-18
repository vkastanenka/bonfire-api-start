package sanitize

import (
	"strings"
	"unicode"
)

// SanitizeEmail trims spaces and makes emails lowercase
func SanitizeEmail(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

// SanitizeText strips out harmful control characters and normalizes spaces
func SanitizeText(input string) string {
	// Trim leading and trailing whitespace first
	input = strings.TrimSpace(input)

	var sb strings.Builder
	var lastWasSpace bool

	for _, runeValue := range input {
		// Strip out invisible control characters, zero-width spaces, and formatting characters
		if unicode.Is(unicode.Cc, runeValue) || unicode.Is(unicode.Cf, runeValue) {
			continue
		}

		// Collapse multiple consecutive spaces into a single space
		if unicode.IsSpace(runeValue) {
			if !lastWasSpace {
				sb.WriteRune(' ')
				lastWasSpace = true
			}
			continue
		}

		sb.WriteRune(runeValue)
		lastWasSpace = false
	}

	return sb.String()
}
