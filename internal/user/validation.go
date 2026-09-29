package user

const (
	tagBio              = "user_bio"
	tagBioRules         = "max=190"
	tagDisplayName      = "user_display_name"
	tagDisplayNameRules = "min=3,max=32"
	tagPassword         = "user_password"
	tagPasswordRules    = "min=12,max=255"
	tagUsername         = "user_username"
	tagUsernameRules    = "min=3,max=32,alphanum"
)

func ValidationAliases() map[string]string {
	return map[string]string{
		tagBio:         tagBioRules,
		tagDisplayName: tagDisplayNameRules,
		tagPassword:    tagPasswordRules,
		tagUsername:    tagUsernameRules,
	}
}
