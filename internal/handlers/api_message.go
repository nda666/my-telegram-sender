package handlers

import (
	"net/http"
	"strconv"
)

// APISendMessage — POST /api/send
// Header: X-Api-Key: <uuid>
// Body: { "chat_id": "123456789", "message": "Hello!", "peer_type": "user", "access_hash": 0 }
//
// peer_type: "user" | "chat" | "channel" (default: "user")
// access_hash: wajib untuk user/channel, 0 untuk chat group biasa
func (h *Handlers) APISendMessage(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey == "" {
		jsonError(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	device, err := h.Devices.FindByAPIKey(apiKey)
	if err != nil {
		jsonError(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	type Req struct {
		ChatID     string `json:"chat_id"`
		Message    string `json:"message"`
		PeerType   string `json:"peer_type"`   // "user" | "chat" | "channel"
		AccessHash int64  `json:"access_hash"` // 0 untuk group chat biasa
	}
	req, err := Bind[Req](r)
	if err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ChatID == "" || req.Message == "" {
		jsonError(w, "chat_id and message are required", http.StatusUnprocessableEntity)
		return
	}

	peerID, err := strconv.ParseInt(req.ChatID, 10, 64)
	if err != nil {
		jsonError(w, "chat_id must be a numeric Telegram ID", http.StatusUnprocessableEntity)
		return
	}

	peerType := req.PeerType
	if peerType == "" {
		peerType = "user"
	}

	if err := h.Telegram.SendTelegramMessage(r.Context(), device.ID, peerType, peerID, req.AccessHash, req.Message); err != nil {
		jsonError(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"message": "Message sent"})
}
