package server

import "net/http"

func (s *Server) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_token")
		if err != nil || cookie.Value != "vk_logged_in" {
			http.Redirect(w, r, "/auth/login/vk", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
