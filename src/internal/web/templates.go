package web

import (
	"html/template"
	"strings"

	"github.com/Metal-Bat/sidequest/internal/quest"
)

func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"rowData": func(d Data, q quest.Quest) Data {
			d.Changed = q
			return d
		},
		"difficulties": func() []string { return []string{"1", "2", "3", "4", "5"} },
		"level": func(x int) int {
			level, _ := quest.Level(x)
			return level
		},
		"progress": func(x int) int {
			_, progress := quest.Level(x)
			return progress
		},
		"percent":  func(x int) int { return (x % 200) / 2 },
		"stars":    func(x int) string { return strings.Repeat("★", x) + strings.Repeat("☆", 5-x) },
		"statuses": func() []string { return []string{"available", "active", "completed", "abandoned"} },
		"label": func(s string) string {
			return map[string]string{"available": "Unclaimed", "active": "Accepted", "completed": "Completed", "abandoned": "Abandoned"}[s]
		},
	}
}
