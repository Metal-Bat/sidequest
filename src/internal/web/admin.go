package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Metal-Bat/sidequest/internal/account"
)

func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	if !d.User.IsAdmin {
		a.fail(w, r, http.StatusForbidden, "Only a guild administrator can create users.", nil)
		return
	}
	d.Page = "admin-users"
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("created") == "1" {
			d.Message = "User created. They can now sign in."
		}
		a.render(w, 200, "layout", d)
		return
	}
	d.Username = strings.TrimSpace(r.PostForm.Get("username"))
	select {
	case a.hashing <- struct{}{}:
		defer func() { <-a.hashing }()
	default:
		a.fail(w, r, 429, "The guild desk is busy. Please try again shortly.", nil)
		return
	}
	err := account.Create(r.Context(), a.pool, d.Username, r.PostForm.Get("password"), false)
	if errors.Is(err, account.ErrInvalid) || errors.Is(err, account.ErrExists) {
		d.Error = err.Error()
		a.render(w, 422, "layout", d)
		return
	}
	if err != nil {
		a.fail(w, r, 500, "The user could not be created.", err)
		return
	}
	redirect(w, r, "/admin/users/new?created=1")
}
