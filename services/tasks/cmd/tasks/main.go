package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/omnikk/pz8/services/tasks/internal/client/authclient"
	httphandler "github.com/omnikk/pz8/services/tasks/internal/http"
	"github.com/omnikk/pz8/services/tasks/internal/repository"
	"github.com/omnikk/pz8/services/tasks/internal/service"
	"github.com/omnikk/pz8/shared/middleware"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	port := env("TASKS_PORT", "8082")

	// DSN Р В РўвЂР В Р’В»Р РЋР РЏ PostgreSQL Р РЋР С“Р В РЎвЂўР В Р’В±Р В РЎвЂР РЋР вЂљР В Р’В°Р В Р’ВµР В РЎВ Р В РЎвЂР В Р’В· Р В РЎвЂ”Р В Р’ВµР РЋР вЂљР В Р’ВµР В РЎВР В Р’ВµР В Р вЂ¦Р В Р вЂ¦Р РЋРІР‚в„–Р РЋРІР‚В¦ Р В РЎвЂўР В РЎвЂќР РЋР вЂљР РЋРЎвЂњР В Р’В¶Р В Р’ВµР В Р вЂ¦Р В РЎвЂР РЋР РЏ.
	// Р В РІР‚в„ў docker-compose Р В РЎвЂўР В Р вЂ¦Р В РЎвЂ Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР В РЎвЂќР В РЎвЂР В Р вЂ¦Р РЋРЎвЂњР РЋРІР‚С™Р РЋРІР‚в„–, Р В Р’В»Р В РЎвЂўР В РЎвЂќР В Р’В°Р В Р’В»Р РЋР Р‰Р В Р вЂ¦Р В РЎвЂў Р В РЎВР В РЎвЂўР В Р’В¶Р В Р вЂ¦Р В РЎвЂў Р В РЎвЂ”Р В РЎвЂўР РЋР С“Р РЋРІР‚С™Р В Р’В°Р В Р вЂ Р В РЎвЂР РЋРІР‚С™Р РЋР Р‰ .env Р В РЎвЂР В Р’В»Р В РЎвЂ Р РЋР РЉР В РЎвЂќР РЋР С“Р В РЎвЂ”Р В РЎвЂўР РЋР вЂљР РЋРІР‚С™Р В РЎвЂР РЋР вЂљР В РЎвЂўР В Р вЂ Р В Р’В°Р РЋРІР‚С™Р РЋР Р‰ Р В Р вЂ Р РЋР вЂљР РЋРЎвЂњР РЋРІР‚РЋР В Р вЂ¦Р РЋРЎвЂњР РЋР вЂ№.
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		env("DB_HOST", "localhost"),
		env("DB_PORT", "5432"),
		env("DB_USER", "tasks_user"),
		env("DB_PASSWORD", "tasks_pass"),
		env("DB_NAME", "tasks_db"),
		env("DB_SSLMODE", "disable"),
	)

	repo, err := repository.NewPostgres(dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer repo.Close()
	log.Printf("connected to postgres: %s@%s/%s",
		env("DB_USER", "tasks_user"),
		env("DB_HOST", "localhost"),
		env("DB_NAME", "tasks_db"))

	authURL := env("AUTH_BASE_URL", "http://localhost:8081")
	auth := authclient.New(authURL)

	svc := service.New(repo)
	handler := httphandler.NewWithAuth(svc, auth)

	// Р СџР С•РЎР‚РЎРЏР Т‘Р С•Р С” middleware Р Р†Р В°Р В¶Р ВµР Р… (РЎвЂЎР С‘РЎвЂљР В°РЎвЂљРЎРЉ РЎРѓР Р…Р С‘Р В·РЎС“ Р Р†Р Р†Р ВµРЎР‚РЎвЂ¦ РІР‚вЂќ Р Р…Р В°РЎР‚РЎС“Р В¶Р Р…РЎвЂ№Р в„– Р С—РЎР‚Р С‘Р СР ВµР Р…РЎРЏР ВµРЎвЂљРЎРѓРЎРЏ Р С—Р С•РЎРѓР В»Р ВµР Т‘Р Р…Р С‘Р С):
	// 1) RequestID  РІР‚вЂќ Р С—РЎР‚Р С‘РЎРѓР Р†Р В°Р С‘Р Р†Р В°Р ВµРЎвЂљ X-Request-ID, Р Т‘Р С•Р В»Р В¶Р ВµР Р… Р В±РЎвЂ№РЎвЂљРЎРЉ РЎРѓР В°Р СРЎвЂ№Р С Р Р†Р Р…Р ВµРЎв‚¬Р Р…Р С‘Р С Р Т‘Р В»РЎРЏ Р Р†РЎРѓР ВµРЎвЂ¦ Р В»Р С•Р С–Р С•Р Р†
	// 2) Logging    РІР‚вЂќ Р С—Р С‘РЎв‚¬Р ВµРЎвЂљ access-log РЎРѓ request-id
	// 3) Security   РІР‚вЂќ Р Т‘Р С•Р В±Р В°Р Р†Р В»РЎРЏР ВµРЎвЂљ CSP/X-Frame-Options Р С”Р С• Р вЂ™Р РЋР вЂўР Сљ Р С•РЎвЂљР Р†Р ВµРЎвЂљР В°Р С (Р Т‘Р В°Р В¶Р Вµ Р С” 401/403)
	// 4) CSRF       РІР‚вЂќ Р С—РЎР‚Р С•Р Р†Р ВµРЎР‚РЎРЏР ВµРЎвЂљ РЎвЂљР С•Р С”Р ВµР Р… Р Т‘Р В»РЎРЏ POST/PATCH/DELETE
	var mux http.Handler = handler.Routes()
	mux = middleware.CSRFProtection(mux)
	mux = middleware.SecurityHeaders(mux)
	mux = middleware.Logging(mux)
	mux = middleware.RequestID(mux)

	log.Printf("Tasks service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Tasks service failed: %v", err)
	}
}
