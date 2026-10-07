package parse

import (
	"strconv"

	"bonfire-api/internal/pkg/sanitize"
)

// EnumInt validates integer inputs against enum upper bounds (0 < raw < max).
func EnumInt[T ~int](raw int, max T, fieldName string) (T, error) {
	val := T(raw)
	if raw <= 0 || val >= max {
		return 0, ErrValueInvalid(fieldName, strconv.Itoa(raw))
	}
	return val, nil
}

// EnumIntBytes converts a byte slice containing integer digits ("1")
// into an enum value bounded by max (0 < val < max).
func EnumIntBytes[T ~int](b []byte, max T, fieldName string) (T, error) {
	raw, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, ErrIntegerInvalid(fieldName, string(b))
	}

	return EnumInt(raw, max, fieldName)
}

// EnumString sanitizes a string input and parses it into an enum value.
func EnumString[T ~int](input string, names []string, fieldName string) (T, error) {
	cleaned := sanitize.Enum(input)
	if cleaned == "" {
		return 0, ErrFieldRequired(fieldName, input)
	}

	for i, name := range names {
		if name == cleaned {
			return T(i), nil
		}
	}

	return 0, ErrValueInvalid(fieldName, input)
}

// EnumStringBytes sanitizes a byte slice (including JSON strings) and parses it into an enum value.
func EnumStringBytes[T ~int](b []byte, names []string, fieldName string) (T, error) {
	cleaned := sanitize.EnumBytes(b)
	if len(cleaned) == 0 {
		return 0, ErrFieldRequired(fieldName, string(b))
	}

	for i, name := range names {
		if name == cleaned {
			return T(i), nil
		}
	}

	return 0, ErrValueInvalid(fieldName, string(b))
}
