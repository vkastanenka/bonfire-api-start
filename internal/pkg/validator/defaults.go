package validator

import (
	"reflect"
	"regexp"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

var (
	rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)
)

const (
	tagToken      = "token"
	tagTokenRules = "max=1024"
	tagHexColor   = "hexcolor"
)

// defaultTagNameFunc extracts key names from "json", "form", or "path" struct tags in priority order.
func defaultTagNameFunc(fld reflect.StructField) string {
	for _, tagKey := range []string{"json", "form", "path"} {
		if tag := fld.Tag.Get(tagKey); tag != "" && tag != "-" {
			if idx := strings.IndexByte(tag, ','); idx != -1 {
				return tag[:idx]
			}
			return tag
		}
	}
	return fld.Name
}

// defaultAliases returns package-level default tag aliases.
func defaultAliases() map[string]string {
	return map[string]string{
		tagToken: tagTokenRules,
	}
}

// defaultValidations returns package-level default custom validation functions.
func defaultValidations() map[string]goValidator.Func {
	return map[string]goValidator.Func{
		tagHexColor: validateHexColor,
	}
}

// validateHexColor validates that a field is an optional or non-empty 6-digit hex color code.
func validateHexColor(fl goValidator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	str := field.String()
	return str == "" || rgxHexColor.MatchString(str)
}
