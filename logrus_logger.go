package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var _ ILogger = (*LogrusLogger)(nil)

// LogrusLogger file logger
type LogrusLogger struct {
	*logrus.Logger
	conf   *Config
	fields logrus.Fields
	lock   sync.Mutex
}

// NewFileLogger providers a file logger based on logrus
func NewLogrusLogger(conf *Config, option func(l *LogrusLogger)) ILogger {
	l := &LogrusLogger{
		Logger: &logrus.Logger{
			Out: os.Stderr,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			},
			Hooks: make(logrus.LevelHooks),
		},
		conf:   conf,
		fields: make(map[string]interface{}),
	}
	option(l)
	return l
}

func (l *LogrusLogger) withFinger(format string) IBaseLogger {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.fields["fingerprint"] = []string{format}

	if l.conf.LogDetail {
		if caller := getCaller(l.conf.LogSkip); caller != nil {
			l.fields["file"] = caller.File
			l.fields["func"] = path.Base(caller.Function)
			l.fields["line"] = caller.Line
		}
	}

	return l.Logger.WithFields(l.fields)
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) ILogger {
	return &LogrusLogger{
		Logger: l.Logger,
		conf:   l.conf,
		fields: fields,
	}
}

func (l *LogrusLogger) Data(v interface{}) ILogger {
	return l.WithFields(logrus.Fields{
		"data": v,
	})
}

func (l *LogrusLogger) AddHook(hook logrus.Hook) {
	l.Logger.AddHook(hook)
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.withFinger(format).Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.withFinger(format).Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.withFinger(format).Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.withFinger(format).Fatalf(format, args...)
}

func (l *LogrusLogger) Tracef(format string, args ...interface{}) {
	l.withFinger(format).Tracef(format, args...)
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Debug(args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Info(args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Warn(args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Error(args...)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Fatal(args...)
}

func (l *LogrusLogger) Trace(args ...interface{}) {
	l.withFinger(argsFormat(args...)).Trace(args...)
}

func argsFormat(args ...interface{}) string {
	format := ""
	if len(args) > 0 {
		format = fmt.Sprint(args[0])
	}

	return format
}

const (
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 4
)

var (
	// qualified package name, cached at first use
	logrusPackage string

	// Positions in the call stack when tracing to report the calling method
	minimumCallerDepth = 1

	// Used for caller information initialisation
	callerInitOnce sync.Once
)

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}

// getCaller retrieves the name of the first non-logrus calling function
func getCaller(skip int) *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)
		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			if skip == 0 {
				return &f
			} else {
				skip--
			}
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}
