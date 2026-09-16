package validator

const (
	errValidationFailed       = "Validation failed for the request."
	errInvalidConstraintValue = "Invalid value for constraint: %s"
	errRequired               = "This field is required."
	errWhitespace             = "Cannot consist entirely of whitespace."
	errEmail                  = "Must be a valid email address."
	errAlphanum               = "Must contain only letters and numbers."
	errHexColor               = "Must be a valid hex code (e.g., #FF5733)."
	errUUID                   = "Must be a valid UUID."
	errURL                    = "Must be a valid URL."
	errMinString              = "Must be at least %s characters."
	errMinNumeric             = "Must be %s or greater."
	errMinCollection          = "Must contain at least %s items."
	errMaxString              = "Cannot be longer than %s characters."
	errMaxNumeric             = "Must be %s or less."
	errMaxCollection          = "Cannot contain more than %s items."
)
