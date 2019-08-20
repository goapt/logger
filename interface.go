package logger

import "github.com/sirupsen/logrus"

// ILogger is the logger interface
type IBaseLogger interface {
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
}

// ILogger is the logger interface
type ILogger interface {
	IBaseLogger
	AddHook(hook logrus.Hook)
	WithFields(map[string]interface{}) ILogger
	Data(interface{}) ILogger
}
