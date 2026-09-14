package user

import (
	"bonfire-api/internal/sanitize"
	"bytes"
	"fmt"
	"strconv"
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
	durationMax
)

var durationNames = [...]string{
	PreferredPresenceDurationUnknown: "UNKNOWN",
	PreferredPresenceDuration15Min:   "15_MIN",
	PreferredPresenceDuration1Hour:   "1_HOUR",
	PreferredPresenceDuration8Hours:  "8_HOURS",
	PreferredPresenceDuration24Hours: "24_HOURS",
	PreferredPresenceDuration3Days:   "3_DAYS",
	PreferredPresenceDurationForever: "FOREVER",
}

func ParsePreferredPresenceDuration(raw int) (PreferredPresenceDuration, error) {
	d := PreferredPresenceDuration(raw)
	if !d.IsValid() {
		return PreferredPresenceDurationUnknown, fmt.Errorf("invalid duration value: %d", raw)
	}
	return d, nil
}

func ParsePreferredPresenceDurationString(s string) (PreferredPresenceDuration, error) {
	str := sanitize.EnumValue(s)
	if str == "" {
		return PreferredPresenceDurationUnknown, nil
	}
	for i, name := range durationNames {
		if name == str {
			return PreferredPresenceDuration(i), nil
		}
	}
	return PreferredPresenceDurationUnknown, fmt.Errorf("invalid duration string: %q", s)
}

func ParsePreferredPresenceDurationBytes(raw []byte) (PreferredPresenceDuration, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return PreferredPresenceDurationUnknown, nil
	}
	return ParsePreferredPresenceDurationString(string(cleaned))
}

func (d PreferredPresenceDuration) IsValid() bool {
	return d > PreferredPresenceDurationUnknown && d < durationMax
}

func (d PreferredPresenceDuration) String() string {
	if uint(d) < uint(len(durationNames)) {
		return durationNames[d]
	}
	return durationNames[PreferredPresenceDurationUnknown]
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
		return nil, fmt.Errorf("cannot calculate expiry for invalid duration: %s", d)
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

func (d PreferredPresenceDuration) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, d.String()), nil
}

func (d *PreferredPresenceDuration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*d = PreferredPresenceDurationUnknown
		return nil
	}

	parsed, err := ParsePreferredPresenceDurationBytes(data)
	if err != nil {
		return err
	}

	*d = parsed
	return nil
}
