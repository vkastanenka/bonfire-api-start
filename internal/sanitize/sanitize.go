package sanitize

import (
	"strings"
	"unicode"
)

func SanitizeEmail(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func SanitizeText(input string) string {
	input = strings.TrimSpace(input)

	var sb strings.Builder
	var lastWasSpace bool

	for _, runeValue := range input {
		if unicode.Is(unicode.Cc, runeValue) || unicode.Is(unicode.Cf, runeValue) {
			continue
		}

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
