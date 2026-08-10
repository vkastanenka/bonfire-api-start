package fields

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

type ValidateCfg struct {
	Field     string
	MinLen    int
	MaxLen    int
	Regex     *regexp.Regexp
	IsRuneLen bool
}

func Validate(s string, cfg ValidateCfg) error {
	var length int
	if cfg.IsRuneLen {
		length = utf8.RuneCountInString(s)
	} else {
		length = len(s)
	}

	if cfg.MinLen > 0 && length < cfg.MinLen {
		return ErrTooShort(cfg.Field, cfg.MinLen, fmt.Sprintf("Must be at least %d characters.", cfg.MinLen))
	}
	if cfg.MaxLen > 0 && length > cfg.MaxLen {
		return ErrTooLong(cfg.Field, cfg.MaxLen, fmt.Sprintf("Must not exceed %d characters.", cfg.MaxLen))
	}
	if cfg.Regex != nil && !cfg.Regex.MatchString(s) {
		return ErrInvalidFormat(cfg.Field, "Format is invalid.")
	}
	return nil
}
