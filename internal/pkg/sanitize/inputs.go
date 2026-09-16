package sanitize

import (
	"bytes"
	"net/url"
	"strings"
	"unicode"
)

// Text trims spaces, removes control characters, and collapses whitespace.
func Text(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	return cleanRunes(trimmed)
}

// Bytes applies Text cleaning guarantees to byte slices.
func Bytes(raw []byte) []byte {
	trimmed := bytes.TrimSpace(bytes.Trim(bytes.TrimSpace(raw), `"`))
	if len(trimmed) == 0 {
		return nil
	}

	cleaned := cleanRunes(string(trimmed))
	return []byte(cleaned)
}

// Email normalizes email inputs by cleaning text and lowercasing.
func Email(input string) string {
	return strings.ToLower(Text(input))
}

// URL lowercases the scheme and hostname without altering paths or queries.
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

// cleanRunes strips control characters and collapses consecutive whitespace runs.
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
