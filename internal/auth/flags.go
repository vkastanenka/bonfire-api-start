package auth

// UserFlag represents a bitmask of account states or achievements.
type UserFlag int64

const (
	UserFlagNone         UserFlag = 0
	UserFlagVerified     UserFlag = 1 << 0 // Bit 0 (Value: 1)
	// UserFlagStaff        UserFlag = 1 << 1 // Bit 1 (Value: 2)
	// UserFlagPartner      UserFlag = 1 << 2 // Bit 2 (Value: 4)
	// UserFlagEarlyAdopter UserFlag = 1 << 3 // Bit 3 (Value: 8)
	// UserFlagBanned       UserFlag = 1 << 4 // Bit 4 (Value: 16)
)

// Has evaluates whether a specific flag is set within the current mask.
func (f UserFlag) Has(flag UserFlag) bool {
	return (f & flag) == flag
}
