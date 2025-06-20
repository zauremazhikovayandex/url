package drivers

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"os"
)

type StdoutDriver struct {
	log   *logrus.Logger
	level logrus.Level
}

func (l *StdoutDriver) Debug(msg *message.LogMessage) {
	l.write(logrus.DebugLevel, msg)
}

func (l *StdoutDriver) Info(msg *message.LogMessage) {
	l.write(logrus.InfoLevel, msg)
}

func (l *StdoutDriver) Warn(msg *message.LogMessage) {
	l.write(logrus.WarnLevel, msg)
}

func (l *StdoutDriver) Error(msg *message.LogMessage) {
	l.write(logrus.ErrorLevel, msg)
}

func (l *StdoutDriver) Fatal(msg *message.LogMessage) {
	l.write(logrus.FatalLevel, msg)
}

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

func MakeStdoutLogger(level string) *StdoutDriver {
	var lev logrus.Level

	switch level {
	case message.TraceLevel:
		lev = logrus.TraceLevel
		break
	case message.DebugLevel:
		lev = logrus.DebugLevel
		break
	case message.InfoLevel:
		lev = logrus.InfoLevel
		break
	case message.WarnLevel:
		lev = logrus.WarnLevel
		break
	case message.ErrorLevel:
		lev = logrus.ErrorLevel
		break
	case message.FatalLevel:
		lev = logrus.FatalLevel
		break
	case message.PanicLevel:
		lev = logrus.PanicLevel
		break
	default:
		lev = logrus.WarnLevel
	}

	l := logrus.New()
	l.SetLevel(lev)

	return &StdoutDriver{log: l, level: lev}
}
