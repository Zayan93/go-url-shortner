package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// создаем новый Writer, с добавлением функциональности компрессии
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// Header добавляет к новой писалке методы необходимые  для его использования вместо оригинального
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write непосредственно меняет писалку на писалку с компрессией
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader добавляем установку статуса
func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// newCompressWriter основной интерфейс который служит заменой ResponseWriter
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Декомпрессия запроса, если клиент прислал gzip
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "text/html") {
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				cr, err := newCompressReader(r.Body)
				if err != nil {
					http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
					return
				}
				defer cr.Close()
				r.Body = cr
			}
		}

		// Проверка, поддерживает ли клиент gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			w = cw // заменяем writer
			defer cw.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	})

}
