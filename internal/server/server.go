package server

import (
	"html/template"
	"net/http"
	"sync"

	"coreblog/internal/blog"
)

// Server представляет собой HTTP-сервер приложения.
type Server struct {
	store          *blog.Store
	mux            *http.ServeMux
	cache          map[string][]byte
	mu             sync.RWMutex
	vkClientID         string
	vkClientSecret     string
	vkRedirectURI      string
	adminVKID          string
	yandexClientID     string
	yandexClientSecret string
	yandexRedirectURI  string
	adminYandexID      string
	sessions           map[string]*blog.User
	sessionsMu         sync.RWMutex
}

// NewServer создает новый экземпляр Server.
func NewServer(store *blog.Store, vkClientID, vkClientSecret, vkRedirectURI, adminVKID, yandexClientID, yandexClientSecret, yandexRedirectURI, adminYandexID string) *Server {
	s := &Server{
		store:              store,
		mux:                http.NewServeMux(),
		cache:              make(map[string][]byte),
		vkClientID:         vkClientID,
		vkClientSecret:     vkClientSecret,
		vkRedirectURI:      vkRedirectURI,
		adminVKID:          adminVKID,
		yandexClientID:     yandexClientID,
		yandexClientSecret: yandexClientSecret,
		yandexRedirectURI:  yandexRedirectURI,
		adminYandexID:      adminYandexID,
		sessions:           make(map[string]*blog.User),
	}
	s.routes()
	return s
}

// Router возвращает обработчик маршрутов.
func (s *Server) Router() http.Handler {
	return s.mux
}

func (s *Server) getUser(r *http.Request) *blog.User {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	return s.sessions[cookie.Value]
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	s.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.png")
	})
	s.mux.HandleFunc("GET /{$}", s.handleIndex())
	s.mux.HandleFunc("GET /article/{slug}", s.handleArticle())

	// OAuth VK
	s.mux.HandleFunc("GET /auth/login/vk", s.handleLoginVK())
	s.mux.HandleFunc("GET /auth/callback/vk", s.handleCallbackVK())

	// OAuth Yandex
	s.mux.HandleFunc("GET /auth/login/yandex", s.handleLoginYandex())
	s.mux.HandleFunc("GET /auth/callback/yandex", s.handleCallbackYandex())

	// Admin
	s.mux.Handle("GET /admin", s.adminOnly(s.handleAdmin()))
	s.mux.Handle("GET /admin/editor", s.adminOnly(s.handleEditorGET()))

	// Privacy
	s.mux.HandleFunc("GET /privacy", s.handlePrivacy())

	// Logout
	s.mux.HandleFunc("GET /auth/logout", s.handleLogout())
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, data map[string]any) {
	// Автоматически добавляем пользователя для каждой страницы
	data["User"] = s.getUser(r)

	// Парсим шаблоны на лету (обеспечивает Live Reload HTML без рестарта сервера)
	t, err := template.ParseGlob("templates/*.html")
	if err != nil {
		http.Error(w, "Ошибка сборки шаблонов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, "Ошибка рендеринга: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		articles, err := s.store.GetLatestArticles(r.Context(), 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.render(w, r, map[string]any{
			"IsIndex":  true,
			"Articles": articles,
		})
	}
}

func (s *Server) handleArticle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		article, err := s.store.GetArticleBySlug(r.Context(), slug)
		if err != nil {
			http.Error(w, "Статья не найдена", http.StatusNotFound)
			return
		}

		s.render(w, r, map[string]any{
			"IsIndex": false,
			"Article": article,
		})
	}
}

func (s *Server) handleAdmin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Запрашиваем с запасом 100 последних статей
		articles, err := s.store.GetLatestArticles(r.Context(), 100)
		if err != nil {
			http.Error(w, "Ошибка загрузки статей", http.StatusInternalServerError)
			return
		}

		s.render(w, r, map[string]any{
			"IsAdmin":  true,
			"Articles": articles,
		})
	}
}

func (s *Server) handlePrivacy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, map[string]any{
			"IsPrivacy": true,
		})
	}
}

func (s *Server) handleEditorGET() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, map[string]any{
			"IsAdmin": true,
			"IsEditor": true,
		})
	}
}
