package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tiar/telegram-sender/internal/services"
)

type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func (h *Handlers) LoginShow(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.Session.GetUserID(r); ok {
		h.redirect(w, r, "/devices")
		return
	}
	h.render(w, r, "Auth/Login", nil)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[LoginRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	fmt.Println(username, password)
	user, err := h.Users.Authenticate(username, password)
	if err != nil {
		msg := "Login gagal"
		if errors.Is(err, services.ErrInvalidCredentials) {
			msg = "Username atau password salah"
		}
		h.render(w, r, "Auth/Login", map[string]any{"error": msg})
		return
	}

	if err := h.Session.SetUserID(w, r, user.ID); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	h.Logs.Write("info", "auth.login", "User login: "+user.Username, nil)
	h.redirect(w, r, "/devices")
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.Session.Clear(w, r)
	h.redirect(w, r, "/login")
}
