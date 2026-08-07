package user

const (
	EventRequestUpdateEmailCode = "user.request-update-email-code"
	EventRequestUpdatePhoneCode = "user.request-update-phone-code"
)

type RequestUpdateEmailCodePayload struct {
	Code  string `json:"code"`
	Email string `json:"email"`
}

type RequestUpdatePhoneCodePayload struct {
	Code  string `json:"code"`
	Phone string `json:"phone"`
}
