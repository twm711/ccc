package middleware

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/divord97/ccc/pkg/redact"
)

type redactWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
	isJSON bool
}

func (w *redactWriter) WriteHeader(code int) {
	w.status = code
	ct := w.Header().Get("Content-Type")
	w.isJSON = strings.Contains(ct, "application/json")
	if !w.isJSON {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *redactWriter) Write(b []byte) (int, error) {
	if w.isJSON {
		return w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// PIIRedact masks PII (phone, ID card) in JSON API responses.
func PIIRedact() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &redactWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			if rw.isJSON && rw.buf.Len() > 0 {
				masked := redact.Mask(rw.buf.String())
				w.WriteHeader(rw.status)
				w.Write([]byte(masked))
			}
		})
	}
}
