package postgres

import "errors"

// ErrConflict возвращается при конфликте версий при обновлении.
var ErrConflict = errors.New("version conflict")

// ErrNotFound возвращается когда запись не найдена или принадлежит другому пользователю.
var ErrNotFound = errors.New("not found")

// ErrDuplicate возвращается при нарушении ограничения уникальности.
var ErrDuplicate = errors.New("duplicate")
