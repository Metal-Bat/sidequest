package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestGuildNavigation(t *testing.T) {
	templates, err := template.New("").Funcs(templateFunctions()).ParseGlob("../../templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		data          Data
		wantAdminLink bool
	}{
		{name: "visitor", data: Data{Page: "login"}},
		{name: "member", data: Data{Page: "home", User: User{ID: 1, Username: "hero"}}},
		{name: "admin", data: Data{Page: "home", User: User{ID: 2, Username: "guildmaster", IsAdmin: true}}, wantAdminLink: true},
		{name: "recruitment", data: Data{Page: "admin-users", User: User{ID: 2, Username: "guildmaster", IsAdmin: true}}, wantAdminLink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var page bytes.Buffer
			if err := templates.ExecuteTemplate(&page, "layout", tc.data); err != nil {
				t.Fatal(err)
			}
			hasAdminLink := strings.Contains(page.String(), `href="/admin/users/new"`)
			if hasAdminLink != tc.wantAdminLink {
				t.Fatalf("admin navigation visible = %v, want %v", hasAdminLink, tc.wantAdminLink)
			}
			if strings.Contains(page.String(), "ZgotmplZ") {
				t.Fatal("template produced unsafe attribute placeholder")
			}
		})
	}
}
