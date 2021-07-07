package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/goapt/logrus-sentry-hook"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LogName       string `toml:"log_name" json:"log_name"`
	LogFormat     string `toml:"log_format" json:"log_format"`
	LogPath       string `toml:"log_path" json:"log_path"`
	LogMode       string `toml:"log_mode" json:"log_mode"`
	LogLevel      string `toml:"log_level" json:"log_level"`
	LogDetail     bool   `toml:"log_detail" json:"log_detail"`
	LogMaxFiles   int    `toml:"log_max_files" json:"log_max_files"`
	LogSentryDSN  string `toml:"log_sentry_dsn" json:"log_sentry_dsn"`
	LogSentryType string `toml:"log_sentry_type" json:"log_sentry_type"`
	LogSkip       int
	LogWriter     io.Writer
}

var (
	DefaultLogger ILogger
	defaultConfig *Config
)

func init() {
	defaultConfig = &Config{
		LogName:  "app",
		LogMode:  "std",
		LogLevel: "debug",
	}
	DefaultLogger = newLogger(defaultConfig)
}

func Setting(option func(c *Config)) {
	option(defaultConfig)
	DefaultLogger = newLogger(defaultConfig)
}

func NewLogger(options ...func(c *Config)) ILogger {
	// clone default config
	conf := *defaultConfig
	for _, option := range options {
		option(&conf)
	}
	return newLogger(&conf)
}

func newLogger(conf *Config) ILogger {
	return NewLogrusLogger(conf, func(l *LogrusLogger) {
		l.Level, _ = logrus.ParseLevel(conf.LogLevel)

		if conf.LogFormat == "text" {
			l.SetFormatter(&logrus.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			})
		}

		if conf.LogMode == "file" {
			if writer, err := NewFileWriter(conf); err == nil {
				l.SetOutput(writer)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "new file writer error %s", err)
			}
		}

		if conf.LogMode == "custom" {
			l.SetOutput(conf.LogWriter)
		}

		if conf.LogSentryDSN != "" {
			hook, err := sentry.NewHook(sentry.Options{
				Dsn:              conf.LogSentryDSN,
				AttachStacktrace: true,
			},
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.WarnLevel,
				logrus.InfoLevel,
			)

			if err == nil {
				hook.SetTags(map[string]string{
					"type": conf.LogSentryType,
				})
				l.Hooks.Add(hook)
			}
		}
	})
}

func AddHook(hook logrus.Hook) {
	DefaultLogger.AddHook(hook)
}

func Debugf(str string, args ...interface{}) {
	DefaultLogger.Debugf(str, args...)
}

func Infof(str string, args ...interface{}) {
	DefaultLogger.Infof(str, args...)
}

func Warnf(str string, args ...interface{}) {
	DefaultLogger.Warnf(str, args...)
}

func Errorf(str string, args ...interface{}) {
	DefaultLogger.Errorf(str, args...)
}

func Fatalf(str string, args ...interface{}) {
	DefaultLogger.Fatalf(str, args...)
}

func Tracef(str string, args ...interface{}) {
	DefaultLogger.Tracef(str, args...)
}

func Debug(args ...interface{}) {
	DefaultLogger.Debug(args...)
}

func Info(args ...interface{}) {
	DefaultLogger.Info(args...)
}

func Warn(args ...interface{}) {
	DefaultLogger.Warn(args...)
}

func Error(args ...interface{}) {
	DefaultLogger.Error(args...)
}

func Fatal(args ...interface{}) {
	DefaultLogger.Fatal(args...)
}

func Trace(args ...interface{}) {
	DefaultLogger.Trace(args...)
}

func WithFields(fields map[string]interface{}) *logrus.Entry {
	return DefaultLogger.WithFields(fields)
}

func Data(v interface{}) *logrus.Entry {
	return DefaultLogger.Data(v)
}

func Skip(i int) ILogger {
	return NewLogger(func(c *Config) {
		c.LogSkip = i
	})
}
