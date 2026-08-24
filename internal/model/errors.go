package model

import "errors"

// 领域错误：供 service 层做错误映射（HTTP 状态码）与业务分支判断。
var (
	ErrNotFound         = errors.New("not found")
	ErrDuplicate        = errors.New("duplicate")
	ErrUnknownNode      = errors.New("unknown node")
	ErrUnitMismatch     = errors.New("offset unit mismatch")
	ErrTopologyCycle    = errors.New("undeclared topology cycle")
	ErrSealed           = errors.New("event sealed: cannot modify")
	ErrInvalidState     = errors.New("invalid state transition")
	ErrSnapshotArchived = errors.New("snapshot archived")
	ErrNegativeDelay    = errors.New("negative roundtrip delay")
	ErrOffsetOverflow   = errors.New("offset out of representable range")
)

// IsNotFound 判断错误是否为未找到。
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsDuplicate 判断错误是否为重复（幂等冲突）。
func IsDuplicate(err error) bool { return errors.Is(err, ErrDuplicate) }

// IsSealed 判断错误是否为封存事件被修改。
func IsSealed(err error) bool { return errors.Is(err, ErrSealed) }
