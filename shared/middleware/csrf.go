package middleware

import (
	"encoding/json"
	"net/http"
)

func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// safe methods РІР‚вЂќ Р С—РЎР‚Р С•Р С—РЎС“РЎРѓР С”Р В°Р ВµР С Р В±Р ВµР В· Р С—РЎР‚Р С•Р Р†Р ВµРЎР‚Р С”Р С‘
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("csrf_token")
		header := r.Header.Get("X-CSRF-Token")

		// Р Р†РЎРѓР Вµ РЎвЂљРЎР‚Р С‘ РЎС“РЎРѓР В»Р С•Р Р†Р С‘РЎРЏ Р Т‘Р С•Р В»Р В¶Р Р…РЎвЂ№ Р Р†РЎвЂ№Р С—Р С•Р В»Р Р…РЎРЏРЎвЂљРЎРЉРЎРѓРЎРЏ: cookie Р ВµРЎРѓРЎвЂљРЎРЉ, header Р ВµРЎРѓРЎвЂљРЎРЉ, Р С•Р Р…Р С‘ РЎР‚Р В°Р Р†Р Р…РЎвЂ№.
		if err != nil || cookie.Value == "" || header == "" || cookie.Value != header {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "CSRF token invalid"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
