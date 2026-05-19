package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sso.pelajarnumagetan.or.id/internal/config"
)

type JWTClaims struct {
	UserID   uuid.UUID `json:"sub"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Scope    string    `json:"scope"`
	ClientID string    `json:"client_id"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func GenerateAccessToken(userID uuid.UUID, email, name, scope, clientID string) (string, error) {
	cfg := config.Get()

	claims := JWTClaims{
		UserID:   userID,
		Email:    email,
		Name:     name,
		Scope:    scope,
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "sso.pelajarnumagetan.or.id",
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWTAccessExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func ParseAccessToken(tokenStr string) (*JWTClaims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func GenerateRefreshToken() string {
	return uuid.New().String() + "-" + uuid.New().String()
}

type EmailClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateEmailVerificationToken(email string) (string, error) {
	cfg := config.Get()
	claims := EmailClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "sso.pelajarnumagetan.or.id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Valid 24 jam
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func ParseEmailVerificationToken(tokenStr string) (string, error) {
	cfg := config.Get()
	token, err := jwt.ParseWithClaims(tokenStr, &EmailClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*EmailClaims)
	if !ok || !token.Valid {
		return "", errors.New("token tidak valid")
	}
	return claims.Email, nil
}
