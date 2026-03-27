package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (s *Server) handleLoginVK() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authURL := fmt.Sprintf("https://oauth.vk.com/authorize?client_id=%s&redirect_uri=%s&display=page&response_type=code&v=5.131",
			s.vkClientID, url.QueryEscape(s.vkRedirectURI))
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

func (s *Server) handleCallbackVK() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}

		tokenURL := fmt.Sprintf("https://oauth.vk.com/access_token?client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
			s.vkClientID, s.vkClientSecret, url.QueryEscape(s.vkRedirectURI), code)

		resp, err := http.Get(tokenURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var result struct {
			AccessToken string `json:"access_token"`
			UserID      int64  `json:"user_id"`
			Error       string `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if result.Error != "" {
			http.Error(w, result.Error, http.StatusUnauthorized)
			return
		}

		if fmt.Sprint(result.UserID) == s.adminVKID {
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_token",
				Value:    "vk_logged_in",
				Path:     "/",
				MaxAge:   86400,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		http.Error(w, "Доступ запрещен", http.StatusForbidden)
	}
}
