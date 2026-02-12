package gorm

import "fmt"

// Logger interface
type Logger interface {
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})

	Infof(format string, args ...interface{})
	Info(args ...interface{})

	Warnf(format string, args ...interface{})
	Warn(args ...interface{})

	Errorf(format string, args ...interface{})
	Error(args ...interface{})
}

type DefaultLogger struct {
}

func (d DefaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (d DefaultLogger) Debug(args ...interface{}) {
	fmt.Println(args...)
}

func (d DefaultLogger) Infof(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))

}

func (d DefaultLogger) Info(args ...interface{}) {
	fmt.Println(args...)
}

func (d DefaultLogger) Warnf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))

}

func (d DefaultLogger) Warn(args ...interface{}) {
	fmt.Println(args...)
}

func (d DefaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (d DefaultLogger) Error(args ...interface{}) {
	fmt.Println(args...)
}
