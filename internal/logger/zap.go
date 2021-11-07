package logger

import "go.uber.org/zap"

var logger *zap.SugaredLogger

type Logger struct {
	*zap.SugaredLogger
}

func GetLogger() Logger {
	if logger == nil {
		zaplog, _ := zap.NewDevelopment()
		logger = zaplog.Sugar()
	}

	return Logger{SugaredLogger: logger}
}
