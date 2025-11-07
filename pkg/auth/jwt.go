package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	jwtSecret []byte
)

type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

func GenerateJWT(userID, sessionID string) (string, time.Time, error) {
	expirationTime := time.Now().Add(24 * time.Hour * 7) // 7 days
	claims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chitchat",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", expirationTime, err
	}

	return tokenString, expirationTime, nil
}

func ValidateJWT(tokenString string) (*Claims, error) {
	if jwtSecret == nil {
		return nil, errors.New("JWT not initialized")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func RefreshJWT(tokenString string) (string, time.Time, error) {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	// Don't refresh if token expires in more than 24 hours
	if time.Until(claims.ExpiresAt.Time) > 24*time.Hour {
		return tokenString, claims.ExpiresAt.Time, nil
	}

	return GenerateJWT(claims.UserID, claims.SessionID)
}
