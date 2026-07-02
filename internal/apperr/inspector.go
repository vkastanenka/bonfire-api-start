package apperr

// IsBadRequest checks if the error resolves to a CodeBadRequest.
func IsBadRequest(err error) bool { return IsCode(err, CodeBadRequest) }

// IsInvalidInput checks if the error resolves to a CodeInvalidInput.
func IsInvalidInput(err error) bool { return IsCode(err, CodeInvalidInput) }

// IsUnauthorized checks if the error resolves to a CodeUnauthorized.
func IsUnauthorized(err error) bool { return IsCode(err, CodeUnauthorized) }

// IsForbidden checks if the error resolves to a CodeForbidden.
func IsForbidden(err error) bool { return IsCode(err, CodeForbidden) }

// IsNotFound checks if the error resolves to a CodeNotFound.
func IsNotFound(err error) bool { return IsCode(err, CodeNotFound) }

// IsMethodNotAllowed checks if the error resolves to a CodeMethodNotAllowed.
func IsMethodNotAllowed(err error) bool { return IsCode(err, CodeMethodNotAllowed) }

// IsConflict checks if the error resolves to a CodeConflict.
func IsConflict(err error) bool { return IsCode(err, CodeConflict) }

// IsGone checks if the error resolves to a CodeGone.
func IsGone(err error) bool { return IsCode(err, CodeGone) }

// IsPreconditionFailed checks if the error resolves to a CodePreconditionFailed.
func IsPreconditionFailed(err error) bool { return IsCode(err, CodePreconditionFailed) }

// IsPayloadTooLarge checks if the error resolves to a CodePayloadTooLarge.
func IsPayloadTooLarge(err error) bool { return IsCode(err, CodePayloadTooLarge) }

// IsUnsupportedMediaType checks if the error resolves to a CodeUnsupportedMediaType.
func IsUnsupportedMediaType(err error) bool { return IsCode(err, CodeUnsupportedMediaType) }

// IsUnprocessableEntity checks if the error resolves to a CodeUnprocessableEntity.
func IsUnprocessableEntity(err error) bool { return IsCode(err, CodeUnprocessableEntity) }

// IsTooManyRequests checks if the error resolves to a CodeTooManyRequests.
func IsTooManyRequests(err error) bool { return IsCode(err, CodeTooManyRequests) }

// IsInternal checks if the error resolves to a CodeInternal.
func IsInternal(err error) bool { return IsCode(err, CodeInternal) }

// IsNotImplemented checks if the error resolves to a CodeNotImplemented.
func IsNotImplemented(err error) bool { return IsCode(err, CodeNotImplemented) }

// IsBadGateway checks if the error resolves to a CodeBadGateway.
func IsBadGateway(err error) bool { return IsCode(err, CodeBadGateway) }

// IsServiceUnavailable checks if the error resolves to a CodeServiceUnavailable.
func IsServiceUnavailable(err error) bool { return IsCode(err, CodeServiceUnavailable) }

// IsGatewayTimeout checks if the error resolves to a CodeGatewayTimeout.
func IsGatewayTimeout(err error) bool { return IsCode(err, CodeGatewayTimeout) }

// IsRequestTimeout checks if the error resolves to a CodeRequestTimeout.
func IsRequestTimeout(err error) bool { return IsCode(err, CodeRequestTimeout) }

// IsClientClosedRequest checks if the error resolves to a CodeClientClosedRequest.
func IsClientClosedRequest(err error) bool { return IsCode(err, CodeClientClosedRequest) }
