// Package drivers содержит реализации драйверов логирования.
package drivers

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"os"
)

// StdoutDriver — драйвер логирования на базе logrus, пишущий в stdout.
type StdoutDriver struct {
	log   *logrus.Logger
	level logrus.Level
}

// Debug пишет сообщение уровня debug.
func (l *StdoutDriver) Debug(msg *message.LogMessage) {
	l.write(logrus.DebugLevel, msg)
}

// Info пишет сообщение уровня info.
func (l *StdoutDriver) Info(msg *message.LogMessage) {
	l.write(logrus.InfoLevel, msg)
}

// Warn пишет сообщение уровня warn.
func (l *StdoutDriver) Warn(msg *message.LogMessage) {
	l.write(logrus.WarnLevel, msg)
}

// Error пишет сообщение уровня error.
func (l *StdoutDriver) Error(msg *message.LogMessage) {
	l.write(logrus.ErrorLevel, msg)
}

// Fatal пишет сообщение уровня fatal.
func (l *StdoutDriver) Fatal(msg *message.LogMessage) {
	l.write(logrus.FatalLevel, msg)
}

// Panic пишет сообщение уровня panic.
func (l *StdoutDriver) Panic(msg *message.LogMessage) {
	l.write(logrus.PanicLevel, msg)
}

func (l *StdoutDriver) write(level logrus.Level, msg *message.LogMessage) {
	j, err := json.Marshal(msg)

	if err != nil {
		return
	}

	l.log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: false})
	l.log.SetOutput(os.Stdout)
	l.log.Log(level, string(j))
}

// MakeStdoutLogger создает и настраивает StdoutDriver по уровню.
func MakeStdoutLogger(level string) *StdoutDriver {
	var lev logrus.Level

	switch level {
	case message.TraceLevel:
		lev = logrus.TraceLevel
	case message.DebugLevel:
		lev = logrus.DebugLevel
	case message.InfoLevel:
		lev = logrus.InfoLevel
	case message.WarnLevel:
		lev = logrus.WarnLevel
	case message.ErrorLevel:
		lev = logrus.ErrorLevel
	case message.FatalLevel:
		lev = logrus.FatalLevel
	case message.PanicLevel:
		lev = logrus.PanicLevel
	default:
		lev = logrus.WarnLevel
	}

	l := logrus.New()
	l.SetLevel(lev)

	return &StdoutDriver{log: l, level: lev}
}
