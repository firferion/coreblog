package blog

import (
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
