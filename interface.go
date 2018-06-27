package logger

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelError = "error"
	LevelFatal = "fatal"
)

// ILogger is the logger interface
type ILogger interface {
	Fatal(string, ...interface{})
	Debug(string, ...interface{})
	Info(string, ...interface{})
	Error(string, ...interface{})
}
