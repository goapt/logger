package logger

import "time"

type ILoggerRotate interface {
	Current() string
	Prev(n int) string
}

type LoggerRotate struct{}

func (r *LoggerRotate) Current() string {
	return time.Now().Format("2006-01-02")
}

func (r *LoggerRotate) Prev(n int) string {
	return time.Now().AddDate(0, 0, -n).Format("2006-01-02")
}
