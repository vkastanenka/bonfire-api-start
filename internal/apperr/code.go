package apperr

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

func (c Code) String() string {
	switch c {
	case CodeOK:
		return "OK"
	case CodeCancelled:
		return "CANCELLED"
	case CodeUnknown:
		return "UNKNOWN"
	case CodeInvalidArgument:
		return "INVALID_ARGUMENT"
	case CodeDeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case CodeNotFound:
		return "NOT_FOUND"
	case CodeAlreadyExists:
		return "ALREADY_EXISTS"
	case CodePermissionDenied:
		return "PERMISSION_DENIED"
	case CodeResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case CodeFailedPrecondition:
		return "FAILED_PRECONDITION"
	case CodeAborted:
		return "ABORTED"
	case CodeOutOfRange:
		return "OUT_OF_RANGE"
	case CodeUnimplemented:
		return "UNIMPLEMENTED"
	case CodeInternal:
		return "INTERNAL"
	case CodeUnavailable:
		return "UNAVAILABLE"
	case CodeDataLoss:
		return "DATA_LOSS"
	case CodeUnauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "UNKNOWN"
	}
}

func (c Code) Message() string {
	switch c {
	case CodeOK:
		return "The operation completed successfully."
	case CodeCancelled:
		return "The operation was cancelled."
	case CodeInvalidArgument:
		return "An invalid argument was provided."
	case CodeDeadlineExceeded:
		return "The deadline expired before the operation could complete."
	case CodeNotFound:
		return "The requested entity could not be found."
	case CodeAlreadyExists:
		return "The entity you attempted to create already exists."
	case CodePermissionDenied:
		return "You do not have permission to execute this operation."
	case CodeResourceExhausted:
		return "A system resource or rate quota has been exhausted."
	case CodeFailedPrecondition:
		return "The operation was rejected because the system is not in a state required for execution."
	case CodeAborted:
		return "The operation was aborted, typically due to a system concurrency or transaction conflict."
	case CodeOutOfRange:
		return "The operation was attempted past the valid bounds or index range."
	case CodeUnimplemented:
		return "This system capability is not implemented or enabled in this service."
	case CodeInternal:
		return "An internal error occurred."
	case CodeUnavailable:
		return "The service is temporarily unavailable. Please retry with backoff."
	case CodeDataLoss:
		return "Unrecoverable data loss or system corruption occurred."
	case CodeUnauthenticated:
		return "The request lacks valid authentication credentials for the operation."
	default:
		return "An unknown system error occurred."
	}
}
