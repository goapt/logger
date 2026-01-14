package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/goapt/logger/rolling"
)

type Mode string

const (
	ModeFile   Mode = "file"
	ModeStd    Mode = "std"
	ModeCustom Mode = "custom"
)

type Config struct {
	Mode     Mode       `json:"mode" yaml:"mode"`           // default  std
	Level    slog.Level `json:"level" yaml:"level"`         // default info
	FileName string     `json:"filename" yaml:"filename"`   // only used for file mode
	MaxFiles int        `json:"max_files" yaml:"max_files"` // default keep the last 3 files
	MaxSize  int64      `json:"max_size" yaml:"max_size"`   // default 200MB
	Detail   bool       `json:"detail" yaml:"detail"`       // add file path and line number
	Writer   io.Writer  `json:"-" yaml:"-"`                 // only used for custom mode
}

func New(conf *Config) *slog.Logger {
	if conf.Mode == "" {
		conf.Mode = ModeStd
	}
	return slog.New(newHandler(conf))
}

var defaultLogger atomic.Pointer[slog.Logger]

func init() {
	defaultLogger.Store(New(&Config{}))
}

// Default returns the standard logger used by the package-level output functions.
func Default() *slog.Logger { return defaultLogger.Load() }

func SetDefault(logger *slog.Logger) { defaultLogger.Store(logger) }

func newRoller(conf *Config) *rolling.Roller {
	if conf.MaxFiles == 0 {
		conf.MaxFiles = 3
	}

	if conf.FileName == "" {
		conf.FileName = "app"
	}

	if conf.MaxSize == 0 {
		conf.MaxSize = 1024 * 1024 * 200
	}

	roller, err := rolling.NewRoller(conf.FileName, conf.MaxSize, rolling.WithMaxBackups(conf.MaxFiles), rolling.WithMaxAge(3))
	if err != nil {
		panic(fmt.Sprintf("new file writer error %s", err))
	}
	return roller
}

func newHandler(conf *Config) slog.Handler {
	isStdout := false
	var w io.Writer
	switch conf.Mode {
	case ModeFile:
		w = newRoller(conf)
	case ModeCustom:
		w = conf.Writer
	default:
		w = os.Stdout
		isStdout = true
	}

	if os.Getenv("DEBUG_LOG") == "true" && !isStdout {
		w = io.MultiWriter(w, os.Stdout)
	}

	opts := &slog.HandlerOptions{
		AddSource: conf.Detail,
		Level:     conf.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindTime {
				return slog.String(a.Key, a.Value.Time().Format("2006-01-02 15:04:05.000"))
			}
			return a
		},
	}

	return slog.NewJSONHandler(w, opts)
}
