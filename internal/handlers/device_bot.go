package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tiar/telegram-sender/internal/telegram"
)

type ConnectBotRequest struct {
	BotToken string `json:"botToken" form:"botToken"`
}

type ToggleBotRequest struct {
	BotEnabled bool `json:"botEnabled" form:"botEnabled"`
}

// GET /devices/{id}/bot
func (h *Handlers) DeviceBotShow(w http.ResponseWriter, r *http.Request) {
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

	h.render(w, r, "Devices/Bot", map[string]any{
		"device": deviceJSON(*device),
	})
}

// POST /devices/{id}/bot/connect
func (h *Handlers) DeviceBotConnect(w http.ResponseWriter, r *http.Request) {
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

	req, err := Bind[ConnectBotRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.BotToken = strings.TrimSpace(req.BotToken)
	if req.BotToken == "" {
		h.render(w, r, "Devices/Bot", map[string]any{
			"device": deviceJSON(*device),
			"error":  "Bot Token tidak boleh kosong.",
		})
		return
	}

	botUser, err := h.Telegram.ConnectBot(r.Context(), id, req.BotToken)
	if err != nil {
		h.render(w, r, "Devices/Bot", map[string]any{
			"device": deviceJSON(*device),
			"error":  fmt.Sprintf("Gagal menghubungkan bot: %s", err.Error()),
		})
		return
	}

	_ = botUser
	h.redirect(w, r, fmt.Sprintf("/devices/%d/bot", id))
}

// POST /devices/{id}/bot/disconnect
func (h *Handlers) DeviceBotDisconnect(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := h.Telegram.DisconnectBot(r.Context(), id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	h.redirect(w, r, fmt.Sprintf("/devices/%d/bot", id))
}

// POST /devices/{id}/bot/toggle
func (h *Handlers) DeviceBotToggle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	req, _ := Bind[ToggleBotRequest](r)

	if err := h.Telegram.ToggleBot(r.Context(), id, req.BotEnabled); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	h.redirect(w, r, fmt.Sprintf("/devices/%d/bot", id))
}

type SetBotCommandsRequest struct {
	Commands []telegram.BotCommandItem `json:"commands"`
}

// GET /devices/{id}/bot/commands
func (h *Handlers) DeviceBotGetCommands(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	cmds, err := h.Telegram.GetBotCommands(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"commands": cmds,
	})
}

// POST /devices/{id}/bot/commands
func (h *Handlers) DeviceBotSetCommands(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid device ID"})
		return
	}

	req, err := Bind[SetBotCommandsRequest](r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad request body"})
		return
	}

	if err := h.Telegram.SetBotCommands(r.Context(), id, req.Commands); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"commands": req.Commands,
	})
}

type UpdateBotStartConfigRequest struct {
	BotRequestPhoneOnStart  bool   `json:"botRequestPhoneOnStart" form:"botRequestPhoneOnStart"`
	BotStartMessage         string `json:"botStartMessage" form:"botStartMessage"`
	BotPhoneButtonText      string `json:"botPhoneButtonText" form:"botPhoneButtonText"`
	BotPhoneReceivedMessage string `json:"botPhoneReceivedMessage" form:"botPhoneReceivedMessage"`
}

// POST /devices/{id}/bot/start-config
func (h *Handlers) DeviceBotUpdateStartConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	req, err := Bind[UpdateBotStartConfigRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.Devices.UpdateBotStartConfig(id, req.BotRequestPhoneOnStart, req.BotStartMessage, req.BotPhoneButtonText, req.BotPhoneReceivedMessage); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	h.Telegram.Webhook().RestartDevice(id)
	h.redirect(w, r, fmt.Sprintf("/devices/%d/bot", id))
}

type UpdateBotMenuConfigRequest struct {
	BotMenuSource string `json:"botMenuSource" form:"botMenuSource"`
	BotMenuURL    string `json:"botMenuUrl" form:"botMenuUrl"`
}

// POST /devices/{id}/bot/menu-config
func (h *Handlers) DeviceBotUpdateMenuConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	req, err := Bind[UpdateBotMenuConfigRequest](r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.BotMenuSource = strings.ToLower(strings.TrimSpace(req.BotMenuSource))
	if req.BotMenuSource == "" {
		req.BotMenuSource = "manual"
	}
	req.BotMenuURL = strings.TrimSpace(req.BotMenuURL)

	if err := h.Devices.UpdateBotMenuConfig(id, req.BotMenuSource, req.BotMenuURL); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Terapkan default commands ke Telegram
	if req.BotMenuSource == "url" {
		_ = h.Telegram.SetBotCommands(r.Context(), id, telegram.DefaultBotCommands())
	}

	h.Telegram.Webhook().RestartDevice(id)
	h.redirect(w, r, fmt.Sprintf("/devices/%d/bot", id))
}



