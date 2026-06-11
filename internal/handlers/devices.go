package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

var validate = validator.New()

type CreateDeviceRequest struct {
	Name  string `json:"name" form:"name" validate:"required"`
	Phone string `json:"phone" form:"phone" validate:"required,e164"`
}

type UpdateDeviceRequest struct {
	Name  string `json:"name" form:"name" validate:"required"`
	Phone string `json:"phone" form:"phone" validate:"required,e164"`
}

// Helper untuk generate pesan error dinamis sesuai field yang melanggar rule
func getValidationError(err error) string {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return "Terjadi kesalahan validasi data."
	}

	// Ambil error pertama saja biar user fokus benerin satu-satu
	errField := ve[0]

	switch errField.Field() {
	case "Name":
		if errField.Tag() == "required" {
			return "Nama device wajib diisi."
		}
	case "Phone":
		if errField.Tag() == "required" {
			return "Nomor HP wajib diisi."
		}
		if errField.Tag() == "e164" {
			return "Format nomor HP salah. Harus format internasional (contoh: +62812345678)."
		}
	}

	return "Data yang dikirim tidak valid."
}

func getValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"global": "Terjadi kesalahan validasi data."}
	}

	for _, errField := range ve {
		// Ubah nama field ke lowercase/snake_case biar match dengan attribute name di HTML Form
		fieldKey := strings.ToLower(errField.Field())

		switch errField.Field() {
		case "Name":
			if errField.Tag() == "required" {
				errors[fieldKey] = "Nama device wajib diisi."
			}
		case "Phone":
			if errField.Tag() == "required" {
				errors[fieldKey] = "Nomor HP wajib diisi."
			} else if errField.Tag() == "e164" {
				errors[fieldKey] = "Format nomor HP salah (contoh: +62812345678)."
			}
		default:
			errors[fieldKey] = "Data tidak valid."
		}
	}

	return errors
}

func (h *Handlers) DevicesIndex(w http.ResponseWriter, r *http.Request) {
	devices, err := h.Devices.List()
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]any, len(devices))
	for i, d := range devices {
		items[i] = deviceJSON(d)
	}

	// Pindah ke bawah loop biar gak race condition saat mapping ke 'items'
	go h.Telegram.CheckAllStatus(context.Background(), devices)

	h.render(w, r, "Devices/Index", map[string]any{
		"devices": items,
	})
}

func (h *Handlers) DevicesCreate(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "Devices/Form", map[string]any{
		"device": nil,
	})
}

func (h *Handlers) DevicesStore(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[CreateDeviceRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)

	if err := validate.Struct(req); err != nil {
		// Panggil helper buat nentuin pesan error-nya
		errs := getValidationErrors(err)
		errMsg := getValidationError(err)

		h.render(w, r, "Devices/Form", map[string]any{
			"device":    map[string]any{"name": req.Name, "phone": req.Phone},
			"error":     errMsg,
			"error_bag": errs,
		})
		return
	}

	if _, err := h.Devices.Create(req.Name, req.Phone); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	h.redirect(w, r, "/devices")
}

func (h *Handlers) DevicesEdit(w http.ResponseWriter, r *http.Request) {
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

	h.render(w, r, "Devices/Form", map[string]any{
		"device": deviceJSON(*device),
	})
}

func (h *Handlers) DevicesUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	req, err := Bind[UpdateDeviceRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)

	if err := validate.Struct(req); err != nil {
		// Panggil helper buat nentuin pesan error-nya
		errMsg := getValidationError(err)
		errs := getValidationErrors(err)

		h.render(w, r, "Devices/Form", map[string]any{
			"device":    map[string]any{"id": id, "name": req.Name, "phone": req.Phone},
			"error":     errMsg,
			"error_bag": errs,
		})
		return
	}

	if _, err := h.Devices.Update(id, req.Name, req.Phone); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	h.redirect(w, r, "/devices")
}

func (h *Handlers) DevicesDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := h.Devices.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	h.redirect(w, r, "/devices")
}

func deviceJSON(d models.Device) map[string]any {
	return map[string]any{
		"id":                d.ID,
		"name":              d.Name,
		"apiKey":            d.ApiKey,
		"phone":             d.Phone,
		"telegramUserId":    d.TelegramUserID,
		"telegramFirstName": d.TelegramFirstName,
		"telegramLastName":  d.TelegramLastName,
		"telegramPhone":     d.TelegramPhone,
		"avatarColor":       d.AvatarColor,
		"status":            d.DisplayStatus(),
		"hasSession":        d.HasSession(),
		"createdAt":         d.CreatedAt.Format("2006-01-02 15:04"),
	}
}

func (h *Handlers) DeviceStatusStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // penting kalau pakai nginx

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Kirim status awal langsung
	device, err := h.Devices.Find(id)
	if err != nil {
		return
	}
	initialStatus := "no_session"
	if device.HasSession() {
		initialStatus = device.Status
	}
	fmt.Fprintf(w, "data: %s\n\n", initialStatus)
	flusher.Flush()

	ctx := r.Context()
	h.Telegram.WatchOnline(ctx, id, 3*time.Second, func(status string) {
		fmt.Fprintf(w, "data: %s\n\n", status)
		flusher.Flush()
	})
}

func (h *Handlers) DeviceGetProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	device, err := h.Devices.Find(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Device not found"})
		return
	}

	if !device.HasSession() {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Device belum punya session"})
		return
	}

	h.Devices.EnsureAPIKey(int64(id))

	err = h.Telegram.RefreshProfile(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Ambil ulang device setelah update
	device, err = h.Devices.Find(id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"device": deviceJSON(*device),
	})
}
