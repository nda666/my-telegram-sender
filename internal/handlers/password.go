package handlers

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// ShowChangePassword — GET /settings/password
func (h *Handlers) ShowChangePassword(w http.ResponseWriter, r *http.Request) {
	h.Inertia.Render(w, r, "Settings/ChangePassword", map[string]any{})
}

// ChangePassword — POST /settings/password
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userId, _ := h.Session.GetUserID(r)

	type Req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		Confirmation    string `json:"password_confirmation"`
	}

	var req Req
	req, err := Bind[Req](r)
	if err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.Confirmation {
		jsonError(w, "Password confirmation does not match", http.StatusUnprocessableEntity)
		return
	}
	if len(req.NewPassword) < 8 {
		jsonError(w, "Password must be at least 8 characters", http.StatusUnprocessableEntity)
		return
	}

	dbUser, err := h.Users.FindByID(userId)
	if err != nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(req.CurrentPassword)); err != nil {
		jsonError(w, "Current password is incorrect", http.StatusUnprocessableEntity)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	if err := h.Users.UpdatePassword(userId, string(hash)); err != nil {
		jsonError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "Password updated successfully"})
}
