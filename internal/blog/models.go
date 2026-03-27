package blog

import (
	"html/template"
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

// Tag представляет собой модель тега.
type Tag struct {
	ID   int
	Name string
	Slug string
}
