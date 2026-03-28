package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// generateRandomBase64url генерирует случайную строку заданной длины в байтах (base64url без паддинга)
func generateRandomBase64url(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) handleLoginVK() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 32 байта -> 43 символа в base64url (стандарт PKCE)
		verifier := generateRandomBase64url(32)

		// Вычисляем S256 challenge (SHA256 хеш от verifier)
		h := sha256.New()
		h.Write([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

		// State тоже делаем надежной длины (43 символа)
		state := generateRandomBase64url(32)

		http.SetCookie(w, &http.Cookie{Name: "vk_verifier", Value: verifier, Path: "/", HttpOnly: true, MaxAge: 300})
		http.SetCookie(w, &http.Cookie{Name: "vk_state", Value: state, Path: "/", HttpOnly: true, MaxAge: 300})

		// ИСПОЛЬЗУЕМ ВЕБ-ЭНДПОИНТ /authorize А НЕ СЕРВЕРНЫЙ /auth!
		authURL := fmt.Sprintf("https://id.vk.com/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s&code_challenge=%s&code_challenge_method=S256",
			s.vkClientID, url.QueryEscape(s.vkRedirectURI), state, challenge)

		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

func (s *Server) handleCallbackVK() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG OAuth: ClientID='%s', URI='%s'", s.vkClientID, s.vkRedirectURI)

		errReason := r.URL.Query().Get("error")
		if errReason != "" {
			http.Error(w, "VK Error: "+errReason, http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}

		state := r.URL.Query().Get("state")
		cookieState, err := r.Cookie("vk_state")
		if err != nil || cookieState.Value != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		cookieVerifier, err := r.Cookie("vk_verifier")
		if err != nil {
			http.Error(w, "Verifier cookie missing", http.StatusBadRequest)
			return
		}

		// Обмен кода на токен через новый эндпоинт VK ID с передачей code_verifier и device_id
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("client_id", s.vkClientID)
		data.Set("client_secret", s.vkClientSecret)
		data.Set("redirect_uri", s.vkRedirectURI)
		data.Set("code", code)
		data.Set("code_verifier", cookieVerifier.Value)
		
		deviceID := r.URL.Query().Get("device_id")
		if deviceID != "" {
			data.Set("device_id", deviceID)
		}
		
		stateStr := r.URL.Query().Get("state")
		if stateStr != "" {
			data.Set("state", stateStr)
		}

		req, _ := http.NewRequest("POST", "https://id.vk.com/oauth2/auth", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Token request failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var result struct {
			AccessToken string `json:"access_token"`
			UserID      int64  `json:"user_id"`
			Error       string `json:"error"`
			ErrorDesc   string `json:"error_description"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			http.Error(w, "JSON decode failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if result.Error != "" {
			http.Error(w, fmt.Sprintf("Token error: %s (%s)", result.Error, result.ErrorDesc), http.StatusUnauthorized)
			return
		}

		if fmt.Sprint(result.UserID) == s.adminVKID {
			http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "vk_logged_in", Path: "/", MaxAge: 86400, HttpOnly: true})
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		http.Error(w, "Доступ запрещен", http.StatusForbidden)
	}
}

func (s *Server) handleLoginYandex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authURL := fmt.Sprintf("https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s",
			s.yandexClientID, url.QueryEscape(s.yandexRedirectURI))
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

func (s *Server) handleCallbackYandex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errReason := r.URL.Query().Get("error")
		if errReason != "" {
			http.Error(w, "Yandex Error: "+errReason, http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}

		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("client_id", s.yandexClientID)
		data.Set("client_secret", s.yandexClientSecret)

		resp, err := http.PostForm("https://oauth.yandex.ru/token", data)
		if err != nil {
			http.Error(w, "Token request failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var tokenRes struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
			ErrorDesc   string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
			http.Error(w, "JSON decode failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if tokenRes.Error != "" {
			http.Error(w, fmt.Sprintf("Token error: %s (%s)", tokenRes.Error, tokenRes.ErrorDesc), http.StatusUnauthorized)
			return
		}

		reqInfo, err := http.NewRequest("GET", "https://login.yandex.ru/info", nil)
		if err != nil {
			http.Error(w, "Info request creation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		reqInfo.Header.Set("Authorization", "OAuth "+tokenRes.AccessToken)

		respInfo, err := http.DefaultClient.Do(reqInfo)
		if err != nil {
			http.Error(w, "Info request failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer respInfo.Body.Close()

		var infoRes struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(respInfo.Body).Decode(&infoRes); err != nil {
			http.Error(w, "Info decode failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("YANDEX LOGIN ID: %s", infoRes.ID)

		if infoRes.ID == s.adminYandexID {
			http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "vk_logged_in", Path: "/", MaxAge: 86400, HttpOnly: true})
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		http.Error(w, "Доступ запрещен", http.StatusForbidden)
	}
}
