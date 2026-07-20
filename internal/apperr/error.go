package apperr

import (
	"fmt"
)

var _ error = (*Error)(nil)

type Error struct {
	Code                Code                 `json:"code"`
	Message             string               `json:"message"`
	ErrorInfo           *ErrorInfo           `json:"info,omitempty"`
	RetryInfo           *RetryInfo           `json:"retryInfo,omitempty"`
	DebugInfo           *DebugInfo           `json:"debugInfo,omitempty"`
	QuotaFailure        *QuotaFailure        `json:"quotaFailure,omitempty"`
	PreconditionFailure *PreconditionFailure `json:"preconditionFailure,omitempty"`
	BadRequest          *BadRequest          `json:"badRequest,omitempty"`
	RequestInfo         *RequestInfo         `json:"requestInfo,omitempty"`
	ResourceInfo        *ResourceInfo        `json:"resourceInfo,omitempty"`
	Help                *Help                `json:"help,omitempty"`
	LocalizedMessage    *LocalizedMessage    `json:"localizedMessage,omitempty"`
	Err                 error                `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil apperr.Error>"
	}

	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
