package user

import (
	"bonfire-api/internal/pkg/parse"
	"time"
)

type PreferredPresenceDuration int

const (
	PreferredPresenceDurationUnknown PreferredPresenceDuration = iota
	PreferredPresenceDuration15Min
	PreferredPresenceDuration1Hour
	PreferredPresenceDuration8Hours
	PreferredPresenceDuration24Hours
	PreferredPresenceDuration3Days
	PreferredPresenceDurationForever
	preferredPresenceDurationMax
)

var preferredPresenceDurationNames = [...]string{
	PreferredPresenceDurationUnknown: "UNKNOWN",
	PreferredPresenceDuration15Min:   "15_MIN",
	PreferredPresenceDuration1Hour:   "1_HOUR",
	PreferredPresenceDuration8Hours:  "8_HOURS",
	PreferredPresenceDuration24Hours: "24_HOURS",
	PreferredPresenceDuration3Days:   "3_DAYS",
	PreferredPresenceDurationForever: "FOREVER",
}

func ParsePreferredPresenceDurationInt(i int) (PreferredPresenceDuration, error) {
	return parse.EnumInt(i, preferredPresenceDurationMax, "preferred_presence_duration")
}

func ParsePreferredPresenceDurationIntBytes(b []byte) (PreferredPresenceDuration, error) {
	return parse.EnumIntBytes(b, preferredPresenceDurationMax, "preferred_presence_duration")
}

func ParsePreferredPresenceDurationString(s string) (PreferredPresenceDuration, error) {
	return parse.EnumString[PreferredPresenceDuration](s, preferredPresenceDurationNames[:], "preferred_presence_duration")
}

func ParsePreferredPresenceDurationStringBytes(b []byte) (PreferredPresenceDuration, error) {
	return parse.EnumStringBytes[PreferredPresenceDuration](b, preferredPresenceDurationNames[:], "preferred_presence_duration")
}

func (d PreferredPresenceDuration) IsValid() bool {
	return d > PreferredPresenceDurationUnknown && d < preferredPresenceDurationMax
}

func (p PreferredPresenceDuration) Int() int {
	if p.IsValid() {
		return int(p)
	}
	return int(PreferredPresenceDurationUnknown)
}

func (d PreferredPresenceDuration) String() string {
	if uint(d) < uint(len(preferredPresenceDurationNames)) {
		return preferredPresenceDurationNames[d]
	}
	return preferredPresenceDurationNames[PreferredPresenceDurationUnknown]
}

func (d PreferredPresenceDuration) Is15Min() bool   { return d == PreferredPresenceDuration15Min }
func (d PreferredPresenceDuration) Is1Hour() bool   { return d == PreferredPresenceDuration1Hour }
func (d PreferredPresenceDuration) Is8Hours() bool  { return d == PreferredPresenceDuration8Hours }
func (d PreferredPresenceDuration) Is24Hours() bool { return d == PreferredPresenceDuration24Hours }
func (d PreferredPresenceDuration) Is3Days() bool   { return d == PreferredPresenceDuration3Days }
func (d PreferredPresenceDuration) IsForever() bool { return d == PreferredPresenceDurationForever }

func (d PreferredPresenceDuration) ToPreferredPresenceDuration() (time.Duration, bool) {
	switch d {
	case PreferredPresenceDuration15Min:
		return 15 * time.Minute, true
	case PreferredPresenceDuration1Hour:
		return time.Hour, true
	case PreferredPresenceDuration8Hours:
		return 8 * time.Hour, true
	case PreferredPresenceDuration24Hours:
		return 24 * time.Hour, true
	case PreferredPresenceDuration3Days:
		return 72 * time.Hour, true
	case PreferredPresenceDurationForever:
		return 0, true
	default:
		return 0, false
	}
}

func (d PreferredPresenceDuration) CalculateUntil(now time.Time) (*time.Time, error) {
	if !d.IsValid() {
		return nil, ErrPreferredPresenceDurationInvalid()
	}
	if d.IsForever() {
		return nil, nil
	}

	stdPreferredPresenceDuration, _ := d.ToPreferredPresenceDuration()
	until := now.Add(stdPreferredPresenceDuration)
	return &until, nil
}

func (d PreferredPresenceDuration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *PreferredPresenceDuration) UnmarshalText(text []byte) error {
	parsed, err := ParsePreferredPresenceDurationString(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
