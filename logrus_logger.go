package logger

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var _ ILogger = (*LogrusLogger)(nil)

// LogrusLogger file logger
type LogrusLogger struct {
	*logrus.Logger
	Fingerprint bool
	fields      map[string]interface{}
}

// NewFileLogger providers a file logger based on logrus
func NewLogrusLogger(option func(l *LogrusLogger)) ILogger {
	l := &LogrusLogger{
		Logger: &logrus.Logger{
			Out: os.Stderr,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			},
			Hooks: make(logrus.LevelHooks),
		},
		fields: make(map[string]interface{}),
	}
	option(l)
	return l
}

func (l *LogrusLogger) withFinger(format string) ILogger {
	if l.Fingerprint {
		return l.WithFields(map[string]interface{}{
			"fingerprint": []string{format},
		})
	}

	return l
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) ILogger {
	for k, v := range fields {
		l.fields[k] = v
	}

	return l
}

func (l *LogrusLogger) Data(v interface{}) ILogger {
	return l.WithFields(map[string]interface{}{
		"data": v,
	})
}

func (l *LogrusLogger) AddHook(hook logrus.Hook) {
	l.Logger.AddHook(hook)
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Fatalf(format, args...)
}

func (l *LogrusLogger) Tracef(format string, args ...interface{}) {
	l.withFinger(format)
	l.Logger.WithFields(l.fields).Tracef(format, args...)
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Debug(args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Info(args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Warn(args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Error(args...)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Fatal(args...)
}

func (l *LogrusLogger) Trace(args ...interface{}) {
	l.withFinger(argsFormat(args...))
	l.Logger.WithFields(l.fields).Trace(args...)
}

func argsFormat(args ...interface{}) string {
	format := ""
	if len(args) > 0 {
		format = fmt.Sprint(args[0])
	}

	return format
}
