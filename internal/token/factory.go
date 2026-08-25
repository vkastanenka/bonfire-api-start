package token

import (
	"time"

	"bonfire-api/internal/fields"
)

type Pair struct {
	Access           string
	AccessExpiresAt  time.Time
	Refresh          string
	RefreshExpiresAt time.Time
}

func (p *Provider) GeneratePair(uid, sid fields.ID) (Pair, error) {
	access, accessExpiresAt, err := p.GenerateAccess(uid, sid)
	if err != nil {
		return Pair{}, err
	}

	refresh, refreshExpiresAt, err := p.GenerateRefresh(uid, sid)
	if err != nil {
		return Pair{}, err
	}

	return Pair{
		Access:           access,
		AccessExpiresAt:  accessExpiresAt,
		Refresh:          refresh,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (p *Provider) GenerateAccess(uid, sid fields.ID) (string, time.Time, error) {
	return p.generate(NewTypeAccess(), Claims{
		UserID:    uid,
		SessionID: sid,
	})
}

func (p *Provider) GenerateRefresh(uid, sid fields.ID) (string, time.Time, error) {
	return p.generate(NewTypeRefresh(), Claims{
		UserID:    uid,
		SessionID: sid,
	})
}

func (p *Provider) GenerateEmailVerify(userID fields.ID) (string, time.Time, error) {
	return p.generate(NewTypeEmailVerify(), Claims{
		UserID: userID,
	})
}

func (p *Provider) GeneratePasswordReset(userID fields.ID) (string, time.Time, error) {
	return p.generate(NewTypePasswordReset(), Claims{
		UserID: userID,
	})
}

func (p *Provider) VerifyAccess(tokenStr string) (*Claims, error) {
	return p.verify(NewTypeAccess(), tokenStr)
}

func (p *Provider) VerifyRefresh(tokenStr string) (*Claims, error) {
	return p.verify(NewTypeRefresh(), tokenStr)
}

func (p *Provider) VerifyEmailVerify(tokenStr string) (*Claims, error) {
	return p.verify(NewTypeEmailVerify(), tokenStr)
}

func (p *Provider) VerifyPasswordReset(tokenStr string) (*Claims, error) {
	return p.verify(NewTypePasswordReset(), tokenStr)
}
