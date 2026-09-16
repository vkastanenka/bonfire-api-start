package validator

import (
	"reflect"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

// defaultTagNameFunc checks json, form, and path tags in sequence for field naming.
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

const (
	TagToken      = "token"
	TagTokenRules = "max=1024"
	TagHexColor   = "hexcolor"
	TagVerCode    = "vercode"
)

// defaultAliases maps shared infrastructure-level aliases using constants.
func defaultAliases() map[string]string {
	return map[string]string{
		TagToken: TagTokenRules,
	}
}

// defaultValidations maps package-level validation functions using constants.
func defaultValidations() map[string]goValidator.Func {
	return map[string]goValidator.Func{
		TagHexColor: validateHexColor,
		TagVerCode:  validateVerCode,
	}
}

func validateHexColor(fl goValidator.FieldLevel) bool {
	str := fl.Field().String()
	return str == "" || rgxHexColor.MatchString(str)
}

func validateVerCode(fl goValidator.FieldLevel) bool {
	str := fl.Field().String()
	return str == "" || rgxVerCode.MatchString(str)
}
