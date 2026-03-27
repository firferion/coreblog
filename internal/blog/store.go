package blog

import (
	"context"

	"github.com/jackc/pgx/v5"
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

	articles, err := pgx.CollectRows(rows, pgx.RowToStructByName[Article])
	if err != nil {
		return nil, err
	}

	for i := range articles {
		articles[i].Content = RenderMarkdown(string(articles[i].Content))
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
	return a, nil
}
