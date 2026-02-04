package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService issues and validates JWT tokens.
type JWTService struct {
	secret []byte
}

// NewJWTService creates a JWTService with the provided secret.
func NewJWTService(secret string) (*JWTService, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is required")
	}
	return &JWTService{secret: []byte(secret)}, nil
}

// Claims are JWT claims used by the server.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// IssueToken creates a signed JWT.
func (s *JWTService) IssueToken(userID, email string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ParseToken validates a JWT and returns claims.
func (s *JWTService) ParseToken(tokenStr string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}

	return *claims, nil
}
