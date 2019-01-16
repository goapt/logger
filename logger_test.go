package logger

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestInfo(t *testing.T) {
	Info("[CURL ERROR]", "123123", map[string]string{
		"Url":         "sdfsdfsdfsdf",
		"RequestTime": "2018-15-15",
		"ErrorInfo":   "sdfsdfs",
	})
}

func TestNewLogger(t *testing.T) {
	log := NewLogger(func(c *Config) {
		c.LogName = "test"
		c.LogMode = "file"
		c.LogPath = "/tmp/"
	})

	log.Error("error", map[string]string{
		"test": "123",
	})
	time.Sleep(5 * time.Second)
	log.Error("error", map[string]string{
		"test": "456",
	})
	log.Error("error", map[string]string{
		"test": "789",
	})

	log2 := NewLogger(func(c *Config) {
		c.LogName = "test2"
		c.LogMode = "file"
		c.LogPath = "/tmp/"
	})

	log2.Error("error", map[string]string{
		"test": "qwe",
	})
	time.Sleep(5 * time.Second)
	log2.Error("error", map[string]string{
		"test": "asd",
	})
	log2.Error("error", map[string]string{
		"test": "zxc",
	})
}

func TestTraceLog(t *testing.T) {
	log := NewLogger(func(c *Config) {
		c.LogName = "test"
		c.LogMode = "std"
		c.LogPath = "/tmp/"
		c.LogLevel = "trace"
	})

	log.Debug("sdfsdfsdf")
}

func TestNewCustom(t *testing.T) {
	log := NewLogger(func(c *Config) {
		c.LogMode = "custom"
		c.LogLevel = "trace"
		c.LogName = "test"
		c.LogDetail = true
		c.LogWriter = os.Stderr
	})

	log.WithFields(map[string]interface{}{
		"aaa":   123,
		"bbb": "sadf",
	}).Error("test custom logger", map[string]interface{}{
		"id":   1,
		"name": "test",
	})

	log.Error("test custom logger2", map[string]interface{}{
		"id":   1,
		"name": "test",
	})

	Setting(func(c *Config) {
		c.LogMode = "custom"
		c.LogLevel = "trace"
		c.LogName = "test"
		c.LogDetail = true
		c.LogWriter = os.Stderr
	})

	WithFields(map[string]interface{}{
		"aaa":   123,
		"bbb": "sadf",
	}).Error("test custom logger", map[string]interface{}{
		"id":   1,
		"name": "test",
	})

	Error("test custom logger2", map[string]interface{}{
		"id":   1,
		"name": "test",
	})
}
