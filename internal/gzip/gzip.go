// Package gzip содержит обертки для gzip-сжатия/распаковки HTTP-тел.
package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// NewCompressWriter возвращает http.ResponseWriter, сжимающий ответ gzip'ом.
func NewCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header возвращает заголовки исходного http.ResponseWriter.
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает несжатые данные p, сжимая их во внутренний gzip.Writer.
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader устанавливает статус ответа и проставляет Content-Encoding: gzip
// для успешных (statusCode < 300) ответов до записи тела.
func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает внутренний gzip.Writer, досылая все буферизированные данные.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// compressReader реализует io.ReadCloser и прозрачно декомпрессирует входящее
// gzip-тело запроса для дальнейшего чтения обработчиком.
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader создает обертку для чтения gzip-сжатого тела запроса.
// На вход ожидается io.ReadCloser (например, r.Body).
func NewCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read читает разжатые данные в p из внутреннего gzip.Reader.
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close закрывает исходный r и внутренний gzip.Reader.
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
