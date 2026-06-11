package auth

import (
	"net/http"

	"github.com/petaki/inertia-go"
	"github.com/tiar/telegram-sender/internal/services"
)

func RequireAuth(
	sm *SessionManager,
	users *services.UserService,
	inertia *inertia.Inertia,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := sm.GetUserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		user, err := users.FindByID(userID)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := inertia.WithProp(r.Context(), "auth", map[string]any{
			"user": map[string]any{
				"id":       user.ID,
				"username": user.Username,
				"name":     user.Name,
			},
		})

		next(w, r.WithContext(ctx))
	}
}
