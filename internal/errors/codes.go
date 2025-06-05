package errors

import "strings"

// ErrorCode はエラーコードを表します。
type ErrorCode string

// ErrorCategory はエラーカテゴリを表します。
type ErrorCategory string

const (
	ErrorCategoryFile    ErrorCategory = "FILE"
	ErrorCategoryPort    ErrorCategory = "PORT"
	ErrorCategoryDocker  ErrorCategory = "DOCKER"
	ErrorCategoryConfig  ErrorCategory = "CONFIG"
	ErrorCategoryProcess ErrorCategory = "PROCESS"
	ErrorCategoryUnknown ErrorCategory = "UNKNOWN"
)

// ファイル関連エラー
const (
	ErrFileNotFound    ErrorCode = "FILE_NOT_FOUND"
	ErrFilePermission  ErrorCode = "FILE_PERMISSION"
	ErrFileInvalidYAML ErrorCode = "FILE_INVALID_YAML"
	ErrFileWriteFailed ErrorCode = "FILE_WRITE_FAILED"
	ErrFileReadFailed  ErrorCode = "FILE_READ_FAILED"
)

// ポート関連エラー
const (
	ErrPortUnavailable      ErrorCode = "PORT_UNAVAILABLE"
	ErrPortRangeInvalid     ErrorCode = "PORT_RANGE_INVALID"
	ErrPortConflict         ErrorCode = "PORT_CONFLICT"
	ErrPortScanFailed       ErrorCode = "PORT_SCAN_FAILED"
	ErrPortAllocationFailed ErrorCode = "PORT_ALLOCATION_FAILED"
)

// Docker関連エラー
const (
	ErrDockerNotFound  ErrorCode = "DOCKER_NOT_FOUND"
	ErrComposeInvalid  ErrorCode = "COMPOSE_INVALID"
	ErrComposeNotFound ErrorCode = "COMPOSE_NOT_FOUND"
	ErrDockerAPIFailed ErrorCode = "DOCKER_API_FAILED"
)

// 設定関連エラー
const (
	ErrConfigInvalid    ErrorCode = "CONFIG_INVALID"
	ErrConfigNotFound   ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigLoadFailed ErrorCode = "CONFIG_LOAD_FAILED"
)

// プロセス関連エラー
const (
	ErrProcessNotFound    ErrorCode = "PROCESS_NOT_FOUND"
	ErrProcessStartFailed ErrorCode = "PROCESS_START_FAILED"
	ErrProcessStopFailed  ErrorCode = "PROCESS_STOP_FAILED"
)

// 汎用エラー
const (
	ErrUnknown          ErrorCode = "UNKNOWN"
	ErrInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrValidationFailed ErrorCode = "VALIDATION_FAILED"
)

// Category はエラーコードのカテゴリを返します。
func (c ErrorCode) Category() ErrorCategory {
	parts := strings.Split(string(c), "_")
	if len(parts) == 0 {
		return ErrorCategoryUnknown
	}

	switch parts[0] {
	case "FILE":
		return ErrorCategoryFile
	case "PORT":
		return ErrorCategoryPort
	case "DOCKER", "COMPOSE":
		return ErrorCategoryDocker
	case "CONFIG":
		return ErrorCategoryConfig
	case "PROCESS":
		return ErrorCategoryProcess
	default:
		return ErrorCategoryUnknown
	}
}

// String はエラーコードを文字列として返します。
func (c ErrorCode) String() string {
	return string(c)
}
