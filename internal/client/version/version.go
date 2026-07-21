// Package version содержит информацию о версии и дате сборки бинарного файла клиента.
package version

// Переменные устанавливаются через -ldflags при сборке.
var (
	// Version — семантическая версия приложения.
	Version = "dev"
	// BuildDate — дата и время сборки в формате RFC3339.
	BuildDate = "unknown"
	// Commit — короткий хеш git-коммита.
	Commit = "unknown"
)
