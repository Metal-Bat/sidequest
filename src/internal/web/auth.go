package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func hashToken(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func validToken(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32
}

func (a *App) cookieName(kind string) string {
	if a.secure {
		return "__Host-sidequest-" + kind
	}
	return "sidequest-" + kind
}

func (a *App) cookie(w http.ResponseWriter, kind, value string, age int) {
	c := &http.Cookie{
		Name:     a.cookieName(kind),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   age,
	}
	if age < 0 {
		c.Expires = time.Unix(1, 0)
	} else {
		c.Expires = time.Now().Add(time.Duration(age) * time.Second)
	}
	http.SetCookie(w, c)
}

func (a *App) protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(a.cookieName("session"))
		if err != nil || !validToken(c.Value) {
			redirect(w, r, "/login")
			return
		}
		var u User
		err = a.pool.QueryRow(r.Context(), `SELECT u.id,u.username,u.xp,u.is_admin FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=$1 AND s.expires_at>NOW()`, hashToken(c.Value)).Scan(&u.ID, &u.Username, &u.XP, &u.IsAdmin)
		if errors.Is(err, pgx.ErrNoRows) {
			a.cookie(w, "session", "", -1)
			redirect(w, r, "/login")
			return
		}
		if err != nil {
			a.fail(w, r, 500, "The guild is temporarily unavailable.", err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func (a *App) authPage(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = strings.TrimPrefix(r.URL.Path, "/")
	a.render(w, 200, "layout", d)
}

func (a *App) allowAuth(r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, v := range a.attempts {
		if now.After(v.until) {
			delete(a.attempts, k)
		}
	}
	v := a.attempts[ip]
	if v.count == 0 {
		if len(a.attempts) >= 10000 {
			return false
		}
		v.until = now.Add(10 * time.Minute)
	}
	if v.count >= 20 {
		return false
	}
	v.count++
	a.attempts[ip] = v
	return true
}

func (a *App) authenticate(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = strings.TrimPrefix(r.URL.Path, "/")
	d.Username = strings.ToLower(strings.TrimSpace(r.PostForm.Get("username")))
	password := r.PostForm.Get("password")
	show := func(status int, msg string) {
		d.Error = msg
		a.render(w, status, "layout", d)
	}
	if !a.allowAuth(r) {
		w.Header().Set("Retry-After", "600")
		show(429, "Too many attempts. Please try again in ten minutes.")
		return
	}
	if !usernamePattern.MatchString(d.Username) || len(password) < 10 || len(password) > 72 {
		show(422, "Use a username of 3-32 letters, digits or underscores and a password of 10-72 bytes.")
		return
	}
	select {
	case a.hashing <- struct{}{}:
		defer func() { <-a.hashing }()
	default:
		show(429, "The guild desk is busy. Please try again shortly.")
		return
	}
	var id int64
	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		a.fail(w, r, 500, "The guild is temporarily unavailable.", err)
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if d.Page == "register" {
		var hash []byte
		hash, err = bcrypt.GenerateFromPassword([]byte(password), 12)
		if err == nil {
			err = tx.QueryRow(r.Context(), `INSERT INTO users(username,password_hash)VALUES($1,$2)RETURNING id`, d.Username, string(hash)).Scan(&id)
		}
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			show(422, "That username is already in the ledger. Choose another.")
			return
		}
	} else {
		var hash string
		err = tx.QueryRow(r.Context(), `SELECT id,password_hash FROM users WHERE username=$1`, d.Username).Scan(&id, &hash)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword(a.dummy, []byte(password))
			show(422, "Username or password is incorrect.")
			return
		}
		if err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			show(422, "Username or password is incorrect.")
			return
		}
	}
	if err != nil {
		a.fail(w, r, 500, "The guild could not record your sign-in.", err)
		return
	}
	token := randomToken()
	_, err = tx.Exec(r.Context(), `DELETE FROM sessions WHERE expires_at<=NOW()`)
	if err == nil {
		if old, e := r.Cookie(a.cookieName("session")); e == nil {
			_, err = tx.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, hashToken(old.Value))
		}
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO sessions(user_id,token_hash,expires_at)VALUES($1,$2,NOW()+INTERVAL '7 days')`, id, hashToken(token))
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		a.fail(w, r, 500, "The guild could not record your sign-in.", err)
		return
	}
	a.cookie(w, "session", token, 7*86400)
	a.cookie(w, "csrf", randomToken(), 86400)
	redirect(w, r, "/")
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(a.cookieName("session"))
	if _, err := a.pool.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, hashToken(c.Value)); err != nil {
		a.fail(w, r, 500, "Could not sign out. Please try again.", err)
		return
	}
	a.cookie(w, "session", "", -1)
	a.cookie(w, "csrf", "", -1)
	redirect(w, r, "/login")
}
