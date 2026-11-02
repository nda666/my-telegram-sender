package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

type UpdateWebhookRequest struct {
	WebhookURL     string `json:"webhookUrl" form:"webhookUrl" validate:"omitempty,url"`
	WebhookSecret  string `json:"webhookSecret" form:"webhookSecret"`
	WebhookEnabled bool   `json:"webhookEnabled" form:"webhookEnabled"`
}

// GET /devices/{id}/webhook
func (h *Handlers) DeviceWebhookShow(w http.ResponseWriter, r *http.Request) {
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

	h.render(w, r, "Devices/Webhook", map[string]any{
		"device": deviceJSON(*device),
	})
}

// POST /devices/{id}/webhook
func (h *Handlers) DeviceWebhookUpdate(w http.ResponseWriter, r *http.Request) {
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

	req, err := Bind[UpdateWebhookRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.WebhookURL = strings.TrimSpace(req.WebhookURL)
	req.WebhookSecret = strings.TrimSpace(req.WebhookSecret)

	if req.WebhookURL != "" && !strings.HasPrefix(req.WebhookURL, "http://") && !strings.HasPrefix(req.WebhookURL, "https://") {
		h.render(w, r, "Devices/Webhook", map[string]any{
			"device": deviceJSON(*device),
			"error":  "Format URL Webhook tidak valid (harus diawali http:// atau https://).",
		})
		return
	}

	if err := h.Devices.UpdateWebhook(id, req.WebhookURL, req.WebhookSecret, req.WebhookEnabled); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Restart listener dengan konfigurasi baru
	h.Telegram.Webhook().RestartDevice(id)

	h.Logs.Write("info", "device.webhook_update", fmt.Sprintf("Webhook diperbarui: %s (Aktif: %t)", req.WebhookURL, req.WebhookEnabled), &id)
	h.redirect(w, r, fmt.Sprintf("/devices/%d/webhook", id))
}

type TestWebhookRequest struct {
	WebhookURL    string `json:"webhookUrl" form:"webhookUrl"`
	WebhookSecret string `json:"webhookSecret" form:"webhookSecret"`
}

// POST /devices/{id}/webhook/test
func (h *Handlers) DeviceWebhookTest(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	req, _ := Bind[TestWebhookRequest](r)

	result, err := h.Telegram.Webhook().TestWebhook(id, req.WebhookURL, req.WebhookSecret)
	if err != nil && result == nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}
