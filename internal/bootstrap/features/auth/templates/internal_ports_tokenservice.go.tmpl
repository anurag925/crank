package ports

// TokenPair is the response returned to clients on login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// TokenService issues and validates JWT access/refresh token pairs.
type TokenService interface {
	// Issue produces a fresh access+refresh pair for the supplied subject
	// (typically the user's id).
	Issue(subject string) (*TokenPair, error)
	// Refresh exchanges a valid refresh token for a new pair.
	Refresh(refreshToken string) (*TokenPair, error)
	// Subject returns the subject claim encoded in a valid access token.
	// Returns an error if the token is invalid or expired.
	Subject(accessToken string) (string, error)
	// Revoke denylists a refresh token to prevent its reuse. The token is
	// parsed to extract its JTI, which is stored in the TokenDenylist.
	Revoke(refreshToken string) error
}
