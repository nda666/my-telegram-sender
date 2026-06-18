package routes

import (
	"net/http"

	"github.com/petaki/inertia-go"
	"github.com/tiar/telegram-sender/internal/auth"
	"github.com/tiar/telegram-sender/internal/handlers"
	"github.com/tiar/telegram-sender/internal/services"
)

func Register(mux *http.ServeMux, h *handlers.Handlers, i *inertia.Inertia, sessions *auth.SessionManager, userSvc *services.UserService) {
	withAuth := func(fn http.HandlerFunc) http.Handler {
		return i.Middleware(auth.RequireAuth(sessions, userSvc, i, fn))
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
	mux.Handle("GET /devices/{id}/profile", withAuth(h.DeviceGetProfile))
	mux.Handle("GET /devices/{id}/test-api", withAuth(h.DeviceTestApi))

	// Contacts CRUD
	mux.Handle("GET /devices/{id}/contacts", withAuth(h.IndexContacts))
	mux.Handle("POST /devices/{id}/contacts", withAuth(h.StoreContact))
	mux.Handle("PUT /devices/{id}/contacts", withAuth(h.UpdateContact))
	mux.Handle("DELETE /devices/{id}/contacts/{user_telegram_id}", withAuth(h.DeleteContact))

	mux.Handle("GET /logs", withAuth(h.LogsIndex))
	// mux.Handle("/build/", http.StripPrefix("/build/", http.FileServer(http.Dir("./public/build"))))

	// Settings - Change Password
	mux.Handle("GET /settings/password", withAuth(h.ShowChangePassword))
	mux.Handle("POST /settings/password", withAuth(h.ChangePassword))

	// Public API — pakai api key, tidak perlu session auth
	mux.Handle("POST /api/send", http.HandlerFunc(h.APISendMessage))
}
