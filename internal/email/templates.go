package email

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var emailTemplates embed.FS

func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(emailTemplates, "templates/*.html")
}
