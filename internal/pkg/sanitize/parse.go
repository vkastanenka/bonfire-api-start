package sanitize

import (
	"strconv"

	"bonfire-api/internal/pkg/errs"
)

// ParseEnumInt validates integer inputs against enum upper bounds (0 < raw < max).
func ParseEnumInt[T ~int](raw int, max T, fieldName string) (T, error) {
	val := T(raw)
	if raw <= 0 || val >= max {
		return 0, errs.InvalidArgument("invalid "+fieldName).
			Reason("INVALID_"+Enum(fieldName)).
			Meta("value", strconv.Itoa(raw)).
			FieldViolation(fieldName, "provided integer value is out of valid enum bounds", "INVALID_ENUM_VALUE")
	}
	return val, nil
}

// ParseEnumIntBytes converts a byte slice containing integer digits ("1")
// into an enum value bounded by max (0 < val < max).
func ParseEnumIntBytes[T ~int](b []byte, max T, fieldName string) (T, error) {
	s := string(b)
	raw, err := strconv.Atoi(s)
	if err != nil {
		return 0, errs.InvalidArgument("invalid "+fieldName).
			Reason("INVALID_"+Enum(fieldName)).
			Meta("value", s).
			FieldViolation(fieldName, "provided byte value is not a valid integer", "INVALID_ENUM_VALUE")
	}

	return ParseEnumInt(raw, max, fieldName)
}

// ParseEnumString sanitizes a string input and parses it into an enum value.
func ParseEnumString[T ~int](input string, names []string, fieldName string) (T, error) {
	return parseEnumNormalized[T](Enum(input), input, names, fieldName)
}

// ParseEnumStringBytes sanitizes a byte slice (including JSON strings) and parses it into an enum value.
func ParseEnumStringBytes[T ~int](b []byte, names []string, fieldName string) (T, error) {
	return parseEnumNormalized[T](EnumBytes(b), string(b), names, fieldName)
}

// parseEnumNormalized performs the lookup and error generation on an already-sanitized string.
func parseEnumNormalized[T ~int](cleaned string, original string, names []string, fieldName string) (T, error) {
	if cleaned == "" {
		return 0, errs.InvalidArgument(fieldName+" is required").
			Reason("EMPTY_"+Enum(fieldName)).
			Meta("value", original).
			FieldViolation(fieldName, "value cannot be empty", "REQUIRED_FIELD")
	}

	for i, name := range names {
		if name == cleaned {
			return T(i), nil
		}
	}

	return 0, errs.InvalidArgument("invalid "+fieldName).
		Reason("INVALID_"+Enum(fieldName)).
		Meta("value", original).
		FieldViolation(fieldName, "provided value is not supported", "INVALID_ENUM_VALUE")
}
