package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var _ ILogger = (*LogrusLogger)(nil)
// LogrusLogger file logger
type LogrusLogger struct {
	*logrus.Logger
	Conf *Config
}

// NewFileLogger providers a file logger based on logrus
func NewLogrusLogger(option func(l *LogrusLogger)) (ILogger) {
	l := &LogrusLogger{
		Logger: &logrus.Logger{
			Out: os.Stderr,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			},
			Hooks: make(logrus.LevelHooks),
		},
		Conf:&Config{},
	}
	option(l)
	return l
}

func (l *LogrusLogger) withFields(format string) ILogger {
	if l.Conf.LogSentryDSN != "" {
		return l.Logger.WithFields(logrus.Fields{
			"fingerprint": []string{format},
		})
	}

	return l.Logger
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.withFields(format).Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.withFields(format).Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.withFields(format).Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.withFields(format).Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.withFields(format).Fatalf(format, args...)
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.Logger.Debug(args)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.Logger.Info(args)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.Logger.Warn(args)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.Logger.Error(args)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args)
}
