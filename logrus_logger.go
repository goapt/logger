package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var _ ILogger = (*LogrusLogger)(nil)
// LogrusLogger file logger
type LogrusLogger struct {
	*logrus.Logger
}

// NewFileLogger providers a file logger based on logrus
func NewLogrusLogger(options ...func(*logrus.Logger)) (ILogger) {
	l := &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		},
		Hooks: make(logrus.LevelHooks),
	}

	for _, option := range options {
		option(l)
	}

	return &LogrusLogger{
		l,
	}
}

func (l *LogrusLogger) Debug(format string, args ...interface{}) {
	if l.Logger.Out == os.Stderr {
		if len(args) > 0 {
			l.Logger.Debugf(format, args)
		} else {
			l.Logger.Debug(format)
		}
	} else {
		if len(args) > 0 {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Debugf(format, args...)
		} else {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Debug(format)
		}
	}
}

func (l *LogrusLogger) Info(format string, args ...interface{}) {
	if l.Logger.Out == os.Stderr {
		if len(args) > 0 {
			l.Logger.Infof(format, args)
		} else {
			l.Logger.Info(format)
		}
	} else {
		if len(args) > 0 {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Infof(format, args...)
		} else {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Info(format)
		}
	}
}

func (l *LogrusLogger) Error(format string, args ...interface{}) {
	if l.Logger.Out == os.Stderr {
		if len(args) > 0 {
			l.Logger.Errorf(format, args)
		} else {
			l.Logger.Error(format)
		}
	} else {
		if len(args) > 0 {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Errorf(format, args...)
		} else {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Error(format)
		}
	}
}

func (l *LogrusLogger) Fatal(format string, args ...interface{}) {
	if l.Logger.Out == os.Stderr {
		if len(args) > 0 {
			l.Logger.Fatalf(format, args)
		} else {
			l.Logger.Fatal(format)
		}
	} else {
		if len(args) > 0 {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Fatalf(format, args...)
		} else {
			l.Logger.WithFields(logrus.Fields{
				"fingerprint": []string{format},
			}).Fatal(format)
		}
	}
}
