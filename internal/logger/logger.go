package logger

import (
	"github.com/zauremazhikovayandex/url/internal/logger/drivers"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"net/http"
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

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStart := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		Logging.WriteToLog(timeStart, r.RequestURI, r.Method, lrw.statusCode, http.StatusText(lrw.statusCode))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
