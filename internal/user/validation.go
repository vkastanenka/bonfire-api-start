package user

const (
	tagBio      = "user_bio"
	tagBioRules = "max=190"

	tagDisplayName      = "user_display_name"
	tagDisplayNameRules = "min=3,max=32"

	tagPassword      = "user_password"
	tagPasswordRules = "min=12,max=72"

	tagUsername      = "user_username"
	tagUsernameRules = "min=3,max=32,alphanum"

	tagPreferredPresence      = "user_preferred_presence"
	tagPreferredPresenceRules = "oneof=4 5 6"

	tagPreferredPresenceDuration      = "user_preferred_presence_duration"
	tagPreferredPresenceDurationRules = "oneof=1 2 3 4 5 6"
)

func ValidationAliases() map[string]string {
	return map[string]string{
		tagBio:                       tagBioRules,
		tagDisplayName:               tagDisplayNameRules,
		tagPassword:                  tagPasswordRules,
		tagUsername:                  tagUsernameRules,
		tagPreferredPresence:         tagPreferredPresenceRules,
		tagPreferredPresenceDuration: tagPreferredPresenceDurationRules,
	}
}
