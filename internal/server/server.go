package server

import (
	"bytes"
	"fmt"
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
	tmpl           *template.Template
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
		tmpl:               template.Must(template.ParseGlob("templates/*.html")),
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

	// Logout
	s.mux.HandleFunc("GET /auth/logout", s.handleLogout())
}

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		articles, err := s.store.GetLatestArticles(r.Context(), 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmplData := map[string]any{
			"IsIndex":  true,
			"Articles": articles,
			"User":     s.getUser(r),
		}
		var buf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&buf, "base.html", tmplData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		htmlData := buf.Bytes()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlData)
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

		tmplData := map[string]any{
			"IsIndex": false,
			"Article": article,
			"User":    s.getUser(r),
		}
		var buf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&buf, "base.html", tmplData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		htmlData := buf.Bytes()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlData)
	}
}

func (s *Server) handleAdmin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h1>Добро пожаловать в Админку, Фирыч!</h1>")
	}
}
