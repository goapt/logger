package logger

import (
	"errors"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestNewLogrusLogger(t *testing.T) {
	log := NewLogrusLogger(func(l *LogrusLogger) {
		l.Level = logrus.DebugLevel
	})
	log.Debugf("test %s", "cccc")

	log.Info(errors.New("sdfsdfsdf"))
}
