package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

// FileHook to send logs via syslog.
type FileHook struct {
	conf  *Config
	mu    sync.RWMutex
	cache sync.Map
}

func NewFileHook(conf *Config) (*FileHook, error) {

	if _, err := os.Stat(conf.LogPath); err != nil {
		err = os.MkdirAll(conf.LogPath, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("can't mkdirall directory: path = %v, err = %v", conf.LogPath, err)
		}
	}

	hook := &FileHook{
		conf: conf,
	}
	return hook, nil
}

func (h *FileHook) Fire(entry *logrus.Entry) error {
	logFile := filepath.Join(h.conf.LogPath, h.conf.LogName+"-"+h.conf.LogRotate.Current()+".log")

	var logWriter *os.File
	if f, ok := h.cache.Load(logFile); !ok {
		h.mu.Lock()
		defer h.mu.Unlock()

		//Close yesteday file handler
		prevFile := filepath.Join(h.conf.LogPath, h.conf.LogName+"-"+h.conf.LogRotate.Prev(1)+".log")
		if f, ok := h.cache.Load(prevFile); ok {
			f.(*os.File).Close()
		}

		//Delete old log file
		if h.conf.LogMaxFiles > 0 {
			oldFile := filepath.Join(h.conf.LogPath, h.conf.LogName+"-"+h.conf.LogRotate.Prev(h.conf.LogMaxFiles)+".log")
			h.cache.Delete(oldFile)
			os.Remove(oldFile)
		}

		var err error
		logWriter, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("can't open file: path = %v, err = %v", logFile, err)
		}
		h.cache.Store(logFile, logWriter)
	} else {
		logWriter = f.(*os.File)
	}

	entry.Logger.Out = logWriter
	return nil
}

func (h *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
