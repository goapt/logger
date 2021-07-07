package logger

import "time"

type ILoggerRotate interface {
	Current() string
	Prev(n int) string
}

type LoggerRotate struct {
	currTime time.Time
}

func NewLoggerRotate() *LoggerRotate {
	return &LoggerRotate{currTime: time.Now()}
}

func (r *LoggerRotate) Current() string {
	return r.currTime.Format("2006-01-02")
}

func (r *LoggerRotate) Prev(n int) string {
	return r.currTime.AddDate(0, 0, -n).Format("2006-01-02")
}
