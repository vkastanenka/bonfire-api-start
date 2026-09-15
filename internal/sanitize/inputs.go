package sanitize

import (
	"bytes"
	"net/url"
	"strings"
	"unicode"
)

// Text cleans string inputs by removing control/format characters and collapsing whitespace.
func Text(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	return cleanRunes(trimmed)
}

// Bytes applies the same cleaning guarantees as Text to byte slices.
func Bytes(raw []byte) []byte {
	trimmed := bytes.TrimSpace(bytes.Trim(bytes.TrimSpace(raw), `"`))
	if len(trimmed) == 0 {
		return nil
	}

	cleaned := cleanRunes(string(trimmed))
	return []byte(cleaned)
}

// Email normalizes email inputs by stripping control characters, collapsing spaces, and lowercasing.
func Email(input string) string {
	return strings.ToLower(Text(input))
}

// URL normalizes scheme and hostname casing without corrupting paths or query strings.
func URL(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}

	u, err := url.Parse(s)
	if err != nil {
		return s
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	return u.String()
}

// cleanRunes performs two-pass sanitization: pre-scans for allocation necessity,
// then builds a sanitized string if control chars or whitespace runs exist.
func cleanRunes(s string) string {
	var needsAlloc, lastWasSpace bool

	for _, r := range s {
		if isIgnored(r) {
			needsAlloc = true
			break
		}
		if unicode.IsSpace(r) {
			if lastWasSpace {
				needsAlloc = true
				break
			}
			lastWasSpace = true
			continue
		}
		lastWasSpace = false
	}

	if !needsAlloc {
		return s
	}

	var sb strings.Builder
	sb.Grow(len(s))
	lastWasSpace = false

	for _, r := range s {
		if isIgnored(r) {
			continue
		}
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				sb.WriteByte(' ')
				lastWasSpace = true
			}
			continue
		}
		sb.WriteRune(r)
		lastWasSpace = false
	}

	return sb.String()
}

// isIgnored checks for non-printable Unicode Control (Cc) and Format (Cf) characters.
func isIgnored(r rune) bool {
	return unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r)
}
