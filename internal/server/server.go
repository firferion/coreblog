package server

import (
	"bytes"
	"html/template"
	"net/http"
	"sync"

	"coreblog/internal/blog"
)

// Server представляет собой HTTP-сервер приложения.
type Server struct {
	store *blog.Store
	mux   *http.ServeMux
	cache map[string][]byte
	mu    sync.RWMutex
	tmpl  *template.Template
}

// NewServer создает новый экземпляр Server.
func NewServer(store *blog.Store) *Server {
	s := &Server{
		store: store,
		mux:   http.NewServeMux(),
		cache: make(map[string][]byte),
		tmpl:  template.Must(template.ParseGlob("templates/*.html")),
	}
	s.routes()
	return s
}

// Router возвращает обработчик маршрутов.
func (s *Server) Router() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	s.mux.HandleFunc("GET /{$}", s.handleIndex())
	s.mux.HandleFunc("GET /article/{slug}", s.handleArticle())
}

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		data, ok := s.cache["index"]
		if ok {
			s.mu.RUnlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		s.mu.RUnlock()

		articles, err := s.store.GetLatestArticles(r.Context(), 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&buf, "base.html", articles); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		htmlData := buf.Bytes()

		s.mu.Lock()
		s.cache["index"] = htmlData
		s.mu.Unlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlData)
	}
}

func (s *Server) handleArticle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		key := "article:" + slug

		s.mu.RLock()
		data, ok := s.cache[key]
		if ok {
			s.mu.RUnlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		s.mu.RUnlock()

		article, err := s.store.GetArticleBySlug(r.Context(), slug)
		if err != nil {
			http.Error(w, "Статья не найдена", http.StatusNotFound)
			return
		}

		var buf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&buf, "base.html", article); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		htmlData := buf.Bytes()

		s.mu.Lock()
		s.cache[key] = htmlData
		s.mu.Unlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlData)
	}
}
