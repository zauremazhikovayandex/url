// Package logger предоставляет интерфейсы и реализации логирования приложения.
package logger

import (
	"github.com/zauremazhikovayandex/url/internal/logger/drivers"
	"github.com/zauremazhikovayandex/url/internal/logger/message"
	"net/http"
	"time"
)

// Log — активный драйвер логирования, реализующий Interface.
// Настраивается функцией New и используется всеми вызовами логирования.
var Log Interface

// Interface описывает минимально необходимый набор методов логгера.
// Конкретная реализация (stdout, файл и т.п.) должна поддерживать эти уровни.
type Interface interface {
	Debug(msg *message.LogMessage)
	Info(msg *message.LogMessage)
	Warn(msg *message.LogMessage)
	Error(msg *message.LogMessage)
	Fatal(msg *message.LogMessage)
	Panic(msg *message.LogMessage)
}

// New инициализирует глобальный логгер (Log) с заданным уровнем,
// а также выставляет Logging — access-логгер запросов.
// Возвращает активный драйвер логирования.
func New(level string) Interface {
	Log = drivers.MakeStdoutLogger(level)
	Logging = &Writer{}
	return Log
}

// Logging — access-логгер для записи агрегированной информации о HTTP-запросах.
// Должен быть инициализирован (например, через New) перед использованием.
var Logging LogWriter

// LogWriter описывает интерфейс access-логгера, который пишет сводку по запросу/ответу.
type LogWriter interface {
	WriteToLog(timeStart time.Time, originalURL string, requestType string, responseCode int, responseBody string)
}

// Writer — стандартная реализация access-логгера, использующая глобальный Log.
type Writer struct{}

// WriteToLog записывает сводку по HTTP-запросу/ответу: длительность, URI, метод,
// код ответа и краткое описание/тело ответа.
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

// RequestLogger — middleware, логирующий каждый HTTP-запрос/ответ с помощью Logging.
// Он перехватывает статус ответа и передает сводку в WriteToLog.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStart := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		Logging.WriteToLog(timeStart, r.RequestURI, r.Method, lrw.statusCode, http.StatusText(lrw.statusCode))
	})
}

// loggingResponseWriter — обертка над http.ResponseWriter, запоминающая статус ответа.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader сохраняет HTTP-статус в обертке и проксирует вызов исходному ResponseWriter.
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
