package auth

import "github.com/google/uuid"

// AuthenticationTokenPair represents a pair of authentication tokens (Access and Refresh).
type AuthenticationTokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// GenerateToken returns a new AuthenticationTokenPair
func GenerateToken() AuthenticationTokenPair {
	return AuthenticationTokenPair{
		AccessToken:  uuid.New().String(),
		RefreshToken: uuid.New().String(),
	}
}
