package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tiar/telegram-sender/internal/telegram"
)

type SendMessageRequest struct {
	Type       string `json:"type" form:"type" validate:"required"`
	PeerID     string `json:"peer_id" form:"peer_id" validate:"required"`
	AccessHash string `json:"access_hash" form:"access_hash"`
	Message    string `json:"message" form:"message" validate:"required"`
}

func (h *Handlers) DeviceInbox(w http.ResponseWriter, r *http.Request) {
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

	var chats []telegram.ChatItem
	var fetchError string

	if device.HasSession() {
		c, err := h.Telegram.GetDialogs(r.Context(), id)
		if err != nil {
			fetchError = err.Error()
		} else {
			chats = c
		}
	} else {
		fetchError = "Device belum terhubung ke Telegram. Silakan buat session terlebih dahulu."
	}

	h.render(w, r, "Devices/Inbox", map[string]any{
		"device": deviceJSON(*device),
		"chats":  chats,
		"error":  fetchError,
	})
}

func (h *Handlers) DeviceChatHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	peerType := r.URL.Query().Get("type")
	peerIDStr := r.URL.Query().Get("peer_id")
	accessHashStr := r.URL.Query().Get("access_hash")
	offsetIDStr := r.URL.Query().Get("offset_id") // ID pesan tertua yang sudah dimuat

	if peerType == "" || peerIDStr == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing parameters"})
		return
	}

	peerID, err := strconv.ParseInt(peerIDStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid peer ID"})
		return
	}

	var accessHash int64
	if accessHashStr != "" {
		accessHash, err = strconv.ParseInt(accessHashStr, 10, 64)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid access hash"})
			return
		}
	}

	var offsetID int
	if offsetIDStr != "" {
		v, err := strconv.Atoi(offsetIDStr)
		if err == nil {
			offsetID = v
		}
	}

	messages, err := h.Telegram.GetMessages(r.Context(), id, peerType, peerID, accessHash, 50, offsetID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
	})
}

func (h *Handlers) DeviceSendInboxMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	req, err := Bind[SendMessageRequest](r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := validate.Struct(req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Validation failed"})
		return
	}

	peerID, err := strconv.ParseInt(req.PeerID, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid peer ID"})
		return
	}

	var accessHash int64
	if req.AccessHash != "" {
		accessHash, err = strconv.ParseInt(req.AccessHash, 10, 64)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid access hash"})
			return
		}
	}

	err = h.Telegram.SendTelegramMessage(r.Context(), id, req.Type, peerID, accessHash, req.Message)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (h *Handlers) DeviceMediaDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	msgIDStr := r.URL.Query().Get("msg_id")
	mediaType := r.URL.Query().Get("media_type") // kirim dari React
	peerType := r.URL.Query().Get("type")
	peerIDStr := r.URL.Query().Get("peer_id")
	accessHashStr := r.URL.Query().Get("access_hash")

	msgID, _ := strconv.Atoi(msgIDStr)
	peerID, _ := strconv.ParseInt(peerIDStr, 10, 64)
	accessHash, _ := strconv.ParseInt(accessHashStr, 10, 64)

	fmt.Println("mediaType:", mediaType)
	// Set Content-Type berdasarkan mediaType dari client
	switch mediaType {
	case "photo":
		w.Header().Set("Content-Type", "image/jpeg")
	case "gif":
		w.Header().Set("Content-Type", "video/mp4")
	case "video":
		w.Header().Set("Content-Type", "video/mp4")
	case "voice":
		w.Header().Set("Content-Type", "audio/ogg")
	case "audio":
		w.Header().Set("Content-Type", "audio/mpeg")
	case "sticker":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Cache-Control", "max-age=3600")

	err := h.Telegram.DownloadMedia(r.Context(), id, peerType, peerID, accessHash, msgID, w)
	if err != nil {
		// header sudah terkirim kalau ada data, log saja
		_ = err
	}
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
