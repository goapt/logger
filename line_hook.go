package logger

import (
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	maximumCallerDepth int = 25
	minimumCallerDepth int = 7
)

type LineHook struct {
	conf *Config
	mu   *sync.RWMutex
}

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
	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)
		// If the caller isn't part of this package, we're done
		if !strings.HasSuffix(pkg, "github.com/goapt/logger") && !strings.HasSuffix(pkg, "github.com/sirupsen/logrus") {
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

func NewLineHook(conf *Config) (*LineHook, error) {
	hook := &LineHook{
		conf: conf,
		mu:   &sync.RWMutex{},
	}
	return hook, nil
}

func (h *LineHook) Fire(entry *logrus.Entry) error {
	if h.conf.LogDetail {
		h.mu.Lock()
		defer h.mu.Unlock()
		caller := getCaller(h.conf.LogSkip)

		if caller == nil {
			return nil
		}

		entry.Data["file"] = caller.File
		entry.Data["func"] = path.Base(caller.Function)
		entry.Data["line"] = caller.Line
	}
	return nil
}

func (h *LineHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
