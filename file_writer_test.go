package logger

import (
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	log := NewLogger(func(c *Config) {
		c.LogName = "test"
		c.LogMode = "file"
		c.LogPath = "/tmp/"
		c.LogLevel = "debug"
		c.LogSentryDSN = ""
	})

	log.Info("hahahahahah")
	log.Info("hahahahahah")
	log.Info("hahahahahah")
	log.Info("hahahahahah")
	log.WithFields(map[string]interface{}{"id": 1}).Info("hahahahahah")
}
