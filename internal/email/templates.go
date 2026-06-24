package email

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var emailTemplates embed.FS

// LoadTemplates parses all templates in the embedded filesystem.
func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(emailTemplates, "templates/*.html")
}
