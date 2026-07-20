package apperr

import "fmt"

// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto

type Code int32

const (
	CodeOK                 Code = 0
	CodeCancelled          Code = 1
	CodeUnknown            Code = 2
	CodeInvalidArgument    Code = 3
	CodeDeadlineExceeded   Code = 4
	CodeNotFound           Code = 5
	CodeAlreadyExists      Code = 6
	CodePermissionDenied   Code = 7
	CodeResourceExhausted  Code = 8
	CodeFailedPrecondition Code = 9
	CodeAborted            Code = 10
	CodeOutOfRange         Code = 11
	CodeUnimplemented      Code = 12
	CodeInternal           Code = 13
	CodeUnavailable        Code = 14
	CodeDataLoss           Code = 15
	CodeUnauthenticated    Code = 16
)

var codeNames = [...]string{
	CodeOK:                 "OK",
	CodeCancelled:          "CANCELLED",
	CodeUnknown:            "UNKNOWN",
	CodeInvalidArgument:    "INVALID_ARGUMENT",
	CodeDeadlineExceeded:   "DEADLINE_EXCEEDED",
	CodeNotFound:           "NOT_FOUND",
	CodeAlreadyExists:      "ALREADY_EXISTS",
	CodePermissionDenied:   "PERMISSION_DENIED",
	CodeResourceExhausted:  "RESOURCE_EXHAUSTED",
	CodeFailedPrecondition: "FAILED_PRECONDITION",
	CodeAborted:            "ABORTED",
	CodeOutOfRange:         "OUT_OF_RANGE",
	CodeUnimplemented:      "UNIMPLEMENTED",
	CodeInternal:           "INTERNAL",
	CodeUnavailable:        "UNAVAILABLE",
	CodeDataLoss:           "DATA_LOSS",
	CodeUnauthenticated:    "UNAUTHENTICATED",
}

var codeMessages = [...]string{
	CodeOK:                 "The operation completed successfully.",
	CodeCancelled:          "The operation was cancelled.",
	CodeUnknown:            "An unknown system error occurred.",
	CodeInvalidArgument:    "An invalid argument was provided.",
	CodeDeadlineExceeded:   "The deadline expired before the operation could complete.",
	CodeNotFound:           "The requested entity could not be found.",
	CodeAlreadyExists:      "The entity you attempted to create already exists.",
	CodePermissionDenied:   "You do not have permission to execute this operation.",
	CodeResourceExhausted:  "A resource or rate quota has been exhausted.",
	CodeFailedPrecondition: "The operation was rejected because the system is not in a state required for execution.",
	CodeAborted:            "The operation was aborted.",
	CodeOutOfRange:         "The operation was attempted past the valid bounds or index range.",
	CodeUnimplemented:      "This system capability is not implemented or enabled in this service.",
	CodeInternal:           "An internal error occurred.",
	CodeUnavailable:        "The service is temporarily unavailable. Please retry later.",
	CodeDataLoss:           "Unrecoverable data loss or system corruption occurred.",
	CodeUnauthenticated:    "The request lacks valid credentials.",
}

func (c Code) String() string {
	if int(c) >= 0 && int(c) < len(codeNames) {
		return codeNames[c]
	}
	return fmt.Sprintf("CODE_%d", c)
}

func (c Code) Message() string {
	if int(c) >= 0 && int(c) < len(codeMessages) {
		return codeMessages[c]
	}
	return "An unknown error occurred."
}
