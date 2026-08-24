package store

import (
	"database/sql"
	"errors"
	"strings"
)

// isUniqueViolation 判断错误是否为 SQLite UNIQUE 约束冲突。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}

// isNoRows 判断错误是否为无结果。
func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
