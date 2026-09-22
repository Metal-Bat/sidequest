package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Metal-Bat/sidequest/internal/account"
	"github.com/Metal-Bat/sidequest/internal/database"
	"github.com/Metal-Bat/sidequest/internal/quest"
	"github.com/Metal-Bat/sidequest/internal/web"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("sidequest_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE")
		if err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for range 2 {
		if err = database.Migrate(ctx, pool, resources); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func TestDatabaseIsolationSelectionAndConcurrentCompletion(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	s := quest.Store{Pool: p}
	var alice, bob int64
	if err := p.QueryRow(ctx, `INSERT INTO users(username,password_hash)VALUES('alice','unused')RETURNING id`).Scan(&alice); err != nil {
		t.Fatal(err)
	}
	if err := p.QueryRow(ctx, `INSERT INTO users(username,password_hash)VALUES('bob','unused')RETURNING id`).Scan(&bob); err != nil {
		t.Fatal(err)
	}
	create := func(user int64, title string, minutes int) quest.Quest {
		q, err := s.Create(ctx, quest.Quest{UserID: user, Title: title, Difficulty: 5, EstimatedMinutes: minutes, RewardXP: 999999})
		if err != nil {
			t.Fatal(err)
		}
		if q.RewardXP != 170 {
			t.Fatal("trusted submitted XP")
		}
		return q
	}
	short := create(alice, "Short", 15)
	other := create(alice, "Other", 15)
	long := create(alice, "Long", 60)
	_ = create(bob, "Private", 1)
	q, err := s.Pick(ctx, alice, 15, short.ID)
	if err != nil || q.ID != other.ID {
		t.Fatalf("reroll: %+v %v", q, err)
	}
	q, err = s.Pick(ctx, alice, 0, 0)
	if err != nil || q.UserID != alice {
		t.Fatalf("any duration: %+v %v", q, err)
	}
	if _, err = s.Pick(ctx, alice, 1, 0); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("time filter: %v", err)
	}
	for _, action := range []string{"accept", "complete", "abandon", "delete"} {
		if _, err = s.Mutate(ctx, bob, short.ID, action); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("ownership %s: %v", action, err)
		}
	}
	if _, err = s.Mutate(ctx, alice, long.ID, "accept"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Mutate(ctx, alice, long.ID, "abandon"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Mutate(ctx, alice, long.ID, "complete"); !errors.Is(err, quest.ErrTransition) {
		t.Fatalf("abandoned completion: %v", err)
	}
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Mutate(ctx, alice, short.ID, "complete"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var xp int
	var completed bool
	if err = p.QueryRow(ctx, `SELECT xp FROM users WHERE id=$1`, alice).Scan(&xp); err != nil || xp != 170 {
		t.Fatalf("XP=%d err=%v", xp, err)
	}
	if err = p.QueryRow(ctx, `SELECT status='completed' AND completed_at IS NOT NULL FROM quests WHERE id=$1`, short.ID).Scan(&completed); err != nil || !completed {
		t.Fatal("completion not recorded", err)
	}
	if _, err = s.Mutate(ctx, alice, short.ID, "delete"); err != nil {
		t.Fatal(err)
	}
	if err = p.QueryRow(ctx, `SELECT xp FROM users WHERE id=$1`, alice).Scan(&xp); err != nil || xp != 170 {
		t.Fatal("deleting history changed XP", err)
	}
	qs, err := s.List(ctx, bob)
	if err != nil || len(qs) != 1 || qs[0].UserID != bob {
		t.Fatalf("list isolation: %+v %v", qs, err)
	}
}

type browser struct {
	t      *testing.T
	client *http.Client
	base   string
}

func newBrowser(t *testing.T, base string) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t, &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, base}
}

