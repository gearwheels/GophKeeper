// Package model содержит доменные модели сервера.
package model

import (
	"time"

	"github.com/google/uuid"
)

// User представляет зарегистрированного пользователя.
type User struct {
	ID        uuid.UUID `db:"id"`
	Login     string    `db:"login"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
