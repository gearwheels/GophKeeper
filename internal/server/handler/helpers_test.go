package handler_test

import (
	"time"

	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/crypto"
)

// generateTestToken создаёт JWT-токен для тестов.
func generateTestToken(userID uuid.UUID, secret string) (string, error) {
	return crypto.GenerateToken(userID, secret, time.Hour)
}
