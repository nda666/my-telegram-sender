package handlers

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

const pendingCookie = "tg_pending"

type SessionCodeRequest struct {
	Code string `json:"code" form:"code" validate:"required,numeric,min=4,max=6"`
}

type SessionPasswordRequest struct {
	Password string `json:"password" form:"password" validate:"required"`
}

func getSessionValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"global": "Terjadi kesalahan validasi data."}
	}
	for _, errField := range ve {
		fieldKey := strings.ToLower(errField.Field())
		switch errField.Field() {
		case "Code":
			if errField.Tag() == "required" {
				errors[fieldKey] = "Kode OTP wajib diisi."
			} else {
				errors[fieldKey] = "Kode OTP harus berupa angka 4-6 digit."
			}
		case "Password":
			errors[fieldKey] = "Password 2FA wajib diisi."
		default:
			errors[fieldKey] = "Input tidak valid."
		}
	}
	return errors
}

// GET /devices/{id}/session
// Tampilkan step berdasarkan cookie pending
func (h *Handlers) DeviceSessionShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	step := "phone"
	var phone string

	if c, err := r.Cookie(pendingCookie); err == nil && c.Value != "" {
		if pending, ok := h.Pending.Get(c.Value); ok && pending.DeviceID == id {
			step = "code"
			phone = pending.Phone // ambil phone dari pending store
		}
	}

	props := map[string]any{
		"device": deviceJSON(*device),
		"step":   step,
	}
	if phone != "" {
		props["phone"] = phone
	}
	h.render(w, r, "Devices/Session", props)
}

// GET /devices/{id}/session/code
// "Saya sudah punya kode OTP" — hanya bisa diakses jika ada pending session
func (h *Handlers) DeviceSessionCodeShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Harus ada pending session dulu (dari SendCode sebelumnya)
	cookie, err := r.Cookie(pendingCookie)
	if err != nil || cookie.Value == "" {
		h.redirect(w, r, "/devices/"+r.PathValue("id")+"/session")
		return
	}

	pending, ok := h.Pending.Get(cookie.Value)
	if !ok || pending.DeviceID != id {
		h.redirect(w, r, "/devices/"+r.PathValue("id")+"/session")
		return
	}

	h.render(w, r, "Devices/Session", map[string]any{
		"device": deviceJSON(*device),
		"step":   "code",
		"phone":  pending.Phone,
	})
}

// POST /devices/{id}/session
// Kirim kode OTP ke nomor device
func (h *Handlers) DeviceSessionPhone(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	phone := strings.TrimSpace(device.Phone)
	if phone == "" {
		h.render(w, r, "Devices/Session", map[string]any{
			"device": deviceJSON(*device),
			"step":   "phone",
			"error":  "Nomor telepon device kosong. Silakan edit device terlebih dahulu.",
		})
		return
	}

	codeHash, err := h.Telegram.SendCode(r.Context(), id, phone)
	if err != nil {
		if strings.Contains(err.Error(), "AUTH_RESTART") {
			// Clear session lalu retry sekali
			_ = h.Devices.ClearSession(id)
			codeHash, err = h.Telegram.SendCode(r.Context(), id, phone)
		}
		if err != nil {
			h.render(w, r, "Devices/Session", map[string]any{
				"device": deviceJSON(*device),
				"step":   "phone",
				"error":  err.Error(),
			})
			return
		}
	}

	pendingID, err := h.Pending.Create(id, phone, codeHash)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pendingCookie,
		Value:    pendingID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	h.render(w, r, "Devices/Session", map[string]any{
		"device": deviceJSON(*device),
		"step":   "code",
		"phone":  phone,
	})
}

// POST /devices/{id}/session/code
// Verifikasi kode OTP
func (h *Handlers) DeviceSessionCode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cookie, err := r.Cookie(pendingCookie)
	if err != nil || cookie.Value == "" {
		h.redirect(w, r, "/devices/"+r.PathValue("id")+"/session")
		return
	}

	pending, ok := h.Pending.Get(cookie.Value)
	if !ok || pending.DeviceID != id {
		h.redirect(w, r, "/devices/"+r.PathValue("id")+"/session")
		return
	}

	req, err := Bind[SessionCodeRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Code = strings.TrimSpace(req.Code)

	if err := validate.Struct(req); err != nil {
		errs := getSessionValidationErrors(err)
		h.render(w, r, "Devices/Session", map[string]any{
			"device":    deviceJSON(*device),
			"step":      "code",
			"phone":     pending.Phone,
			"error":     errs["code"],
			"error_bag": errs,
		})
		return
	}

	needsPassword, err := h.Telegram.SignIn(r.Context(), id, pending.Phone, req.Code, pending.CodeHash)
	if err != nil {
		// Pesan error lebih user-friendly
		msg := err.Error()
		if strings.Contains(msg, "PHONE_CODE_INVALID") {
			msg = "Kode OTP salah. Silakan coba lagi."
		} else if strings.Contains(msg, "PHONE_CODE_EXPIRED") {
			msg = "Kode OTP sudah kedaluwarsa. Silakan mulai ulang."
		}
		h.render(w, r, "Devices/Session", map[string]any{
			"device": deviceJSON(*device),
			"step":   "code",
			"phone":  pending.Phone,
			"error":  msg,
		})
		return
	}

	if needsPassword {
		// Jangan hapus pending — dibutuhkan kalau user refresh
		h.render(w, r, "Devices/Session", map[string]any{
			"device": deviceJSON(*device),
			"step":   "password",
		})
		return
	}

	h.Pending.Delete(cookie.Value)
	clearPendingCookie(w)
	h.redirect(w, r, "/devices")
}

// POST /devices/{id}/session/password
// Verifikasi password 2FA
func (h *Handlers) DeviceSessionPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	req, err := Bind[SessionPasswordRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		h.render(w, r, "Devices/Session", map[string]any{
			"device": deviceJSON(*device),
			"step":   "password",
			"error":  "Password 2FA wajib diisi.",
		})
		return
	}

	if err := h.Telegram.SignInPassword(r.Context(), id, req.Password); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "PASSWORD_HASH_INVALID") {
			msg = "Password 2FA salah. Silakan coba lagi."
		}
		h.render(w, r, "Devices/Session", map[string]any{
			"device": deviceJSON(*device),
			"step":   "password",
			"error":  msg,
		})
		return
	}

	if c, err := r.Cookie(pendingCookie); err == nil {
		h.Pending.Delete(c.Value)
	}
	clearPendingCookie(w)
	h.redirect(w, r, "/devices")
}

// POST /devices/{id}/session/check
func (h *Handlers) DeviceSessionCheck(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if _, err := h.Devices.Find(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	status, err := h.Telegram.CheckOnline(r.Context(), id)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	idCopy := id
	h.Logs.Write("info", "device.check", "Status: "+status, &idCopy)
	h.redirect(w, r, "/devices")
}

func clearPendingCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     pendingCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
