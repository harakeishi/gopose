// Package file は、ファイル操作の抽象化機能を提供します。
package file

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/harakeishi/gopose/pkg/types"
)

// FileManager はファイルの基本操作を行うインターフェースです。
type FileManager interface {
	Exists(ctx context.Context, path string) (bool, error)
	Read(ctx context.Context, path string) ([]byte, error)
	Write(ctx context.Context, path string, data []byte) error
	Delete(ctx context.Context, path string) error
	Copy(ctx context.Context, src, dst string) error
	Move(ctx context.Context, src, dst string) error
	GetInfo(ctx context.Context, path string) (os.FileInfo, error)
}

// YAMLWriter はYAML形式でのファイル出力を行うインターフェースです。
type YAMLWriter interface {
	WriteYAML(ctx context.Context, path string, data interface{}) error
	WriteYAMLToWriter(ctx context.Context, writer io.Writer, data interface{}) error
	WriteOverrideConfig(ctx context.Context, path string, config *types.OverrideConfig) error
}

// BackupManager はファイルのバックアップ管理を行うインターフェースです。
type BackupManager interface {
	CreateBackup(ctx context.Context, filePath string) (string, error)
	RestoreBackup(ctx context.Context, backupPath string, originalPath string) error
	ListBackups(ctx context.Context, originalPath string) ([]BackupInfo, error)
	CleanupOldBackups(ctx context.Context, originalPath string, maxAge time.Duration) error
}

// FileWatcher はファイル変更監視を行うインターフェースです。
type FileWatcher interface {
	Watch(ctx context.Context, path string) (<-chan types.FileWatchEvent, error)
	WatchMultiple(ctx context.Context, paths []string) (<-chan types.FileWatchEvent, error)
	Stop(ctx context.Context) error
}

// AtomicWriter は原子的ファイル書き込みを行うインターフェースです。
type AtomicWriter interface {
	WriteAtomic(ctx context.Context, path string, data []byte) error
	WriteAtomicWithMode(ctx context.Context, path string, data []byte, mode os.FileMode) error
}

// TemplateManager はテンプレートファイル管理を行うインターフェースです。
type TemplateManager interface {
	LoadTemplate(ctx context.Context, templatePath string) (string, error)
	RenderTemplate(ctx context.Context, templateContent string, data interface{}) (string, error)
	SaveFromTemplate(ctx context.Context, templatePath string, outputPath string, data interface{}) error
}

// BackupInfo はバックアップ情報を表します。
type BackupInfo struct {
	Path         string    `json:"path"`
	OriginalPath string    `json:"original_path"`
	CreatedAt    time.Time `json:"created_at"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum"`
}

// FileOperationResult はファイル操作の結果を表します。
type FileOperationResult struct {
	Success      bool          `json:"success"`
	Path         string        `json:"path"`
	Operation    string        `json:"operation"`
	Duration     time.Duration `json:"duration"`
	Error        error         `json:"error,omitempty"`
	BytesWritten int64         `json:"bytes_written,omitempty"`
	BytesRead    int64         `json:"bytes_read,omitempty"`
}

// WriteOptions は書き込みオプションを表します。
type WriteOptions struct {
	Mode       os.FileMode `json:"mode"`
	CreateDirs bool        `json:"create_dirs"`
	Backup     bool        `json:"backup"`
	Atomic     bool        `json:"atomic"`
	Overwrite  bool        `json:"overwrite"`
}

// ReadOptions は読み込みオプションを表します。
type ReadOptions struct {
	MaxSize     int64 `json:"max_size"`
	BufferSize  int   `json:"buffer_size"`
	FollowLinks bool  `json:"follow_links"`
}
