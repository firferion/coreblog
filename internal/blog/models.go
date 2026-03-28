package blog

import (
	"html"
	"html/template"
	"regexp"
	"time"
)

// User представляет собой модель пользователя.
type User struct {
	ID        int
	Username  string
	AvatarURL string
	Role      string
}

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
	// 1. Вырезаем блоки <pre>...</pre> и <code>...</code> целиком вместе с контентом
	// Используем (?s) для работы точки с переносами строк. 
	// Go regexp не поддерживает обратные ссылки (\1), поэтому ищем раздельно.
	codeRegex := regexp.MustCompile(`(?s)<pre[^>]*>.*?</pre>|<code[^>]*>.*?</code>`)
	cleanContent := codeRegex.ReplaceAllString(string(a.Content), " ")

	// 2. Вырезаем остальные HTML-теги
	tagsRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagsRegex.ReplaceAllString(cleanContent, " ")

	// 3. Декодируем HTML-сущности (кавычки и прочее)
	return html.UnescapeString(text)
}

// Tag представляет собой модель тега.
type Tag struct {
	ID   int
	Name string
	Slug string
}
