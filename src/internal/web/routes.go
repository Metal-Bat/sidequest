package web

import (
	"io/fs"
	"net/http"
	"strings"
)

func (a *App) routes(files fs.FS) (http.Handler, error) {
	static, err := fs.Sub(files, "static")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	staticHandler := http.StripPrefix("/static/", http.FileServerFS(static))
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		info, err := fs.Stat(static, name)
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		staticHandler.ServeHTTP(w, r)
	})
	for _, path := range []string{"/login", "/register"} {
		mux.HandleFunc("GET "+path, a.authPage)
		mux.HandleFunc("POST "+path, a.authenticate)
	}
	mux.HandleFunc("GET /admin/users/new", a.protect(a.adminUsers))
	mux.HandleFunc("POST /admin/users", a.protect(a.adminUsers))
	mux.HandleFunc("POST /logout", a.protect(a.logout))
	mux.HandleFunc("GET /{$}", a.protect(a.home))
	mux.HandleFunc("GET /quests/new", a.protect(a.newQuest))
	mux.HandleFunc("POST /quests", a.protect(a.create))
	mux.HandleFunc("GET /quests/pick", a.protect(a.pick))
	for _, action := range []string{"accept", "complete", "abandon", "delete"} {
		mux.HandleFunc("POST /quests/{id}/"+action, a.protect(a.mutate))
	}
	return a.middleware(mux), nil
}
