package apperr

import "errors"

// NewBadRequest creates an error for malformed payload syntax or requests.
func NewBadRequest(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeBadRequest, detail, allOpts...)
}

// NewInvalidInput creates an error for payloads failing structural validation rules.
func NewInvalidInput(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeInvalidInput, detail, allOpts...)
}

// NewUnauthorized creates an error for general expired or fallback credential issues.
func NewUnauthorized(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeUnauthorized, detail, allOpts...)
}

// NewForbidden creates an error for authenticated users who lack RBAC/permission rules.
func NewForbidden(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeForbidden, detail, allOpts...)
}

// NewNotFound creates an error when a targeted resource or entity is missing.
func NewNotFound(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeNotFound, detail, allOpts...)
}

// NewMethodNotAllowed creates an error when an unsupported HTTP verb hits an endpoint.
func NewMethodNotAllowed(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeMethodNotAllowed, detail, allOpts...)
}

// NewConflict creates an error for state duplicates or concurrent race condition blocks.
func NewConflict(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeConflict, detail, allOpts...)
}

// NewGone creates an error for historical resources that have been permanently purged.
func NewGone(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeGone, detail, allOpts...)
}

// NewPreconditionFailed creates an error for optimistic concurrency check failures (ETags).
func NewPreconditionFailed(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodePreconditionFailed, detail, allOpts...)
}

// NewPayloadTooLarge creates an error when request bodies cross maximum sizing boundaries.
func NewPayloadTooLarge(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodePayloadTooLarge, detail, allOpts...)
}

// NewUnsupportedMediaType creates an error when Content-Type headers do not match expectations.
func NewUnsupportedMediaType(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeUnsupportedMediaType, detail, allOpts...)
}

// NewUnprocessableEntity creates an error when a structurally sound payload breaks semantic business logic rules.
func NewUnprocessableEntity(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeUnprocessableEntity, detail, allOpts...)
}

// NewTooManyRequests creates an error when rate-limit throttling thresholds are breached.
func NewTooManyRequests(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeTooManyRequests, detail, allOpts...)
}

// NewInternal creates a generic infrastructure error, safely encapsulating the root cause.
func NewInternal(err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeInternal, "", allOpts...)
}

// NewNotImplemented creates an error for system paths or capabilities not yet fully written.
func NewNotImplemented(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeNotImplemented, detail, allOpts...)
}

// NewBadGateway creates an error when an integrated external service or microservice returns trash.
func NewBadGateway(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeBadGateway, detail, allOpts...)
}

// NewServiceUnavailable creates an error when systems are down for scaling, overloading, or maintenance.
func NewServiceUnavailable(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeServiceUnavailable, detail, allOpts...)
}

// NewGatewayTimeout creates an error when downstream integrations fail to respond within a network deadline.
func NewGatewayTimeout(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeGatewayTimeout, detail, allOpts...)
}

// NewRequestTimeout creates an error when internal request execution context deadlines expire.
func NewRequestTimeout(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeRequestTimeout, detail, allOpts...)
}

// NewClientClosedRequest creates an error when an upstream router detects the client hung up early.
func NewClientClosedRequest(detail string, err error, opts ...ErrorOption) error {
	allOpts := append([]ErrorOption{Err(err)}, opts...)
	return New(CodeClientClosedRequest, detail, allOpts...)
}

// IsCode verifies if an abstract error unwraps to an apperr matching a target Code classification.
func IsCode(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

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
