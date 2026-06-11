package auth

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const sessionName = "app_session"
const userIDKey = "user_id"

type SessionManager struct {
	store *sessions.CookieStore
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{
		store: sessions.NewCookieStore([]byte(secret)),
	}
}

func (m *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID uint) error {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[userIDKey] = userID
	return sess.Save(r, w)
}

func (m *SessionManager) GetUserID(r *http.Request) (uint, bool) {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	v, ok := sess.Values[userIDKey].(uint)
	if !ok {
		if iv, ok := sess.Values[userIDKey].(int); ok {
			return uint(iv), true
		}
		return 0, false
	}
	return v, true
}

func (m *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

func (m *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
