package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Message string   `json:"message"` // 错误消息
	Cause   error    `json:"-"`       // 原始错误
	Code    int      `json:"-"`       // HTTP Code
	Stack   []string `json:"-"`       // 错误堆栈
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("%s", e.Message)
}

func (e *Error) String() string {
	return e.Error()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) WithStack() *Error {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	stack := make([]string, 0, n)
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}

	e.Stack = stack
	return e
}

func New(cause error, code int, message string) *Error {
	return &Error{
		Message: message,
		Cause:   cause,
		Code:    code,
	}
}

func Newf(cause error, code int, format string, args ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
		Code:    code,
	}
}

func Wrap(err error, message string, code int) *Error {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*Error); ok {
		return &Error{
			Message: message,
			Cause:   appErr.Cause,
			Code:    appErr.Code,
			Stack:   appErr.Stack,
		}
	}

	return New(err, code, message)
}

func GetCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	return http.StatusInternalServerError
}

func RootCause(err error) error {
	for err != nil {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
	return err
}

func Err(c *gin.Context, err error) {
	if appErr, ok := err.(*Error); ok {
		c.JSON(appErr.Code, appErr.Error())
		return
	}

	c.JSON(http.StatusInternalServerError, err.Error())
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