func (b *browser) request(method, path string, values url.Values, hx bool) (int, string, http.Header) {
	b.t.Helper()
	if values == nil {
		values = url.Values{}
	}
	if method == "POST" {
		u, _ := url.Parse(b.base)
		for _, c := range b.client.Jar.Cookies(u) {
			if c.Name == "sidequest-csrf" {
				values.Set("csrf", c.Value)
			}
		}
	}
	r, err := http.NewRequest(method, b.base+path, strings.NewReader(values.Encode()))
	if err != nil {
		b.t.Fatal(err)
	}
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if hx {
		r.Header.Set("HX-Request", "true")
	}
	resp, err := b.client.Do(r)
	if err != nil {
		b.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return resp.StatusCode, string(body), resp.Header
}

func TestHTTPJourney(t *testing.T) {
	p := testPool(t)
	h, err := web.New(p, resources, false)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	b := newBrowser(t, server.URL)
	check := func(method, path string, v url.Values, hx bool, want int) string {
		t.Helper()
		status, body, _ := b.request(method, path, v, hx)
		if status != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, status, want, body)
		}
		return body
	}
	check("GET", "/register", nil, false, 200)
	check("POST", "/register", url.Values{"username": {"hero"}, "password": {"a good secret passphrase"}}, false, 303)
	body := check("GET", "/", nil, false, 200)
	if !strings.Contains(body, "Level 1") || !strings.Contains(body, "list-active") {
		t.Fatal("missing home state")
	}
	for _, path := range []string{"/static/sidequest.css", "/static/vendor/htmx.min.js", "/static/fonts/cinzel.ttf", "/static/fonts/source-sans-3.ttf"} {
		check("GET", path, nil, false, 200)
	}
	check("GET", "/static/index.html", nil, false, 404)
	check("GET", "/static/README.md", nil, false, 404)
	check("GET", "/quests/new", nil, false, 200)
	body = check("POST", "/quests", url.Values{"title": {"keep me"}, "difficulty": {"8"}, "minutes": {"15"}}, true, 422)
	if !strings.Contains(body, `value="keep me"`) || strings.Contains(body, "<!doctype") {
		t.Fatal("validation did not retain form fragment")
	}
	check("POST", "/quests", url.Values{"title": {"<script>alert(1)</script>"}, "description": {"A real task"}, "difficulty": {"5"}, "minutes": {"15"}, "reward_xp": {"999999"}}, true, 200)
	var id int64
	if err = p.QueryRow(context.Background(), `SELECT id FROM quests LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	body = check("GET", "/quests/pick?minutes=15", nil, true, 200)
	if strings.Contains(body, "<script>alert") || !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("title not escaped")
	}
	check("GET", "/quests/pick?minutes=30", nil, false, 200)
	check("GET", "/quests/pick?minutes=-1", nil, true, 400)
	check("POST", fmt.Sprintf("/quests/%d/accept", id), nil, true, 200)
	body = check("POST", fmt.Sprintf("/quests/%d/complete", id), nil, true, 200)
	for _, want := range []string{"afterbegin:#list-completed", "delete:#quest-", "id=\"player-xp\"", "id=\"chosen-quest\"", "170 total XP"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	check("POST", fmt.Sprintf("/quests/%d/complete", id), nil, false, 303)
	check("POST", fmt.Sprintf("/quests/%d/abandon", id), nil, true, 409)
	check("POST", fmt.Sprintf("/quests/%d/delete", id), nil, true, 200)
	check("POST", "/quests", url.Values{"title": {"Second"}, "difficulty": {"2"}, "minutes": {"30"}}, false, 303)
	if err = p.QueryRow(context.Background(), `SELECT id FROM quests LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	check("POST", fmt.Sprintf("/quests/%d/abandon", id), nil, false, 303)
	check("POST", fmt.Sprintf("/quests/%d/delete", id), nil, false, 303)
	check("POST", "/quests/0/complete", nil, true, 400)
	check("POST", "/logout", nil, false, 303)
	status, _, headers := b.request("GET", "/", nil, true)
	if status != 200 || headers.Get("HX-Redirect") != "/login" {
		t.Fatal("expired auth did not redirect HTMX")
	}
	check("GET", "/login", nil, false, 200)
	check("POST", "/login", url.Values{"username": {"hero"}, "password": {"incorrect passphrase"}}, false, 422)
	check("POST", "/login", url.Values{"username": {"HERO"}, "password": {"a good secret passphrase"}}, false, 303)
	var hash string
	if err = p.QueryRow(context.Background(), `SELECT token_hash FROM sessions`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(server.URL)
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == "sidequest-session" && hash == c.Value {
			t.Fatal("raw session stored")
		}
	}
	if _, err = p.Exec(context.Background(), `UPDATE sessions SET expires_at=NOW()-INTERVAL '1 second'`); err != nil {
		t.Fatal(err)
	}
	_, _, headers = b.request("GET", "/", nil, true)
	if headers.Get("HX-Redirect") != "/login" {
		t.Fatal("expired session accepted")
	}
}

func TestAdminCreatesUsers(t *testing.T) {
	p := testPool(t)
	ctx := context.Background()
	if err := account.Create(ctx, p, "guildadmin", "a strong admin password", true); err != nil {
		t.Fatal(err)
	}
	if err := account.Create(ctx, p, "guildadmin", "another strong password", true); !errors.Is(err, account.ErrExists) {
		t.Fatal("duplicate admin must fail", err)
	}
	h, err := web.New(p, resources, false)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	admin := newBrowser(t, server.URL)
	admin.request("GET", "/login", nil, false)
	status, _, _ := admin.request("POST", "/login", url.Values{"username": {"guildadmin"}, "password": {"a strong admin password"}}, false)
	if status != 303 {
		t.Fatal("admin login", status)
	}
	status, body, _ := admin.request("GET", "/admin/users/new", nil, false)
	if status != 200 || !strings.Contains(body, "Create a guild member") {
		t.Fatal("admin form", status)
	}
	status, _, _ = admin.request("POST", "/admin/users", url.Values{"username": {"member"}, "password": {"a strong member password"}, "is_admin": {"true"}}, false)
	if status != 303 {
		t.Fatal("create member", status)
	}
	var isAdmin bool
	if err = p.QueryRow(ctx, `SELECT is_admin FROM users WHERE username='member'`).Scan(&isAdmin); err != nil || isAdmin {
		t.Fatal("member gained admin", err)
	}
	status, _, _ = admin.request("GET", "/admin/users/new", nil, false)
	if status != 200 {
		t.Fatal("admin session changed")
	}
	member := newBrowser(t, server.URL)
	member.request("GET", "/login", nil, false)
	status, _, _ = member.request("POST", "/login", url.Values{"username": {"member"}, "password": {"a strong member password"}}, false)
	if status != 303 {
		t.Fatal("created user cannot login")
	}
	for _, method := range []string{"GET", "POST"} {
		path := "/admin/users/new"
		if method == "POST" {
			path = "/admin/users"
		}
		status, _, _ = member.request(method, path, url.Values{"username": {"intruder"}, "password": {"another strong password"}}, false)
		if status != 403 {
			t.Fatal("member accessed admin route", method, status)
		}
	}
	if _, err = p.Exec(ctx, `UPDATE users SET is_admin=false WHERE username='guildadmin'`); err != nil {
		t.Fatal(err)
	}
	status, _, _ = admin.request("GET", "/admin/users/new", nil, false)
	if status != 403 {
		t.Fatal("stale admin permission")
	}
}
