package web

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (a *App) middleware(next http.Handler) http.Handler {
	return logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if a.secure {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "HX-Request")
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		csrf := ""
		if c, err := r.Cookie(a.cookieName("csrf")); err == nil && validToken(c.Value) {
			csrf = c.Value
		}
		if csrf == "" {
			csrf = randomToken()
			a.cookie(w, "csrf", csrf, 86400)
		}
		r = r.WithContext(context.WithValue(r.Context(), csrfKey, csrf))
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 16384)
			if err := r.ParseForm(); err != nil {
				a.fail(w, r, 400, "This form is too large or invalid.", nil)
				return
			}
			origin := r.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				scheme := "http"
				if a.secure || r.TLS != nil {
					scheme = "https"
				}
				if err != nil || u.Host != r.Host || u.Scheme != scheme {
					a.fail(w, r, 403, "Please reload the page before submitting.", nil)
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" || subtle.ConstantTimeCompare([]byte(csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
				a.fail(w, r, 403, "Please reload the page before submitting.", nil)
				return
			}
		}
		next.ServeHTTP(w, r)
	}))
}
