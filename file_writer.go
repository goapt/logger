package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileWriter struct {
	conf        *Config
	mu          sync.Mutex
	currLogFile string
	writer      *os.File
}

func NewFileWriter(conf *Config) (*FileWriter, error) {

	if _, err := os.Stat(conf.LogPath); err != nil {
		err = os.MkdirAll(conf.LogPath, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("can't mkdirall directory: path = %v, err = %v", conf.LogPath, err)
		}
	}
	writer := &FileWriter{
		conf: conf,
	}

	return writer, nil
}

func (h *FileWriter) Write(p []byte) (int, error) {
	logRotate := NewLoggerRotate()
	logFile := filepath.Join(h.conf.LogPath, h.conf.LogName+"-"+logRotate.Current()+".log")

	if h.currLogFile != logFile {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.currLogFile = logFile
		// Close prev file
		if h.writer != nil {
			_ = h.writer.Sync()
			if err := h.writer.Close(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "close prev file error %s", err)
			}
		}

		// Delete old log file
		if h.conf.LogMaxFiles > 0 {
			oldFile := filepath.Join(h.conf.LogPath, h.conf.LogName+"-"+logRotate.Prev(h.conf.LogMaxFiles)+".log")
			_ = os.Remove(oldFile)
		}

		var err error
		h.writer, err = os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

		if err != nil {
			return -1, fmt.Errorf("can't open file: path = %v, err = %v", logFile, err)
		}
	}

	return h.writer.Write(p)
}
