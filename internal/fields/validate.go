package fields

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

type ValidateCfg struct {
	MinLen    int
	MaxLen    int
	Regex     *regexp.Regexp
	IsRuneLen bool
}

func Validate(field, s string, cfg ValidateCfg) error {
	length := len(s)
	if cfg.IsRuneLen {
		length = utf8.RuneCountInString(s)
	}

	if cfg.MinLen > 0 && length < cfg.MinLen {
		return ErrTooShort(field, fmt.Sprintf("Must be at least %d characters.", cfg.MinLen))
	}
	if cfg.MaxLen > 0 && length > cfg.MaxLen {
		return ErrTooLong(field, fmt.Sprintf("Must not exceed %d characters.", cfg.MaxLen))
	}
	if cfg.Regex != nil && !cfg.Regex.MatchString(s) {
		return ErrInvalidFormat(field, "Format is invalid.")
	}
	return nil
}
