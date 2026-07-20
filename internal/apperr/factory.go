package apperr

func NewCancelled(err error, opts ...Option) error {
	return New(CodeCancelled, append(opts, WithError(err))...)
}
func NewUnknown(err error, opts ...Option) error {
	return New(CodeUnknown, append(opts, WithError(err))...)
}
func NewInvalidArgument(err error, opts ...Option) error {
	return New(CodeInvalidArgument, append(opts, WithError(err))...)
}
func NewDeadlineExceeded(err error, opts ...Option) error {
	return New(CodeDeadlineExceeded, append(opts, WithError(err))...)
}
func NewNotFound(err error, opts ...Option) error {
	return New(CodeNotFound, append(opts, WithError(err))...)
}
func NewAlreadyExists(err error, opts ...Option) error {
	return New(CodeAlreadyExists, append(opts, WithError(err))...)
}
func NewPermissionDenied(err error, opts ...Option) error {
	return New(CodePermissionDenied, append(opts, WithError(err))...)
}
func NewResourceExhausted(err error, opts ...Option) error {
	return New(CodeResourceExhausted, append(opts, WithError(err))...)
}
func NewFailedPrecondition(err error, opts ...Option) error {
	return New(CodeFailedPrecondition, append(opts, WithError(err))...)
}
func NewAborted(err error, opts ...Option) error {
	return New(CodeAborted, append(opts, WithError(err))...)
}
func NewOutOfRange(err error, opts ...Option) error {
	return New(CodeOutOfRange, append(opts, WithError(err))...)
}
func NewUnimplemented(err error, opts ...Option) error {
	return New(CodeUnimplemented, append(opts, WithError(err))...)
}
func NewInternal(err error, opts ...Option) error {
	return New(CodeInternal, append(opts, WithError(err))...)
}
func NewUnavailable(err error, opts ...Option) error {
	return New(CodeUnavailable, append(opts, WithError(err))...)
}
func NewDataLoss(err error, opts ...Option) error {
	return New(CodeDataLoss, append(opts, WithError(err))...)
}
func NewUnauthenticated(err error, opts ...Option) error {
	return New(CodeUnauthenticated, append(opts, WithError(err))...)
}

func IsCancelled(err error) bool          { return IsCode(err, CodeCancelled) }
func IsUnknown(err error) bool            { return IsCode(err, CodeUnknown) }
func IsInvalidArgument(err error) bool    { return IsCode(err, CodeInvalidArgument) }
func IsDeadlineExceeded(err error) bool   { return IsCode(err, CodeDeadlineExceeded) }
func IsNotFound(err error) bool           { return IsCode(err, CodeNotFound) }
func IsAlreadyExists(err error) bool      { return IsCode(err, CodeAlreadyExists) }
func IsPermissionDenied(err error) bool   { return IsCode(err, CodePermissionDenied) }
func IsResourceExhausted(err error) bool  { return IsCode(err, CodeResourceExhausted) }
func IsFailedPrecondition(err error) bool { return IsCode(err, CodeFailedPrecondition) }
func IsAborted(err error) bool            { return IsCode(err, CodeAborted) }
func IsOutOfRange(err error) bool         { return IsCode(err, CodeOutOfRange) }
func IsUnimplemented(err error) bool      { return IsCode(err, CodeUnimplemented) }
func IsInternal(err error) bool           { return IsCode(err, CodeInternal) }
func IsUnavailable(err error) bool        { return IsCode(err, CodeUnavailable) }
func IsDataLoss(err error) bool           { return IsCode(err, CodeDataLoss) }
func IsUnauthenticated(err error) bool    { return IsCode(err, CodeUnauthenticated) }
