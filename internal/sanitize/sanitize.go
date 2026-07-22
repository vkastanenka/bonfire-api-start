package sanitize

import (
	"reflect"
	"strings"
	"unicode"
)

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

		if fieldVal.Kind() == reflect.Struct {
			if fieldVal.CanAddr() {
				Normalize(fieldVal.Addr().Interface())
			}
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

			directives := strings.Split(tag, ",")
			for _, d := range directives {
				switch strings.TrimSpace(d) {
				case "email":
					str = Email(str)
				case "text":
					str = Text(str)
				}
			}

			targetVal.SetString(str)
		}
	}
}

func Email(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func Text(input string) string {
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
