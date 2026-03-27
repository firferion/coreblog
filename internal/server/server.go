package server

import (
	"net/http"

	"coreblog/internal/blog"
)

// Server представляет собой HTTP-сервер приложения.
type Server struct {
	store *blog.Store
	mux   *http.ServeMux
}

// NewServer создает новый экземпляр Server.
func NewServer(store *blog.Store) *Server {
	s := &Server{
		store: store,
		mux:   http.NewServeMux(),
	}
	return s
}

// Router возвращает обработчик маршрутов.
func (s *Server) Router() http.Handler {
	return s.mux
}
