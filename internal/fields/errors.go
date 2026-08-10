package fields

import (
	"bonfire-api/internal/errs"
	"fmt"
	"strings"
)

func NewError(field, title, detail, reason, rule string) error {
	if field == "" {
		field = "field"
	}
	if title == "" {
		title = field
	}
	if reason == "" {
		reason = fmt.Sprintf("%s_%s", strings.ToUpper(field), rule)
	}
	return errs.InvalidArgument(fmt.Sprintf("Invalid %s.", title)).
		Reason(reason).
		FieldViolation(field, detail, rule)
}

func ErrFieldRequired(field, reason string) error {
	return NewError(field, field, "Field cannot be empty or zero-value", reason, "REQUIRED")
}
