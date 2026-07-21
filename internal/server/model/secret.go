package model

import (
	"time"

	"github.com/google/uuid"
)

// SecretType перечисляет допустимые типы секретов.
type SecretType string

const (
	SecretTypeLoginPassword SecretType = "login_password"
	SecretTypeText          SecretType = "text"
	SecretTypeBinary        SecretType = "binary"
	SecretTypeCard          SecretType = "card"
)

// Secret представляет единицу хранимых приватных данных.
type Secret struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	Type      SecretType `db:"type"`
	Name      string     `db:"name"`
	Data      []byte     `db:"data"`
	Meta      string     `db:"meta"`
	Version   int64      `db:"version"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// ClientVersion содержит ID и версию секрета со стороны клиента.
type ClientVersion struct {
	ID      uuid.UUID `json:"id"`
	Version int64     `json:"version"`
}

// SyncConflict описывает конфликт версий между клиентом и сервером.
type SyncConflict struct {
	ID            uuid.UUID `json:"id"`
	ServerVersion int64     `json:"server_version"`
	ClientVersion int64     `json:"client_version"`
}
