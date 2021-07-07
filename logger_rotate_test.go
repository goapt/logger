package logger

import (
	"testing"
	"time"
)

func TestLoggerRotate_Current(t *testing.T) {
	currTime := "2021-10-11 12:23:23"
	ct, _ := time.Parse("2006-01-02 15:04:05", currTime)
	logrot := &LoggerRotate{
		currTime: ct,
	}

	now := logrot.Current()
	prev := logrot.Prev(1)

	if now != "2021-10-11" {
		t.Errorf("current time want %s ,but get %s", "2021-10-11", now)
	}

	if prev != "2021-10-10" {
		t.Errorf("current time want %s ,but get %s", "2021-10-10", now)
	}
}
