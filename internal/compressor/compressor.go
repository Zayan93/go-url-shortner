package compressor

import (
	"compress/gzip"
	"go-url-shortner/internal/logger"
	"io"
	"net/http"
	"strings"
)

// создаем новый Writer, с добавлением функциональности компрессии
type compressWriter struct {
	w       http.ResponseWriter
	zw      *gzip.Writer
	started bool
}

// Header добавляет к новой писалке методы необходимые  для его использования вместо оригинального
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write непосредственно меняет писалку на писалку с компрессией
func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.started {
		ct := c.w.Header().Get("Content-Type")
		if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html") {
			c.w.Header().Set("Content-Encoding", "gzip")
			c.zw = gzip.NewWriter(c.w)
		} else {
			return c.w.Write(p) // просто пишем без сжатия
		}
		c.started = true
	}
	return c.zw.Write(p)
}

// WriteHeader добавляем установку статуса
func (c *compressWriter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// newCompressWriter основной интерфейс который служит заменой ResponseWriter
func NewCompressWriter(w http.ResponseWriter) *compressWriter {
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

func GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Декодируем входящий gzip, если есть
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := NewCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Bad Request - invalid gzip encoding", http.StatusBadRequest)
				return
			}
			r.Body = cr
			defer cr.Close()
			logger.Log.Info("Входящий gzip-декодирован")
		}

		// Проверяем, хочет ли клиент gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := NewCompressWriter(w)
			defer cw.Close()
			logger.Log.Info("Исходящий gzip будет использоваться")
			h.ServeHTTP(cw, r)
			return
		}

		// Без компрессии
		h.ServeHTTP(w, r)
	})
}
