package blog

import (
	"html"
	"html/template"
	"regexp"
	"time"
)

// Article представляет собой модель статьи.
type Article struct {
	ID        int
	Title     string
	Slug      string
	Content   template.HTML
	Tags      []Tag
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Snippet возвращает содержимое статьи без HTML-тегов для превью.
func (a Article) Snippet() string {
	re := regexp.MustCompile(`<[^>]*>`)
	return html.UnescapeString(re.ReplaceAllString(string(a.Content), " "))
}

// Tag представляет собой модель тега.
type Tag struct {
	ID   int
	Name string
	Slug string
}
