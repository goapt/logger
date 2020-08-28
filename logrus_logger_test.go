package logger

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLogrusLogger(t *testing.T) {
	log := NewLogrusLogger(defaultConfig, func(l *LogrusLogger) {
		l.Level = logrus.DebugLevel
	})
	log.Debugf("test %s", "cccc")

	log.Info(errors.New("sdfsdfsdf"))
}
