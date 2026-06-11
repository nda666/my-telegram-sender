package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/petaki/inertia-go"
	"github.com/tiar/telegram-sender/internal/auth"
	"github.com/tiar/telegram-sender/internal/services"
	"github.com/tiar/telegram-sender/internal/telegram"
)

type Handlers struct {
	Inertia  *inertia.Inertia
	Session  *auth.SessionManager
	Users    *services.UserService
	Devices  *services.DeviceService
	Logs     *services.LogService
	Telegram *telegram.Service
	Pending  *auth.PendingStore
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, page string, props map[string]any) {
	r = h.withAuth(r)
	if props == nil {
		props = map[string]any{}
	}
	if err := h.Inertia.Render(w, r, page, props); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h *Handlers) redirect(w http.ResponseWriter, r *http.Request, url string) {
	h.Inertia.Location(w, r, url)
}

func (h *Handlers) withAuth(r *http.Request) *http.Request {
	userID, ok := h.Session.GetUserID(r)
	if !ok {
		return r
	}
	user, err := h.Users.FindByID(userID)
	if err != nil {
		return r
	}
	ctx := h.Inertia.WithProp(r.Context(), "auth", map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
		},
	})
	return r.WithContext(ctx)
}

func parseID(r *http.Request, key string) (uint, bool) {
	v := r.PathValue(key)
	if v == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}
