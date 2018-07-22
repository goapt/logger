package logger

import (
	"testing"
	"github.com/sirupsen/logrus"
)

func TestNewLogrusLogger(t *testing.T) {
	log := NewLogrusLogger(func(l *LogrusLogger) {
		l.Level = logrus.DebugLevel
	})
	log.Debugf("test %s","cccc")
}
