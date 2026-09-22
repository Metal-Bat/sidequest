package web

import (
	"bytes"
	"log/slog"
	"net/http"
)

func isHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func redirect(w http.ResponseWriter, r *http.Request, path string) {
	if isHX(r) {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, path, http.StatusSeeOther)
	}
}

func (a *App) data(r *http.Request) Data {
	d := Data{Difficulty: "1", Estimate: "15", Minutes: 15}
	d.User, _ = r.Context().Value(userKey).(User)
	d.CSRF, _ = r.Context().Value(csrfKey).(string)
	return d
}

func (a *App) render(w http.ResponseWriter, status int, name string, d Data) {
	var b bytes.Buffer
	if err := a.templates.ExecuteTemplate(&b, name, d); err != nil {
		slog.Error("template", "error", err)
		http.Error(w, "The ledger could not be opened.", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(b.Bytes())
}

func (a *App) fail(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	if err != nil {
		slog.Error("request failed", "path", r.URL.Path, "error", err)
	}
	d := a.data(r)
	d.Error = message
	d.Page = "error"
	name := "layout"
	if isHX(r) {
		name = "error"
		w.Header().Set("HX-Retarget", "#feedback")
		w.Header().Set("HX-Reswap", "innerHTML")
	}
	a.render(w, status, name, d)
}
