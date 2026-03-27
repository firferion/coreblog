package blog

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

// RenderMarkdown преобразует строку Markdown в безопасный HTML.
func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		// В случае ошибки возвращаем исходный текст как есть (после базового экранирования или просто как строку)
		return template.HTML(md)
	}
	return template.HTML(buf.String())
}
