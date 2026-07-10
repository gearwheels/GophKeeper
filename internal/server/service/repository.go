package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/timofeevav/gophkeeper/internal/server/model"
)

// UserRepository описывает операции с пользователями.
type UserRepository interface {
	// Create создаёт нового пользователя и возвращает его ID.
	Create(ctx context.Context, login, passwordHash string) (uuid.UUID, error)
	// GetByLogin возвращает пользователя по логину.
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

// SecretRepository описывает операции с секретами.
type SecretRepository interface {
	// Create создаёт секрет и возвращает его ID.
	Create(ctx context.Context, secret *model.Secret) (uuid.UUID, error)
	// GetByID возвращает секрет по ID с проверкой владельца.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Secret, error)
	// List возвращает список секретов пользователя (без поля data).
	List(ctx context.Context, userID uuid.UUID, secretType model.SecretType, since *time.Time) ([]*model.Secret, error)
	// Update обновляет секрет с проверкой версии (optimistic locking).
	Update(ctx context.Context, secret *model.Secret) error
	// Delete выполняет мягкое удаление секрета.
	Delete(ctx context.Context, id, userID uuid.UUID) error
	// GetVersions возвращает текущие версии всех секретов пользователя.
	GetVersions(ctx context.Context, userID uuid.UUID, since time.Time) ([]*model.Secret, error)
}
