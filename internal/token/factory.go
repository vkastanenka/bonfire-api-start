package token

import (
	"time"

	"github.com/google/uuid"
)

type Pair struct {
	Access           string
	AccessExpiresAt  time.Time
	Refresh          string
	RefreshExpiresAt time.Time
}

func (p *Provider) GeneratePair(uid, sid uuid.UUID) (Pair, error) {
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

func (p *Provider) GenerateAccess(uid, sid uuid.UUID) (string, time.Time, error) {
	return p.generate(VariantAccess, Claims{
		UserID:    uid,
		SessionID: sid,
	})
}

func (p *Provider) GenerateRefresh(uid, sid uuid.UUID) (string, time.Time, error) {
	return p.generate(VariantRefresh, Claims{
		UserID:    uid,
		SessionID: sid,
	})
}

func (p *Provider) GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error) {
	return p.generate(VariantEmailVerify, Claims{
		UserID: userID,
	})
}

func (p *Provider) GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error) {
	return p.generate(VariantPasswordReset, Claims{
		UserID: userID,
	})
}

func (p *Provider) VerifyAccess(tokenStr string) (*Claims, error) {
	return p.verify(VariantAccess, tokenStr)
}

func (p *Provider) VerifyRefresh(tokenStr string) (*Claims, error) {
	return p.verify(VariantRefresh, tokenStr)
}

func (p *Provider) VerifyEmailVerify(tokenStr string) (*Claims, error) {
	return p.verify(VariantEmailVerify, tokenStr)
}

func (p *Provider) VerifyPasswordReset(tokenStr string) (*Claims, error) {
	return p.verify(VariantPasswordReset, tokenStr)
}
