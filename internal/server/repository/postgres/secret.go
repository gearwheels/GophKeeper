package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timofeevav/gophkeeper/internal/server/model"
)

// SecretRepo реализует service.SecretRepository поверх pgxpool.
type SecretRepo struct {
	db *pgxpool.Pool
}

// NewSecretRepo создаёт новый SecretRepo.
func NewSecretRepo(db *pgxpool.Pool) *SecretRepo {
	return &SecretRepo{db: db}
}

// Create вставляет новый секрет в БД.
func (r *SecretRepo) Create(ctx context.Context, s *model.Secret) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`INSERT INTO secrets (user_id, type, name, data, meta)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.UserID, s.Type, s.Name, s.Data, s.Meta,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create secret: %w", err)
	}
	return id, nil
}

// GetByID возвращает секрет по ID с проверкой владельца.
func (r *SecretRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (*model.Secret, error) {
	s := &model.Secret{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, type, name, data, meta, version, created_at, updated_at, deleted_at
		 FROM secrets WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID,
	).Scan(&s.ID, &s.UserID, &s.Type, &s.Name, &s.Data, &s.Meta,
		&s.Version, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get secret by id: %w", err)
	}
	return s, nil
}

// List возвращает список секретов без поля data.
func (r *SecretRepo) List(ctx context.Context, userID uuid.UUID, secretType model.SecretType, since *time.Time) ([]*model.Secret, error) {
	query := `SELECT id, user_id, type, name, meta, version, created_at, updated_at, deleted_at
	          FROM secrets WHERE user_id = $1`
	args := []interface{}{userID}

	if secretType != "" {
		args = append(args, secretType)
		query += fmt.Sprintf(" AND type = $%d", len(args))
	}
	if since != nil {
		args = append(args, *since)
		query += fmt.Sprintf(" AND updated_at > $%d", len(args))
	} else {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*model.Secret
	for rows.Next() {
		s := &model.Secret{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Name, &s.Meta,
			&s.Version, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		secrets = append(secrets, s)
	}
	return secrets, rows.Err()
}

// Update обновляет секрет с проверкой версии (optimistic locking).
func (r *SecretRepo) Update(ctx context.Context, s *model.Secret) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE secrets SET name=$1, data=$2, meta=$3, version=version+1, updated_at=NOW()
		 WHERE id=$4 AND user_id=$5 AND version=$6 AND deleted_at IS NULL`,
		s.Name, s.Data, s.Meta, s.ID, s.UserID, s.Version,
	)
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// Delete выполняет мягкое удаление секрета.
func (r *SecretRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE secrets SET deleted_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetVersions возвращает все секреты пользователя, изменённые после since (включая удалённые).
func (r *SecretRepo) GetVersions(ctx context.Context, userID uuid.UUID, since time.Time) ([]*model.Secret, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, name, data, meta, version, created_at, updated_at, deleted_at
		 FROM secrets WHERE user_id=$1 AND updated_at > $2 ORDER BY updated_at`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}
	defer rows.Close()

	var secrets []*model.Secret
	for rows.Next() {
		s := &model.Secret{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Name, &s.Data, &s.Meta,
			&s.Version, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		secrets = append(secrets, s)
	}
	return secrets, rows.Err()
}
