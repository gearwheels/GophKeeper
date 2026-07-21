// Package migration выполняет автоматическое применение SQL-миграций при старте сервера.
package migration

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/timofeevav/gophkeeper/migrations"
)

// Run проверяет наличие неприменённых миграций и применяет их.
// Логирует результат: сколько миграций применено или что БД уже актуальна.
func Run(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	currentVersion, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("get migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("database is dirty at version %d — manual fix required", currentVersion)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("migrations: already up to date", "version", currentVersion)
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	newVersion, _, _ := m.Version()
	slog.Info("migrations: applied successfully",
		"from_version", currentVersion,
		"to_version", newVersion,
	)
	return nil
}
