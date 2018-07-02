package logger

import (
	"testing"
	"github.com/sirupsen/logrus"
)

func TestNewLogrusLogger(t *testing.T) {
	log := NewLogrusLogger(func(l *logrus.Logger) {
		l.Level = logrus.DebugLevel
	})
	log.Debug("test %s","cccc")
}
