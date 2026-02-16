package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtIssuer   = "ws-lab"
	jwtAudience = "ws-client"
	jwtTTL      = 5 * time.Minute
)

type WSClaims struct {
	jwt.RegisteredClaims
}

var jwtSecret = []byte(loadJWTSecret())

func loadJWTSecret() string {
	if secret := os.Getenv("WS_JWT_SECRET"); secret != "" {
		return secret
	}
	return "dev-only-change-me"
}

func issueOTP(subject string) (string, error) {
	now := time.Now()
	claims := WSClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(jwtTTL)),
			ID:        randomTokenID(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateOTP(raw string) (*WSClaims, error) {
	if raw == "" {
		return nil, fmt.Errorf("missing otp")
	}

	claims := &WSClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid otp")
	}
	return claims, nil
}

func randomTokenID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
