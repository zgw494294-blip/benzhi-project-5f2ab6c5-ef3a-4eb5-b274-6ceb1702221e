package domain

import "errors"

var (
	ErrValidation  = errors.New("数据校验失败")
	ErrNotFound    = errors.New("记录不存在")
	ErrConflict    = errors.New("版本冲突")
	ErrTransition  = errors.New("状态迁移不合法")
	ErrIdempotency = errors.New("幂等键已用于其他命令")
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []FieldError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ErrValidation.Error()
	}
	return e[0].Field + ": " + e[0].Message
}

func (e ValidationErrors) Unwrap() error { return ErrValidation }

func AddRequired(errs ValidationErrors, field, value string) ValidationErrors {
	if value == "" {
		return append(errs, FieldError{Field: field, Message: "不能为空"})
	}
	return errs
}
