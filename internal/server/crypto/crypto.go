// Package crypto реализует хеширование паролей и генерацию JWT-токенов.
package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword возвращает bcrypt-хеш пароля.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword проверяет соответствие пароля его хешу.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// claims содержит payload JWT-токена.
type claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
}

// GenerateToken создаёт JWT-токен для указанного пользователя.
func GenerateToken(userID uuid.UUID, secret string, ttl time.Duration) (string, error) {
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken разбирает и валидирует JWT-токен, возвращая ID пользователя.
func ParseToken(tokenStr, secret string) (uuid.UUID, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse token: %w", err)
	}
	c, ok := t.Claims.(*claims)
	if !ok || !t.Valid {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}
	return c.UserID, nil
}
