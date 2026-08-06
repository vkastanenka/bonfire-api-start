package user

const (
	EventRequestUpdateEmailCode = "user.request-update-email-code"
)

type RequestUpdateEmailCodePayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
