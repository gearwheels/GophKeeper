// Package storage реализует локальное хранилище данных клиента на основе bbolt.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketSecrets = []byte("secrets")
	bucketMeta    = []byte("meta")
)

// LocalSecret представляет секрет в локальном хранилище.
type LocalSecret struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Data      []byte     `json:"data"` // зашифрованный payload
	Meta      string     `json:"meta"`
	Version   int64      `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// Storage управляет локальной базой данных bbolt.
type Storage struct {
	db *bolt.DB
}

// Open открывает или создаёт локальное хранилище по указанному пути.
func Open(dbPath string) (*Storage, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt db: %w", err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketSecrets); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(bucketMeta)
		return err
	}); err != nil {
		return nil, fmt.Errorf("init buckets: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close закрывает хранилище.
func (s *Storage) Close() error {
	return s.db.Close()
}

// UpsertSecret сохраняет или обновляет секрет в локальном хранилище.
func (s *Storage) UpsertSecret(secret *LocalSecret) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		data, err := json.Marshal(secret)
		if err != nil {
			return fmt.Errorf("marshal secret: %w", err)
		}
		return b.Put([]byte(secret.ID), data)
	})
}

// GetSecret возвращает секрет по ID.
func (s *Storage) GetSecret(id string) (*LocalSecret, error) {
	var secret LocalSecret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("secret not found")
		}
		return json.Unmarshal(data, &secret)
	})
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

// ListSecrets возвращает все секреты без soft-deleted.
func (s *Storage) ListSecrets() ([]*LocalSecret, error) {
	var secrets []*LocalSecret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		return b.ForEach(func(_, v []byte) error {
			var s LocalSecret
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			if s.DeletedAt == nil {
				secrets = append(secrets, &s)
			}
			return nil
		})
	})
	return secrets, err
}

// DeleteSecret помечает секрет как удалённый.
func (s *Storage) DeleteSecret(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("secret not found")
		}
		var secret LocalSecret
		if err := json.Unmarshal(data, &secret); err != nil {
			return err
		}
		now := time.Now()
		secret.DeletedAt = &now
		updated, err := json.Marshal(secret)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), updated)
	})
}

// GetLastSyncAt возвращает время последней синхронизации.
func (s *Storage) GetLastSyncAt() (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMeta)
		data := b.Get([]byte("last_sync_at"))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &t)
	})
	return t, err
}

// SetLastSyncAt сохраняет время последней синхронизации.
func (s *Storage) SetLastSyncAt(t time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMeta)
		data, err := json.Marshal(t)
		if err != nil {
			return err
		}
		return b.Put([]byte("last_sync_at"), data)
	})
}
