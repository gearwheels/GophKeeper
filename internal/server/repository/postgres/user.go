// Package postgres реализует репозитории на основе PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timofeevav/gophkeeper/internal/server/model"
)

// UserRepo реализует service.UserRepository поверх pgxpool.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo создаёт новый UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// Create вставляет нового пользователя в БД.
func (r *UserRepo) Create(ctx context.Context, login, passwordHash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`,
		login, passwordHash,
	).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return uuid.Nil, ErrDuplicate
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

// GetByLogin возвращает пользователя по логину.
func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, login, password, created_at, updated_at FROM users WHERE login = $1`,
		login,
	).Scan(&u.ID, &u.Login, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by login: %w", err)
	}
	return u, nil
}
