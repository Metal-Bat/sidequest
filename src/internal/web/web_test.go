package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTokensAndCookies(t *testing.T) {
	token := randomToken()
	if !validToken(token) || token == randomToken() || hashToken(token) == token || validToken("bad") {
		t.Fatal("invalid token behavior")
	}
	a := App{secure: true}
	w := httptest.NewRecorder()
	a.cookie(w, "session", token, 60)
	c := w.Result().Cookies()[0]
	if c.Name != "__Host-sidequest-session" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" {
		t.Fatalf("insecure cookie: %+v", c)
	}
}

func TestCSRF(t *testing.T) {
	a := App{templates: errorTemplates(t)}
	token := randomToken()
	for _, tt := range []struct {
		name, token, origin, site string
		want                      int
	}{{"valid", token, "http://example.com", "same-origin", 204}, {"missing", "", "", "", 403}, {"wrong", randomToken(), "", "", 403}, {"foreign", token, "https://evil.example", "same-site", 403}, {"cross-site", token, "", "cross-site", 403}, {"scheme", token, "https://example.com", "", 403}} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://example.com/quests", strings.NewReader(url.Values{"csrf": {tt.token}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", tt.origin)
			r.Header.Set("Sec-Fetch-Site", tt.site)
			r.AddCookie(&http.Cookie{Name: "sidequest-csrf", Value: token})
			w := httptest.NewRecorder()
			a.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("got %d want %d", w.Code, tt.want)
			}
		})
	}
}

func TestUnauthenticatedHTMXRedirect(t *testing.T) {
	a := App{}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	a.protect(func(http.ResponseWriter, *http.Request) { t.Fatal("reached protected route") })(w, r)
	if w.Header().Get("HX-Redirect") != "/login" || w.Body.Len() != 0 {
		t.Fatal("expected empty HTMX redirect")
	}
}

func errorTemplates(t *testing.T) *template.Template {
	t.Helper()
	return template.Must(template.New("layout").Parse(`{{.Error}}`))
}
