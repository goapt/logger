package logger

import "github.com/sirupsen/logrus"

// ILogger is the logger interface
type ILogger interface {
	Fatalf(string, ...interface{})
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Tracef(string, ...interface{})
	Fatal(...interface{})
	Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Error(...interface{})
	Trace(...interface{})
	AddHook(hook logrus.Hook)
	WithFields(map[string]interface{}) *logrus.Entry
	Data(interface{}) *logrus.Entry
}
