package routes

import (
	"net/http"

	"github.com/petaki/inertia-go"
	"github.com/tiar/telegram-sender/internal/auth"
	"github.com/tiar/telegram-sender/internal/handlers"
)

func Register(mux *http.ServeMux, h *handlers.Handlers, i *inertia.Inertia, sessions *auth.SessionManager) {
	withAuth := func(fn http.HandlerFunc) http.Handler {
		return i.Middleware(auth.RequireAuth(sessions, fn))
	}

	mux.Handle("GET /login", i.Middleware(http.HandlerFunc(h.LoginShow)))
	mux.Handle("POST /login", i.Middleware(http.HandlerFunc(h.Login)))
	mux.Handle("POST /logout", withAuth(h.Logout))

	mux.Handle("GET /{$}", withAuth(func(w http.ResponseWriter, r *http.Request) {
		i.Location(w, r, "/devices")
	}))
	mux.Handle("GET /devices", withAuth(h.DevicesIndex))
	mux.Handle("GET /devices/create", withAuth(h.DevicesCreate))
	mux.Handle("POST /devices", withAuth(h.DevicesStore))
	mux.Handle("GET /devices/{id}/edit", withAuth(h.DevicesEdit))
	mux.Handle("PUT /devices/{id}", withAuth(h.DevicesUpdate))
	mux.Handle("DELETE /devices/{id}", withAuth(h.DevicesDelete))
	mux.Handle("GET /devices/{id}/session", withAuth(h.DeviceSessionShow))
	mux.Handle("POST /devices/{id}/session", withAuth(h.DeviceSessionPhone))
	mux.Handle("GET /devices/{id}/session/code", withAuth(h.DeviceSessionCodeShow)) // <-- baru
	mux.Handle("POST /devices/{id}/session/code", withAuth(h.DeviceSessionCode))
	mux.Handle("POST /devices/{id}/session/password", withAuth(h.DeviceSessionPassword))
	mux.Handle("GET /devices/{id}/session/check", withAuth(h.DeviceSessionCheck))
	mux.Handle("GET /devices/{id}/inbox", withAuth(h.DeviceInbox))
	mux.Handle("GET /devices/{id}/inbox/messages", withAuth(h.DeviceChatHistory))
	mux.Handle("POST /devices/{id}/inbox/send", withAuth(h.DeviceSendInboxMessage))
	mux.Handle("GET /devices/{id}/inbox/media", withAuth(h.DeviceMediaDownload))
	mux.Handle("GET /devices/{id}/status/stream", withAuth(h.DeviceStatusStream))
	mux.Handle("GET /devices/{id}/profile", withAuth(h.DeviceGetProfile))
	mux.Handle("GET /logs", withAuth(h.LogsIndex))
	// mux.Handle("/build/", http.StripPrefix("/build/", http.FileServer(http.Dir("./public/build"))))
}
