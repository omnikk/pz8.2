package httphandler

import (
	"context"
	"encoding/json"
	"errors" // <-- РЎРѓРЎР‹Р Т‘Р В°
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/omnikk/pz8/services/tasks/internal/client/authclient"
	"github.com/omnikk/pz8/services/tasks/internal/service"
	"github.com/omnikk/pz8/shared/middleware"
)

type AuthVerifier interface {
	Verify(ctx context.Context, token, sessionCookie, requestID string) (string, error)
}

type Handler struct {
	svc  *service.TaskService
	auth AuthVerifier
}

func NewWithAuth(svc *service.TaskService, auth AuthVerifier) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tasks", h.tasksCollection)
	mux.HandleFunc("/v1/tasks/search", h.searchTasks)                     // Р В Р’В±Р В Р’ВµР В Р’В·Р В РЎвЂўР В РЎвЂ”Р В Р’В°Р РЋР С“Р В Р вЂ¦Р РЋРІР‚в„–Р В РІвЂћвЂ“ Р В РЎвЂ”Р В РЎвЂўР В РЎвЂР РЋР С“Р В РЎвЂќ
	mux.HandleFunc("/v1/tasks/searchvulnerable", h.searchTasksVulnerable) // Р РЋРЎвЂњР РЋР РЏР В Р’В·Р В Р вЂ Р В РЎвЂР В РЎВР РЋРІР‚в„–Р В РІвЂћвЂ“, Р В РўвЂР В Р’В»Р РЋР РЏ Р В РўвЂР В Р’ВµР В РЎВР В РЎвЂўР В Р вЂ¦Р РЋР С“Р РЋРІР‚С™Р РЋР вЂљР В Р’В°Р РЋРІР‚В Р В РЎвЂР В РЎвЂ
	mux.HandleFunc("/v1/tasks/", h.taskItem)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// internalError Р Р†Р вЂљРІР‚Сњ Р В Р’ВµР В РўвЂР В РЎвЂР В Р вЂ¦Р В Р’В°Р РЋР РЏ Р РЋРІР‚С™Р В РЎвЂўР РЋРІР‚РЋР В РЎвЂќР В Р’В° Р В Р вЂ Р В РЎвЂўР В Р’В·Р В Р вЂ Р РЋР вЂљР В Р’В°Р РЋРІР‚С™Р В Р’В° 500-Р В РЎвЂўР РЋРІвЂљВ¬Р В РЎвЂР В Р’В±Р В РЎвЂўР В РЎвЂќ.
