package apperr

// NewBadRequest creates an error for malformed payload syntax or requests.
func NewBadRequest(err error, detail string, opts ...ErrorOption) error {
	return build(CodeBadRequest, err, detail, opts...)
}

// NewInvalidInput creates an error for payloads failing structural validation rules.
func NewInvalidInput(err error, detail string, opts ...ErrorOption) error {
	return build(CodeInvalidInput, err, detail, opts...)
}

// NewUnauthorized creates an error for general expired or fallback credential issues.
func NewUnauthorized(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnauthorized, err, detail, opts...)
}

// NewForbidden creates an error for authenticated users who lack RBAC/permission rules.
func NewForbidden(err error, detail string, opts ...ErrorOption) error {
	return build(CodeForbidden, err, detail, opts...)
}

// NewNotFound creates an error when a targeted resource or entity is missing.
func NewNotFound(err error, detail string, opts ...ErrorOption) error {
	return build(CodeNotFound, err, detail, opts...)
}

// NewMethodNotAllowed creates an error when an unsupported HTTP verb hits an endpoint.
func NewMethodNotAllowed(err error, detail string, opts ...ErrorOption) error {
	return build(CodeMethodNotAllowed, err, detail, opts...)
}

// NewConflict creates an error for state duplicates or concurrent race condition blocks.
func NewConflict(err error, detail string, opts ...ErrorOption) error {
	return build(CodeConflict, err, detail, opts...)
}

// NewGone creates an error for historical resources that have been permanently purged.
func NewGone(err error, detail string, opts ...ErrorOption) error {
	return build(CodeGone, err, detail, opts...)
}

// NewPreconditionFailed creates an error for optimistic concurrency check failures (ETags).
func NewPreconditionFailed(err error, detail string, opts ...ErrorOption) error {
	return build(CodePreconditionFailed, err, detail, opts...)
}

// NewPayloadTooLarge creates an error when request bodies cross maximum sizing boundaries.
func NewPayloadTooLarge(err error, detail string, opts ...ErrorOption) error {
	return build(CodePayloadTooLarge, err, detail, opts...)
}

// NewUnsupportedMediaType creates an error when Content-Type headers do not match expectations.
func NewUnsupportedMediaType(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnsupportedMediaType, err, detail, opts...)
}

// NewUnprocessableEntity creates an error when a structurally sound payload breaks semantic business logic rules.
func NewUnprocessableEntity(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnprocessableEntity, err, detail, opts...)
}

// NewTooManyRequests creates an error when rate-limit throttling thresholds are breached.
func NewTooManyRequests(err error, detail string, opts ...ErrorOption) error {
	return build(CodeTooManyRequests, err, detail, opts...)
}

// NewInternal creates a generic infrastructure error, safely encapsulating the root cause.
func NewInternal(err error, detail string, opts ...ErrorOption) error {
	return build(CodeInternal, err, detail, opts...)
}

// NewNotImplemented creates an error for system paths or capabilities not yet fully written.
func NewNotImplemented(err error, detail string, opts ...ErrorOption) error {
	return build(CodeNotImplemented, err, detail, opts...)
}

// NewBadGateway creates an error when an integrated external service or microservice returns trash.
func NewBadGateway(err error, detail string, opts ...ErrorOption) error {
	return build(CodeBadGateway, err, detail, opts...)
}

// NewServiceUnavailable creates an error when systems are down for scaling, overloading, or maintenance.
func NewServiceUnavailable(err error, detail string, opts ...ErrorOption) error {
	return build(CodeServiceUnavailable, err, detail, opts...)
}

// NewGatewayTimeout creates an error when downstream integrations fail to respond within a network deadline.
func NewGatewayTimeout(err error, detail string, opts ...ErrorOption) error {
	return build(CodeGatewayTimeout, err, detail, opts...)
}

// NewRequestTimeout creates an error when internal request execution context deadlines expire.
func NewRequestTimeout(err error, detail string, opts ...ErrorOption) error {
	return build(CodeRequestTimeout, err, detail, opts...)
}

// NewClientClosedRequest creates an error when an upstream router detects the client hung up early.
func NewClientClosedRequest(err error, detail string, opts ...ErrorOption) error {
	return build(CodeClientClosedRequest, err, detail, opts...)
}
