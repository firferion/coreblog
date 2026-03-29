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

// AuthenticateUser проверяет наличие пользователя или создает его.
func (s *Store) AuthenticateUser(ctx context.Context, provider, providerID, username, avatarURL string, isAdmin bool) (*User, error) {
	var user User
	query := `SELECT u.id, u.username, u.avatar_url, u.role FROM users u JOIN user_identities ui ON u.id = ui.user_id WHERE ui.provider = $1 AND ui.provider_user_id = $2`
	err := s.pool.QueryRow(ctx, query, provider, providerID).Scan(&user.ID, &user.Username, &user.AvatarURL, &user.Role)

	if err != nil {
		if err.Error() == "no rows in result set" {
			role := "user"
			if isAdmin {
				role = "admin"
			}
			err = s.pool.QueryRow(ctx, `INSERT INTO users (username, avatar_url, role) VALUES ($1, $2, $3) RETURNING id`, username, avatarURL, role).Scan(&user.ID)
			if err != nil {
				return nil, err
			}

			_, err = s.pool.Exec(ctx, `INSERT INTO user_identities (user_id, provider, provider_user_id) VALUES ($1, $2, $3)`, user.ID, provider, providerID)
			if err != nil {
				return nil, err
			}

			user.Username = username
			user.AvatarURL = avatarURL
			user.Role = role
			return &user, nil
		}
		return nil, err
	}
	
	// Если пользователь уже есть, но в .env он теперь прописан как админ — повышаем права
	if isAdmin && user.Role != "admin" {
		_, updateErr := s.pool.Exec(ctx, `UPDATE users SET role = 'admin' WHERE id = $1`, user.ID)
		if updateErr == nil {
			user.Role = "admin"
		}
	}

	return &user, nil
}
