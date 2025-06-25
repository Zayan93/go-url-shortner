package gzip

import (
	"compress/gzip"
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

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка тела запроса, если клиент отправил его с Content-Encoding: gzip
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gr.Close()
			r.Body = gr
		}

		// Проверка: поддерживает ли клиент gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Подготовка gzip.Writer и gzipResponseWriter
		gz := gzip.NewWriter(w)
		defer gz.Close()

		grw := &compressWriter{
			w:  w,
			zw: gz,
		}

		next.ServeHTTP(grw, r)
	})
}
