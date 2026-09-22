package web

import (
	"log/slog"
	"net/http"
	"time"
)

type recorder struct {
	http.ResponseWriter
	status int
}

func (w *recorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *recorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &recorder{ResponseWriter: w}
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panic", "method", r.Method, "path", r.URL.Path, "error", v)
				if rw.status == 0 {
					http.Error(rw, "The ledger could not be opened.", http.StatusInternalServerError)
				}
			}
			if rw.status == 0 {
				rw.status = http.StatusOK
			}
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", time.Since(start))
		}()
		next.ServeHTTP(rw, r)
	})
}
