package user

import "regexp"

var RgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)

func IsValidUsername(username string) bool {
	return RgxUsername.MatchString(username)
}

const (
	TagUsername = "userUsername"
	MsgUsername = "Must start and end with a letter or number. May contain only letters, numbers, and non-consecutive underscores or periods."
)

var Aliases = map[string]string{
	"userUsernameSchema":           "min=3,max=32,userUsername",
	"userPasswordSchema":           "min=12,max=255",
	"userProfileDisplayNameSchema": "min=3,max=32",
}
