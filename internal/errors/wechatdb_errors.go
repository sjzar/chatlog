package errors

import (
	"net/http"
	"time"
)

var (
	ErrTalkerEmpty     = New(nil, http.StatusBadRequest, "talker empty").WithStack()
	ErrKeyEmpty        = New(nil, http.StatusBadRequest, "key empty").WithStack()
	ErrMediaNotFound   = New(nil, http.StatusNotFound, "media not found").WithStack()
	ErrKeyLengthMust32 = New(nil, http.StatusBadRequest, "key length must be 32 bytes").WithStack()
)

// 数据库初始化相关错误
func DBFileNotFound(path, pattern string, cause error) *Error {
	return Newf(cause, http.StatusNotFound, "db file not found %s: %s", path, pattern).WithStack()
}

func DBConnectFailed(path string, cause error) *Error {
	return Newf(cause, http.StatusInternalServerError, "db connect failed: %s", path).WithStack()
}

func DBInitFailed(cause error) *Error {
	return New(cause, http.StatusInternalServerError, "db init failed").WithStack()
}

func TalkerNotFound(talker string) *Error {
	return Newf(nil, http.StatusNotFound, "talker not found: %s", talker).WithStack()
}

func DBCloseFailed(cause error) *Error {
	return New(cause, http.StatusInternalServerError, "db close failed").WithStack()
}

func QueryFailed(query string, cause error) *Error {
	return Newf(cause, http.StatusInternalServerError, "query failed: %s", query).WithStack()
}

func ScanRowFailed(cause error) *Error {
	return New(cause, http.StatusInternalServerError, "scan row failed").WithStack()
}

func TimeRangeNotFound(start, end time.Time) *Error {
	return Newf(nil, http.StatusNotFound, "time range not found: %s - %s", start, end).WithStack()
}

func MediaTypeUnsupported(_type string) *Error {
	return Newf(nil, http.StatusBadRequest, "unsupported media type: %s", _type).WithStack()
}

func ChatRoomNotFound(key string) *Error {
	return Newf(nil, http.StatusNotFound, "chat room not found: %s", key).WithStack()
}

func ContactNotFound(key string) *Error {
	return Newf(nil, http.StatusNotFound, "contact not found: %s", key).WithStack()
}

func InitCacheFailed(cause error) *Error {
	return New(cause, http.StatusInternalServerError, "init cache failed").WithStack()
}

func FileGroupNotFound(name string) *Error {
	return Newf(nil, http.StatusNotFound, "file group not found: %s", name).WithStack()
}
