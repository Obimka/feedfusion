package handlers

import (
	"encoding/json"
	"html/template"
	"net/url"
	"strings"
)

// Tmpl is the global template variable used across all route handlers
var Tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"urlquery": func(s string) string {
		return url.QueryEscape(s)
	},
	"isReddit": func(s string) bool {
		return strings.Contains(strings.ToLower(s), "reddit.com")
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"toJSON": func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	},
}).ParseGlob("web/templates/*.html"))
