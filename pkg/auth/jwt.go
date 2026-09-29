package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret   []byte
	ttl      time.Duration
	issuer   string
	audience string
}

func NewTokenIssuer(secret string, ttl time.Duration, issuer, audience string) *TokenIssuer {
	return &TokenIssuer{
		secret:   []byte(secret),
		ttl:      ttl,
		issuer:   issuer,
		audience: audience,
	}
}

func NewTokenIssuerFromSettings(settings JWTSettings, ttl time.Duration) *TokenIssuer {
	return NewTokenIssuer(string(settings.Secret), ttl, settings.Issuer, settings.Audience)
}

func (t *TokenIssuer) Issue(userID, tenantID, email string, role Role) (string, time.Time, error) {
	exp := time.Now().Add(t.ttl)
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    t.issuer,
			Audience:  jwt.ClaimStrings{t.audience},
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	return signed, exp, err
}

func (t *TokenIssuer) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	}, jwt.WithIssuer(t.issuer), jwt.WithAudience(t.audience))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
