package logger

import (
	"github.com/zauremazhikovayandex/url/internal/logger/drivers"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"time"
)

var Log Interface

type Interface interface {
	Debug(msg *message.LogMessage)
	Info(msg *message.LogMessage)
	Warn(msg *message.LogMessage)
	Error(msg *message.LogMessage)
	Fatal(msg *message.LogMessage)
	Panic(msg *message.LogMessage)
}

func New(level string) Interface {
	Log = drivers.MakeStdoutLogger(level)
	Logging = &Writer{}
	return Log
}

var Logging LogWriter

type LogWriter interface {
	WriteToLog(timeStart time.Time, originalURL string, requestType string, responseCode int, responseBody string)
}
type Writer struct{}

func (l *Writer) WriteToLog(timeStart time.Time, originalURL string, requestType string,
	responseCode int, responseBody string) {
	timeEnd := time.Now()
	duration := timeEnd.Sub(timeStart)

	requestInfo := make(map[string]interface{})
	requestInfo["duration"] = duration
	requestInfo["uri"] = originalURL
	requestInfo["request_type"] = requestType
	requestInfo["response_code"] = responseCode
	requestInfo["response_body"] = responseBody

	Log.Info(&message.LogMessage{Message: "REQUEST INFO: %s",
		Extra: &requestInfo,
	})
}
