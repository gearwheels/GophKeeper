// Package migrations содержит SQL-миграции базы данных и предоставляет
// встроенную файловую систему для их применения при старте сервера.
package migrations

import "embed"

// FS — встроенная файловая система со всеми SQL-миграциями.
//
//go:embed *.sql
var FS embed.FS
