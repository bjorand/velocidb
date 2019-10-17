package log

import (
	"github.com/sirupsen/logrus"
)

type Logger struct {
	l     *logrus.Logger
	e     *logrus.Entry
	src   string
	srcID string
}

// type Fields map[string]interface{}
type Fields logrus.Fields

func (l *Logger) commonFields() logrus.Fields {
	return logrus.Fields{
		"src":    l.src,
		"src_id": l.srcID,
	}
}

// func (l *Logger) Debug(msg ...interface{}) {
// 	l.l.WithFields(l.commonFields()).Debugln(msg...)
// }
func (l *Logger) Debug(fields map[string]interface{}, msg ...interface{}) {

	l.e.WithFields(fields).Debugln(msg...)
}

func (l *Logger) Info(fields map[string]interface{}, msg ...interface{}) {
	l.e.Infoln(msg...)
}

func (l *Logger) Trace(fields map[string]interface{}, msg ...interface{}) {
	l.e.Traceln(msg...)
}

func (l *Logger) NewLogger(fields map[string]interface{}) *Logger {
	return &Logger{
		l: l.l,
		e: l.e.WithFields(fields),
	}
}

func NewLogger(fields map[string]interface{}) *Logger {
	l := logrus.New()
	// TODO: set loglevel from environment
	l.SetLevel(logrus.DebugLevel)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return &Logger{
		l: l,
		e: l.WithFields(fields),
	}
}
