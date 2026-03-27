package blog

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store представляет собой слой доступа к данным для блога.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore создает и возвращает новый экземпляр Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
	}
}

// GetLatestArticles возвращает список последних статей с ограничением по количеству.
func (s *Store) GetLatestArticles(ctx context.Context, limit int) ([]Article, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, slug, content, created_at, updated_at 
		FROM articles 
		ORDER BY created_at DESC 
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var a Article
		var rawContent string
		err := rows.Scan(&a.ID, &a.Title, &a.Slug, &rawContent, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		a.Content = RenderMarkdown(rawContent)
		articles = append(articles, a)
	}

	if len(articles) == 0 {
		return articles, nil
	}

	// Собираем ID статей
	ids := make([]int, len(articles))
	articleMap := make(map[int]*Article)
	for i := range articles {
		ids[i] = articles[i].ID
		articleMap[articles[i].ID] = &articles[i]
	}

	// Загружаем теги для всех статей одним запросом
	tagRows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, at.article_id 
		FROM tags t 
		JOIN article_tags at ON t.id = at.tag_id 
		WHERE at.article_id = ANY($1)`, ids)
	if err != nil {
		return articles, nil // Не критично, если теги не загрузятся
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var t Tag
		var articleID int
		if err := tagRows.Scan(&t.ID, &t.Name, &t.Slug, &articleID); err == nil {
			if a, ok := articleMap[articleID]; ok {
				a.Tags = append(a.Tags, t)
			}
		}
	}

	return articles, nil
}

// GetArticleBySlug возвращает одну статью по её слагу.
func (s *Store) GetArticleBySlug(ctx context.Context, slug string) (Article, error) {
	var a Article
	var rawContent string
	err := s.pool.QueryRow(ctx, `
		SELECT id, title, slug, content, created_at, updated_at 
		FROM articles 
		WHERE slug = $1`, slug).Scan(&a.ID, &a.Title, &a.Slug, &rawContent, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Article{}, err
	}
	a.Content = RenderMarkdown(rawContent)

	// Загружаем теги для этой статьи
	tagRows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug 
		FROM tags t 
		JOIN article_tags at ON t.id = at.tag_id 
		WHERE at.article_id = $1`, a.ID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var t Tag
			if err := tagRows.Scan(&t.ID, &t.Name, &t.Slug); err == nil {
				a.Tags = append(a.Tags, t)
			}
		}
	}

	return a, nil
}
