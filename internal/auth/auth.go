package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator interface {
	GenerateToken(claims jwt.Claims) (string, error)
	ValidateToken(token string) (*jwt.Token, error)
}

type JWTAuthenticator struct {
	secret string
	aud    string
	iss    string
}

func NEWJWTAuthenticator(secret, aud, iss string) (*JWTAuthenticator) {
	return &JWTAuthenticator{secret: secret, aud: aud, iss: iss}
}

func NewJWTAuthenticator(secret, aud, iss string) *JWTAuthenticator {
	return &JWTAuthenticator{secret: secret, aud: aud, iss: iss}
}

func (a *JWTAuthenticator) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func (a *JWTAuthenticator) ValidateToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(a.secret), nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithAudience(a.aud),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
}