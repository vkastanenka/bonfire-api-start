package fields

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

func ValidateRequired(v, field, reason string) error {
	if v == "" {
		return ErrFieldRequired(field, reason)
	}
	return nil
}

func ValidateMinLen(v string, min int, field, title, reason string) error {
	if utf8.RuneCountInString(v) < min {
		detail := fmt.Sprintf("%s must be at least %d characters.", title, min)
		return NewError(field, title, detail, reason, "MIN_LENGTH_NOT_MET")
	}
	return nil
}

func ValidateMaxLen(v string, max int, field, title, reason string) error {
	if utf8.RuneCountInString(v) > max {
		detail := fmt.Sprintf("%s must not exceed %d characters.", title, max)
		return NewError(field, title, detail, reason, "MAX_LENGTH_EXCEEDED")
	}
	return nil
}

func ValidatePattern(v string, rgx *regexp.Regexp, field, title, reason, detail string) error {
	if !rgx.MatchString(v) {
		return NewError(field, title, detail, reason, "INVALID_FORMAT")
	}
	return nil
}
