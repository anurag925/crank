// Package token contains the auth domain types related to JWT revocation.
package token

import "time"

// RevokedToken is the aggregate for a revoked JWT. It is recorded when a
// refresh token is revoked (e.g. on logout) so the token cannot be reused
// before it naturally expires. GORM tags live on the aggregate itself — the
// same struct is used as both the domain model and the persistence row, so
// the persistence adapter needs no separate row DTO.
type RevokedToken struct {
	JTI       string    `gorm:"column:jti;primaryKey"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null"`
}

// NewRevokedToken constructs a RevokedToken for the given JWT id (jti) and
// its original expiration time.
func NewRevokedToken(jti string, expiresAt time.Time) RevokedToken {
	return RevokedToken{JTI: jti, ExpiresAt: expiresAt}
}

// TableName maps the RevokedToken aggregate to the revoked_tokens table.
func (RevokedToken) TableName() string {
	return "revoked_tokens"
}

// IsExpired reports whether the revocation record has outlived the token's
// original expiry and may be pruned.
func (t RevokedToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}
