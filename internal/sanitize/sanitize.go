package sanitize

import (
	"bytes"
	"reflect"
	"strings"
	"unicode"
)

// String removes surrounding quotes, whitespace, and non-printable control characters.
func String(s string) string {
	s = strings.TrimSpace(strings.Trim(s, `"`))
	if s == "" {
		return ""
	}

	// Pre-scan to avoid allocation if s is already clean
	needsCleaning := false
	for _, r := range s {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			needsCleaning = true
			break
		}
	}
	if !needsCleaning {
		return s
	}

	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		if !unicode.Is(unicode.Cc, r) && !unicode.Is(unicode.Cf, r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// String removes surrounding quotes, whitespace, and non-printable control characters.
func EnumValue(s string) string {
	return strings.ToUpper(String(s))
}

// Bytes removes surrounding quotes, whitespace, and control characters from byte slices.
func Bytes(raw []byte) []byte {
	return bytes.TrimSpace(bytes.Trim(raw, `"`))
}

func Email(input string) string {
	return strings.ToLower(String(input))
}

func Text(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	// Fast path: Scan if modification is needed
	needsAlloc := false
	var lastWasSpace bool
	for _, r := range trimmed {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) || (unicode.IsSpace(r) && lastWasSpace) {
			needsAlloc = true
			break
		}
		lastWasSpace = unicode.IsSpace(r)
	}

	if !needsAlloc {
		return trimmed
	}

	var sb strings.Builder
	sb.Grow(len(trimmed))
	lastWasSpace = false

	for _, r := range trimmed {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				sb.WriteRune(' ')
				lastWasSpace = true
			}
			continue
		}
		sb.WriteRune(r)
		lastWasSpace = false
	}

	return sb.String()
}

func Phone(input string) string {
	s := String(input)
	if s == "" {
		return ""
	}

	hasPlus := strings.HasPrefix(s, "+")
	var sb strings.Builder
	sb.Grow(len(s))

	for _, r := range s {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}

	if sb.Len() == 0 {
		return ""
	}
	if hasPlus {
		return "+" + sb.String()
	}
	return sb.String()
}

func URL(input string) string {
	s := String(input)
	if s == "" {
		return ""
	}

	if idx := strings.Index(s, "://"); idx != -1 {
		scheme := strings.ToLower(s[:idx])
		return scheme + s[idx:]
	}
	return s
}

func UUID(input string) string {
	return strings.ToLower(String(input))
}

// Normalize modifies struct fields in-place using "mod" tags zero-alloc.
func Normalize(s any) {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		if fieldVal.Kind() == reflect.Struct && fieldVal.CanAddr() {
			Normalize(fieldVal.Addr().Interface())
			continue
		}
		if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			Normalize(fieldVal.Interface())
			continue
		}

		tag := fieldType.Tag.Get("mod")
		if tag == "" {
			continue
		}

		targetVal := fieldVal
		if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.String {
			targetVal = fieldVal.Elem()
		}

		if targetVal.Kind() == reflect.String && targetVal.CanSet() {
			str := targetVal.String()
			for tag != "" {
				var dir string
				if idx := strings.IndexByte(tag, ','); idx != -1 {
					dir, tag = tag[:idx], tag[idx+1:]
				} else {
					dir, tag = tag, ""
				}

				switch strings.TrimSpace(dir) {
				case "email":
					str = Email(str)
				case "text":
					str = Text(str)
				case "uuid":
					str = UUID(str)
				case "phone":
					str = Phone(str)
				case "url":
					str = URL(str)
				}
			}
			targetVal.SetString(str)
		}
	}
}
