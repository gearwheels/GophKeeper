package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timofeevav/gophkeeper/internal/client/storage"
)

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestStorage_UpsertAndGet(t *testing.T) {
	db := newTestStorage(t)

	secret := &storage.LocalSecret{
		ID:        "secret-id-1",
		Type:      "login_password",
		Name:      "GitHub",
		Data:      []byte("encrypted-data"),
		Meta:      "dev account",
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, db.UpsertSecret(secret))

	got, err := db.GetSecret("secret-id-1")
	require.NoError(t, err)
	assert.Equal(t, secret.ID, got.ID)
	assert.Equal(t, secret.Name, got.Name)
	assert.Equal(t, secret.Data, got.Data)
	assert.Equal(t, secret.Version, got.Version)
}

func TestStorage_GetSecret_NotFound(t *testing.T) {
	db := newTestStorage(t)
	_, err := db.GetSecret("nonexistent-id")
	assert.Error(t, err)
}

func TestStorage_ListSecrets(t *testing.T) {
	db := newTestStorage(t)

	for i, name := range []string{"GitHub", "Google", "AWS"} {
		require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
			ID:      "id-" + string(rune('0'+i)),
			Type:    "login_password",
			Name:    name,
			Data:    []byte("data"),
			Version: 1,
		}))
	}

	secrets, err := db.ListSecrets()
	require.NoError(t, err)
	assert.Len(t, secrets, 3)
}

func TestStorage_ListSecrets_ExcludesDeleted(t *testing.T) {
	db := newTestStorage(t)

	require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
		ID:   "active",
		Type: "text",
		Name: "Active",
		Data: []byte("data"),
	}))

	deleted := time.Now()
	require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
		ID:        "deleted",
		Type:      "text",
		Name:      "Deleted",
		Data:      []byte("data"),
		DeletedAt: &deleted,
	}))

	secrets, err := db.ListSecrets()
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
	assert.Equal(t, "active", secrets[0].ID)
}

func TestStorage_DeleteSecret(t *testing.T) {
	db := newTestStorage(t)

	require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
		ID:   "to-delete",
		Type: "text",
		Name: "ToDelete",
		Data: []byte("data"),
	}))

	require.NoError(t, db.DeleteSecret("to-delete"))

	secrets, err := db.ListSecrets()
	require.NoError(t, err)
	assert.Empty(t, secrets)
}

func TestStorage_DeleteSecret_NotFound(t *testing.T) {
	db := newTestStorage(t)
	err := db.DeleteSecret("nonexistent")
	assert.Error(t, err)
}

func TestStorage_LastSyncAt(t *testing.T) {
	db := newTestStorage(t)

	// Изначально время нулевое
	t1, err := db.GetLastSyncAt()
	require.NoError(t, err)
	assert.True(t, t1.IsZero())

	// Сохраняем время
	now := time.Now().Truncate(time.Millisecond)
	require.NoError(t, db.SetLastSyncAt(now))

	t2, err := db.GetLastSyncAt()
	require.NoError(t, err)
	assert.Equal(t, now, t2)
}

func TestStorage_Upsert_Updates(t *testing.T) {
	db := newTestStorage(t)

	require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
		ID:      "upd-1",
		Name:    "Original",
		Data:    []byte("v1"),
		Version: 1,
	}))

	require.NoError(t, db.UpsertSecret(&storage.LocalSecret{
		ID:      "upd-1",
		Name:    "Updated",
		Data:    []byte("v2"),
		Version: 2,
	}))

	got, err := db.GetSecret("upd-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, int64(2), got.Version)
}

func TestStorage_Open_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "nested")
	db, err := storage.Open(filepath.Join(dir, "data.db"))
	require.NoError(t, err)
	defer db.Close()

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}
