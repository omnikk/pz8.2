package httphandler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/omnikk/pz8/services/auth/internal/service"
)

type Handler struct {
	svc *service.AuthService
}

func New(svc *service.AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", h.login)
	mux.HandleFunc("/v1/auth/verify", h.verify)
	return mux
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	CSRFToken   string `json:"csrf_token"` // Р Т‘РЎС“Р В±Р В»Р С‘РЎР‚РЎС“Р ВµР С Р Т‘Р В»РЎРЏ JS-Р С”Р В»Р С‘Р ВµР Р…РЎвЂљР С•Р Р†, Р С”Р С•РЎвЂљР С•РЎР‚РЎвЂ№Р Вµ Р С—РЎР‚Р ВµР Т‘Р С—Р С•РЎвЂЎР С‘РЎвЂљР В°РЎР‹РЎвЂљ РЎвЂЎР С‘РЎвЂљР В°РЎвЂљРЎРЉ Р С‘Р В· РЎвЂљР ВµР В»Р В°
}

type verifyResponse struct {
	Valid   bool   `json:"valid"`
	Subject string `json:"subject,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	token, ok := h.svc.Login(req.Username, req.Password)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// CSRF-РЎвЂљР С•Р С”Р ВµР Р… РІР‚вЂќ РЎРѓР В»РЎС“РЎвЂЎР В°Р в„–Р Р…РЎвЂ№Р в„– UUID. Р В­РЎвЂљР С• Р С‘ Р ВµРЎРѓРЎвЂљРЎРЉ Р С•РЎРѓР Р…Р С•Р Р†Р Р…Р С•Р в„– РЎРѓР ВµР С”РЎР‚Р ВµРЎвЂљ Р Т‘Р В»РЎРЏ Double Submit Cookie.
	csrfToken := uuid.NewString()

	// session cookie: HttpOnly РІР‚вЂќ JS Р Р…Р Вµ Р СР С•Р В¶Р ВµРЎвЂљ Р С—РЎР‚Р С•РЎвЂЎР С‘РЎвЂљР В°РЎвЂљРЎРЉ (Р В·Р В°РЎвЂ°Р С‘РЎвЂљР В° Р С•РЎвЂљ Р С”РЎР‚Р В°Р В¶Р С‘ РЎвЂЎР ВµРЎР‚Р ВµР В· XSS)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	// csrf cookie: Р СњР вЂў HttpOnly РІР‚вЂќ Р С”Р В»Р С‘Р ВµР Р…РЎвЂљРЎРѓР С”Р С‘Р в„– JS Р Т‘Р С•Р В»Р В¶Р ВµР Р… Р ВµРЎвЂ РЎвЂЎР С‘РЎвЂљР В°РЎвЂљРЎРЉ Р С‘ РЎРѓР В»Р В°РЎвЂљРЎРЉ Р Р† X-CSRF-Token
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		CSRFToken:   csrfToken,
	})
}

// verify Р С—РЎР‚Р С‘Р Р…Р С‘Р СР В°Р ВµРЎвЂљ Р В»Р С‘Р В±Р С• Authorization: Bearer ..., Р В»Р С‘Р В±Р С• session cookie.
// Р СћР В°Р С” РЎРѓРЎвЂљР В°РЎР‚РЎвЂ№Р Вµ Р С”Р В»Р С‘Р ВµР Р…РЎвЂљРЎвЂ№ (curl РЎРѓ Bearer Р С‘Р В· Р СџР вЂ” 5) Р С—РЎР‚Р С•Р Т‘Р С•Р В»Р В¶Р В°РЎР‹РЎвЂљ РЎР‚Р В°Р В±Р С•РЎвЂљР В°РЎвЂљРЎРЉ,
// Р В° Р Р…Р С•Р Р†РЎвЂ№Р Вµ (РЎвЂЎР ВµРЎР‚Р ВµР В· login + cookies) РІР‚вЂќ РЎвЂљР С•Р В¶Р Вµ.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	var token string
	// 1) Р С—РЎР‚Р С•Р В±РЎС“Р ВµР С Р В·Р В°Р С–Р С•Р В»Р С•Р Р†Р С•Р С”
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimPrefix(h, "Bearer ")
	}
	// 2) Р ВµРЎРѓР В»Р С‘ Р Р† Р В·Р В°Р С–Р С•Р В»Р С•Р Р†Р С”Р Вµ Р Р…Р ВµРЎвЂљ РІР‚вЂќ Р С—РЎР‚Р С•Р В±РЎС“Р ВµР С session cookie
	if token == "" {
		if c, err := r.Cookie("session"); err == nil {
			token = c.Value
		}
	}

	if token == "" {
		writeJSON(w, http.StatusUnauthorized, verifyResponse{Valid: false, Error: "unauthorized"})
		return
	}

	subject, valid := h.svc.Verify(token)
	if !valid {
		writeJSON(w, http.StatusUnauthorized, verifyResponse{Valid: false, Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, verifyResponse{Valid: true, Subject: subject})
}
