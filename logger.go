package logger

import (
	"time"

	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LogName       string `toml:"log_name" json:"log_name"`
	LogPath       string `toml:"log_path" json:"log_path"`
	LogMode       string `toml:"log_mode" json:"log_mode"`
	LogLevel      string `toml:"log_level" json:"log_level"`
	LogMaxFiles   int    `toml:"log_max_files" json:"log_max_files"`
	LogSentryDSN  string `toml:"log_sentry_dsn" json:"log_sentry_dsn"`
	LogSentryType string `toml:"log_sentry_type" json:"log_sentry_type"`
}

var (
	// std is the name of the standard logger in stdlib `log`
	std           ILogger
	defaultConfig *Config
)

func init() {
	defaultConfig = &Config{
		LogName:  "app",
		LogMode:  "std",
		LogLevel: "debug",
	}
	std = NewLogger()
}

func Setting(options ...func(*Config)) {
	for _, option := range options {
		option(defaultConfig)
	}

	std = NewLogger(options...)
}

func NewLogger(options ...func(*Config)) ILogger {
	conf := *defaultConfig

	for _, option := range options {
		option(&conf)
	}

	return newLogger(&conf)
}

func newLogger(conf *Config) ILogger {
	return NewLogrusLogger(func(l *logrus.Logger) {

		switch conf.LogLevel {
		case LevelDebug:
			l.Level = logrus.DebugLevel
		case LevelInfo:
			l.Level = logrus.InfoLevel
		case LevelError:
			l.Level = logrus.ErrorLevel
		case LevelFatal:
			l.Level = logrus.FatalLevel
		}

		if conf.LogMode == "file" {
			hook, err := NewFileHook(conf)
			if err == nil {
				l.Hooks.Add(hook)
			}
		}

		if conf.LogSentryDSN != "" {
			tags := map[string]string{
				"type": conf.LogSentryType,
			}

			hook, err := logrus_sentry.NewWithTagsSentryHook(conf.LogSentryDSN, tags, []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.InfoLevel,
			})
			hook.Timeout = 1 * time.Second
			hook.StacktraceConfiguration.Enable = true

			if err == nil {
				l.Hooks.Add(hook)
			}
		}
	})
}

func Debug(str string, args ...interface{}) {
	std.Debug(str, args...)
}

func Info(str string, args ...interface{}) {
	std.Info(str, args...)
}

func Error(str string, args ...interface{}) {
	std.Error(str, args...)
}

func Fatal(str string, args ...interface{}) {
	std.Fatal(str, args...)
}