// Р В РІР‚СњР В Р’ВµР РЋРІР‚С™Р В Р’В°Р В Р’В»Р РЋР Р‰ Р В РЎвЂ”Р В Р’В°Р В РўвЂР В Р’В°Р В Р’ВµР РЋРІР‚С™ Р В Р вЂ  Р В Р’В»Р В РЎвЂўР В РЎвЂ“, Р В РЎвЂќР В Р’В»Р В РЎвЂР В Р’ВµР В Р вЂ¦Р РЋРІР‚С™Р РЋРЎвЂњ Р Р†Р вЂљРІР‚Сњ Р В РЎвЂўР В Р’В±Р РЋРІР‚В°Р В Р’ВµР В Р’Вµ Р РЋР С“Р В РЎвЂўР В РЎвЂўР В Р’В±Р РЋРІР‚В°Р В Р’ВµР В Р вЂ¦Р В РЎвЂР В Р’Вµ (Р В Р вЂ¦Р В Р’Вµ Р РЋР С“Р В Р вЂ Р В Р’ВµР РЋРІР‚С™Р В РЎвЂР В РЎВ Р В Р вЂ Р В Р вЂ¦Р РЋРЎвЂњР РЋРІР‚С™Р РЋР вЂљР РЋР Р‰ Р РЋР С“Р В РЎвЂР РЋР С“Р РЋРІР‚С™Р В Р’ВµР В РЎВР РЋРІР‚в„–).
func internalError(w http.ResponseWriter, rid string, err error, where string) {
	log.Printf("[%s] %s: %v", rid, where, err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func (h *Handler) checkAuth(ctx context.Context, r *http.Request) (string, int) {
	rid := r.Header.Get(middleware.RequestIDHeader)

	// 1) Bearer-РЎвЂљР С•Р С”Р ВµР Р… (Р С”Р В°Р С” Р Р† Р СџР вЂ” 5)
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		// Р С—РЎР‚Р ВµРЎвЂћР С‘Р С”РЎРѓР В° Р Р…Р Вµ Р В±РЎвЂ№Р В»Р С• РІР‚вЂќ Р В·Р Р…Р В°РЎвЂЎР С‘РЎвЂљ РЎРЊРЎвЂљР С• Р Р…Р Вµ Bearer
		token = ""
	}

	// 2) session cookie (Р СџР вЂ” 6)
	var sessionCookie string
	if c, err := r.Cookie("session"); err == nil {
		sessionCookie = c.Value
	}

	if token == "" && sessionCookie == "" {
		return rid, http.StatusUnauthorized
	}

	subject, err := h.auth.Verify(ctx, token, sessionCookie, rid)
	if err != nil {
		if errors.Is(err, authclient.ErrUnauthorized) {
			return rid, http.StatusUnauthorized
		}
		log.Printf("[%s] auth service unavailable: %v", rid, err)
		return rid, http.StatusServiceUnavailable
	}
	log.Printf("[%s] auth ok, subject=%s", rid, subject)
	return rid, 0
}

func (h *Handler) tasksCollection(w http.ResponseWriter, r *http.Request) {
	rid, errStatus := h.checkAuth(r.Context(), r)
	if errStatus != 0 {
		writeJSON(w, errStatus, map[string]string{"error": http.StatusText(errStatus)})
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.createTask(w, r, rid)
	case http.MethodGet:
		h.listTasks(w, r, rid)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) taskItem(w http.ResponseWriter, r *http.Request) {
	rid, errStatus := h.checkAuth(r.Context(), r)
	if errStatus != 0 {
		writeJSON(w, errStatus, map[string]string{"error": http.StatusText(errStatus)})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing task id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getTask(w, r, rid, id)
	case http.MethodPatch:
		h.updateTask(w, r, rid, id)
	case http.MethodDelete:
		h.deleteTask(w, r, rid, id)
	default:
		http.NotFound(w, r)
	}
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request, rid string) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	// description Р С—РЎР‚Р С•РЎвЂ¦Р С•Р Т‘Р С‘РЎвЂљ РЎвЂЎР ВµРЎР‚Р ВµР В· html.EscapeString: <script> РІвЂ вЂ™ &lt;script&gt;
	// Р СџРЎР‚Р С‘ Р Р†РЎвЂ№Р Р†Р С•Р Т‘Р Вµ Р С•Р В±РЎР‚Р В°РЎвЂљР Р…Р С• Р Р† Р В»РЎР‹Р В±Р С•Р в„– HTML-Р С”Р С•Р Р…РЎвЂљР ВµР С”РЎРѓРЎвЂљ Р В±РЎР‚Р В°РЎС“Р В·Р ВµРЎР‚ РЎС“Р Р†Р С‘Р Т‘Р С‘РЎвЂљ Р В±Р ВµР В·Р С•Р С—Р В°РЎРѓР Р…РЎвЂ№Р в„– РЎвЂљР ВµР С”РЎРѓРЎвЂљ.
	// (Р В­РЎвЂљР С• Р С”Р С•Р СР С—РЎР‚Р С•Р СР С‘РЎРѓРЎРѓР Р…Р С•Р Вµ РЎР‚Р ВµРЎв‚¬Р ВµР Р…Р С‘Р Вµ: Р С—РЎР‚Р В°Р Р†Р С‘Р В»РЎРЉР Р…Р С• РЎРЊР С”РЎР‚Р В°Р Р…Р С‘РЎР‚Р С•Р Р†Р В°РЎвЂљРЎРЉ Р Р…Р В° Р вЂ™Р В«Р вЂ™Р С›Р вЂќР вЂў, Р В° Р Р…Р Вµ Р Р…Р В° Р вЂ™Р ТђР С›Р вЂќР вЂў,
	//  Р Р…Р С• Р Т‘Р В»РЎРЏ РЎС“РЎвЂЎР ВµР В±Р Р…Р С•Р в„– РЎР‚Р В°Р В±Р С•РЎвЂљРЎвЂ№ РЎРЊР С”РЎР‚Р В°Р Р…Р С‘РЎР‚РЎС“Р ВµР С Р В·Р В°РЎР‚Р В°Р Р…Р ВµР Вµ.)
	safeDescription := html.EscapeString(req.Description)
	task, err := h.svc.Create(req.Title, safeDescription, req.DueDate)
	if err != nil {
		internalError(w, rid, err, "create task")
		return
	}
	log.Printf("[%s] task created: %s", rid, task.ID)
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request, rid string) {
	tasks, err := h.svc.List()
	if err != nil {
		internalError(w, rid, err, "list tasks")
		return
	}
	log.Printf("[%s] list tasks: %d items", rid, len(tasks))
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, rid, id string) {
	task, ok, err := h.svc.Get(id)
	if err != nil {
		internalError(w, rid, err, "get task")
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type updateRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request, rid, id string) {
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	task, ok, err := h.svc.Update(id, req.Title, req.Done)
	if err != nil {
		internalError(w, rid, err, "update task")
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request, rid, id string) {
	ok, err := h.svc.Delete(id)
	if err != nil {
		internalError(w, rid, err, "delete task")
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Р В РЎвЂ”Р В РЎвЂўР В РЎвЂР РЋР С“Р В РЎвЂќ ----

func (h *Handler) searchTasks(w http.ResponseWriter, r *http.Request) {
	rid, errStatus := h.checkAuth(r.Context(), r)
	if errStatus != 0 {
		writeJSON(w, errStatus, map[string]string{"error": http.StatusText(errStatus)})
		return
	}
	title := r.URL.Query().Get("title")
	tasks, err := h.svc.Search(title)
	if err != nil {
		internalError(w, rid, err, "search")
		return
	}
	log.Printf("[%s] search title=%q: %d items", rid, title, len(tasks))
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) searchTasksVulnerable(w http.ResponseWriter, r *http.Request) {
	rid, errStatus := h.checkAuth(r.Context(), r)
	if errStatus != 0 {
		writeJSON(w, errStatus, map[string]string{"error": http.StatusText(errStatus)})
		return
	}
	title := r.URL.Query().Get("title")
	tasks, err := h.svc.SearchVulnerable(title)
	if err != nil {
		internalError(w, rid, err, "search vulnerable")
		return
	}
	log.Printf("[%s] VULNERABLE search title=%q: %d items", rid, title, len(tasks))
	writeJSON(w, http.StatusOK, tasks)
}
