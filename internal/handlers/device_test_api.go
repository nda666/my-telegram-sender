package handlers

import (
	"net/http"
)

func (h *Handlers) DeviceTestApi(w http.ResponseWriter, r *http.Request) {
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

	var fetchError string
	if !device.HasSession() { // fix: !HasSession
		fetchError = "Device belum terhubung ke Telegram. Silakan buat session terlebih dahulu."
	}

	h.render(w, r, "Devices/TestApi", map[string]any{
		"device": deviceJSON(*device),
		"error":  fetchError,
	})
}
