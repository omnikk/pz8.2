package main

import (
	"log"
	"net/http"
	"os"

	httphandler "github.com/omnikk/pz8/services/auth/internal/http"
	"github.com/omnikk/pz8/services/auth/internal/service"
	"github.com/omnikk/pz8/shared/middleware"
)

func main() {
	httpPort := os.Getenv("AUTH_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	svc := service.New()
	handler := httphandler.New(svc)

	// auth-РЎРѓР ВµРЎР‚Р Р†Р С‘РЎРѓРЎС“ CSRF Р Р…Р Вµ Р Р…РЎС“Р В¶Р ВµР Р… (login РЎРѓР В°Р С Р Р†РЎвЂ№Р Т‘Р В°РЎвЂРЎвЂљ РЎвЂљР С•Р С”Р ВµР Р…), Р В° Р В·Р В°Р С–Р С•Р В»Р С•Р Р†Р С”Р С‘ Р В±Р ВµР В·Р С•Р С—Р В°РЎРѓР Р…Р С•РЎРѓРЎвЂљР С‘ РІР‚вЂќ Р Т‘Р В°
	var mux http.Handler = handler.Routes()
	mux = middleware.SecurityHeaders(mux)
	mux = middleware.Logging(mux)
	mux = middleware.RequestID(mux)

	log.Printf("Auth HTTP server starting on :%s", httpPort)
	if err := http.ListenAndServe(":"+httpPort, mux); err != nil {
		log.Fatalf("Auth HTTP server failed: %v", err)
	}
}
