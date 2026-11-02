package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"errors"

	"github.com/gotd/td/tgerr"
	"github.com/tiar/telegram-sender/internal/telegram"
)

type apiErrorResponse struct {
	Error     string         `json:"error"`
	ErrorType string         `json:"error_type"`
	Detail    map[string]any `json:"detail,omitempty"`
}

type apiTelegramError struct {
	Message    string
	Type       string
	Detail     map[string]any
	HTTPStatus int
}

type apiSendMessageResponse struct {
	Message string `json:"message" example:"Message sent"`
}

type apiSendMessageRequest struct {
	Phone         string `json:"phone" example:"6281234567890"`
	ChatID        string `json:"chat_id" example:"123456789"`
	Message       string `json:"message" example:"Hello World"`
	PeerType      string `json:"peer_type" example:"user"`
	AccessHash    int64  `json:"access_hash" example:"0"`
	Caption       string `json:"caption" example:"Photo Caption"`
	MediaURL      string `json:"media_url" example:"https://example.com/photo.jpg"`
	MediaBase64   string `json:"media_base64"`
	MediaFilename string `json:"media_filename" example:"photo.jpg"`
	ButtonType    string `json:"button_type" example:"inline"`
	Buttons       any    `json:"buttons"`
}

const DefaultTelegramErrorType = "telegram_error"

func telegramErrorToAPI(err error) apiTelegramError {
	if err == nil {
		return apiTelegramError{Message: "", Type: "unknown_error", Detail: nil, HTTPStatus: http.StatusInternalServerError}
	}

	// If telegram service returns a typed error, use it (custom typed interface tetap dipertahankan).
	type typed interface {
		APIError() (msg string, errType string, detail map[string]any, httpStatus int)
	}
	if te, ok := err.(typed); ok {
		msg, errType, detail, httpStatus := te.APIError()
		if msg == "" {
			msg = err.Error()
		}
		if errType == "" {
			errType = "telegram_error"
		}
		if httpStatus == 0 {
			httpStatus = http.StatusInternalServerError
		}
		return apiTelegramError{Message: msg, Type: errType, Detail: detail, HTTPStatus: httpStatus}
	}

	// Cek apakah ini error RPC dari Telegram (gotd)
	var rpcErr *tgerr.Error
	if errors.As(err, &rpcErr) {
		httpStatus := http.StatusInternalServerError
		switch rpcErr.Code {
		case 420:
			httpStatus = http.StatusTooManyRequests
		case 400:
			httpStatus = http.StatusBadRequest
		case 401:
			httpStatus = http.StatusUnauthorized
		case 403:
			httpStatus = http.StatusForbidden
		case 404:
			httpStatus = http.StatusNotFound
		}

		detail := map[string]any{"code": rpcErr.Code}
		if rpcErr.Argument > 0 {
			detail["wait_seconds"] = rpcErr.Argument
		}

		return apiTelegramError{
			Message:    rpcErr.Message,
			Type:       rpcErr.Type, // langsung ambil dari gotd, ex: "FLOOD_WAIT_X", "PEER_ID_INVALID"
			Detail:     detail,
			HTTPStatus: httpStatus,
		}
	}

	return apiTelegramError{Message: err.Error(), Type: "unknown_error", Detail: nil, HTTPStatus: http.StatusInternalServerError}
}

func jsonErrorTyped(w http.ResponseWriter, msg, errorType string, detail map[string]any, code int) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := apiErrorResponse{Error: msg, ErrorType: errorType, Detail: detail}
	_ = json.NewEncoder(w).Encode(resp)
}

// APISendMessage godoc
//
// @Summary Send Telegram message
// @Description Send text, photo, video, audio, or document using a registered device API key.
// @Description
// @Description Authentication:
// @Description - Header: X-Api-Key
// @Description - Or query parameter: api_key
//
// @Tags API
//
// @Accept application/json
// @Accept multipart/form-data
//
// @Produce application/json
//
// @Param X-Api-Key header string true "Device API Key"
// @Param api_key query string false "Device API Key"
//
// @Param request body apiSendMessageRequest false "JSON Request"
//
// @Param phone formData string false "Target phone number"
// @Param chat_id formData string false "Telegram Chat ID"
// @Param peer_type formData string false "user, chat, channel"
// @Param access_hash formData integer false "Telegram access hash"
// @Param message formData string false "Message text"
// @Param caption formData string false "Media caption"
// @Param file formData file false "Media file"
//
// @Success 200 {object} apiSendMessageResponse
// @Failure 400 {object} apiErrorResponse
// @Failure 401 {object} apiErrorResponse
// @Failure 422 {object} apiErrorResponse
// @Failure 500 {object} apiErrorResponse
//
// @Router /api/send [post]
func (h *Handlers) APISendMessage(w http.ResponseWriter, r *http.Request) {
	clientIP := GetClientIP(r)

	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey == "" {
		h.Logs.WriteDetail("warn", "api.auth", "API send ditolak: Missing API key", clientIP, "", nil)
		jsonErrorTyped(w, "Missing API key", "invalid_api_key", nil, http.StatusUnauthorized)
		return
	}

	device, err := h.Devices.FindByAPIKey(apiKey)
	if err != nil {
		h.Logs.WriteDetail("warn", "api.auth", fmt.Sprintf("API send ditolak: Invalid API key '%s'", apiKey), clientIP, "", nil)
		jsonErrorTyped(w, "Invalid API key", "invalid_api_key", nil, http.StatusUnauthorized)
		return
	}

	isBot := false
	if r.URL.Query().Get("bot") == "1" || strings.ToLower(r.URL.Query().Get("bot")) == "true" {
		isBot = true
	}
	if r.Header.Get("X-Bot") == "1" || strings.ToLower(r.Header.Get("X-Bot")) == "true" {
		isBot = true
	}

	// ── parse request ──────────────────────────────────────────────────────────
	var (
		chatIDStr   string
		phoneTarget string
		peerType    string
		accessHash  int64
		message     string
		caption     string
		mediaReader io.Reader
		mediaName   string
		buttonType  string
		buttonsRaw  any
	)

	ct := r.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "multipart/form-data") {
		// ── multipart ──────────────────────────────────────────────────────────
		if err := r.ParseMultipartForm(64 << 20); err != nil { // 64 MB
			jsonErrorTyped(w, "Failed to parse multipart form", "bad_request", nil, http.StatusBadRequest)
			return
		}

		if r.FormValue("bot") == "1" || strings.ToLower(r.FormValue("bot")) == "true" {
			isBot = true
		}

		chatIDStr = r.FormValue("chat_id")
		peerType = r.FormValue("peer_type")
		caption = r.FormValue("caption")
		message = r.FormValue("message")
		buttonType = r.FormValue("button_type")
		buttonsRaw = r.FormValue("buttons")
		if ah := r.FormValue("access_hash"); ah != "" {
			accessHash, _ = strconv.ParseInt(ah, 10, 64)
		}

		file, fh, ferr := r.FormFile("file")
		if ferr == nil {
			defer file.Close()
			buf := &bytes.Buffer{}
			if _, err := io.Copy(buf, file); err != nil {
				jsonError(w, "Failed to read uploaded file", http.StatusInternalServerError)
				return
			}
			mediaReader = buf
			mediaName = fh.Filename
		}
	} else {
		// ── JSON ───────────────────────────────────────────────────────────────
		type Req struct {
			Bot           any    `json:"bot"`
			Phone         string `json:"phone"`
			ChatID        string `json:"chat_id"`
			Message       string `json:"message"`
			PeerType      string `json:"peer_type"`
			AccessHash    int64  `json:"access_hash"`
			Caption       string `json:"caption"`
			MediaURL      string `json:"media_url"`
			MediaBase64   string `json:"media_base64"`
			MediaFilename string `json:"media_filename"`
			ButtonType    string `json:"button_type"`
			Buttons       any    `json:"buttons"`
		}
		req, err := Bind[Req](r)
		if err != nil {
			jsonErrorTyped(w, "Invalid request body", "bad_request", nil, http.StatusBadRequest)
			return
		}

		if req.Bot != nil {
			switch v := req.Bot.(type) {
			case bool:
				if v {
					isBot = true
				}
			case float64:
				if v == 1 {
					isBot = true
				}
			case int:
				if v == 1 {
					isBot = true
				}
			case string:
				if v == "1" || strings.ToLower(v) == "true" {
					isBot = true
				}
			}
		}

		chatIDStr = req.ChatID
		phoneTarget = req.Phone
		peerType = req.PeerType
		accessHash = req.AccessHash
		message = req.Message
		caption = req.Caption
		buttonType = req.ButtonType
		buttonsRaw = req.Buttons

		switch {
		case req.MediaURL != "":
			// download from URL
			resp, err := http.Get(req.MediaURL) //nolint:gosec
			if err != nil {
				jsonError(w, "Failed to download media: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer resp.Body.Close()
			buf := &bytes.Buffer{}
			if _, err := io.Copy(buf, resp.Body); err != nil {
				jsonError(w, "Failed to read media from URL", http.StatusInternalServerError)
				return
			}
			mediaReader = buf
			mediaName = filepath.Base(req.MediaURL)
			if idx := strings.Index(mediaName, "?"); idx != -1 {
				mediaName = mediaName[:idx]
			}
			if mediaName == "" || mediaName == "." {
				mediaName = "file"
			}

		case req.MediaBase64 != "":
			// strip data URI prefix if present: "data:image/jpeg;base64,..."
			b64 := req.MediaBase64
			if idx := strings.Index(b64, ","); idx != -1 {
				b64 = b64[idx+1:]
			}
			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				// try URL encoding
				raw, err = base64.URLEncoding.DecodeString(b64)
				if err != nil {
					jsonError(w, "Invalid base64 data", http.StatusBadRequest)
					return
				}
			}
			mediaReader = bytes.NewReader(raw)
			mediaName = req.MediaFilename
			if mediaName == "" {
				mediaName = "file"
			}
		}
	}

	if isBot {
		if !device.HasBot() || !device.BotEnabled {
			jsonErrorTyped(w, "Bot is not connected or enabled on this device", "bot_not_enabled", nil, http.StatusBadRequest)
			return
		}
	}

	// ── dispatch by phone (shortcut, skip peer resolution) ────────────────────
	if strings.HasPrefix(ct, "multipart/form-data") {
		phoneTarget = r.FormValue("phone")
	}

	// ── validate & dispatch ────────────────────────────────────────────────────
	mtype := classifyMedia(strings.ToLower(strings.TrimPrefix(filepath.Ext(mediaName), ".")))
	replyMarkup := telegram.ParseReplyMarkup(buttonsRaw, buttonType)

	// Simpan ringkasan request payload untuk log detail
	reqPayloadMap := map[string]any{
		"client_ip":   clientIP,
		"phone":       phoneTarget,
		"chat_id":     chatIDStr,
		"peer_type":   peerType,
		"access_hash": accessHash,
		"is_bot":      isBot,
		"message":     message,
		"caption":     caption,
		"media_name":  mediaName,
		"media_type":  mtype,
		"button_type": buttonType,
		"buttons":     buttonsRaw,
	}
	reqDataBytes, _ := json.Marshal(reqPayloadMap)
	reqDataStr := string(reqDataBytes)

	// PATH A: kirim by phone
	if phoneTarget != "" {
		if mediaReader != nil {
			var err error
			if isBot {
				err = h.Telegram.SendBotTelegramMediaByPhone(r.Context(), device.ID, phoneTarget, mediaReader, mediaName, mtype, caption, replyMarkup)
			} else {
				err = h.Telegram.SendTelegramMediaByPhone(r.Context(), device.ID, phoneTarget, mediaReader, mediaName, mtype, caption, replyMarkup)
			}
			if err != nil {
				apiErr := telegramErrorToAPI(err)
				h.Logs.WriteDetail("error", "api.send", fmt.Sprintf("Gagal kirim %s ke phone %s: %s", mtype, phoneTarget, apiErr.Message), clientIP, reqDataStr, &device.ID)
				jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
				return
			}

			h.Logs.WriteDetail("info", "api.send", fmt.Sprintf("Berhasil kirim %s ke phone %s (file: %s)", mtype, phoneTarget, mediaName), clientIP, reqDataStr, &device.ID)
			jsonOK(w, map[string]any{"message": fmt.Sprintf("%s sent", mtype)})
			return
		}
		if message == "" {
			h.Logs.WriteDetail("warn", "api.send", fmt.Sprintf("Validasi gagal untuk phone %s: message or media file is required", phoneTarget), clientIP, reqDataStr, &device.ID)
			jsonErrorTyped(w, "message or media file is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
			return
		}

		var err error
		if isBot {
			err = h.Telegram.SendBotTelegramMessageByPhone(r.Context(), device.ID, phoneTarget, message, replyMarkup)
		} else {
			err = h.Telegram.SendTelegramMessageByPhone(r.Context(), device.ID, phoneTarget, message, replyMarkup)
		}
		if err != nil {
			apiErr := telegramErrorToAPI(err)
			h.Logs.WriteDetail("error", "api.send", fmt.Sprintf("Gagal kirim pesan ke phone %s: %s", phoneTarget, apiErr.Message), clientIP, reqDataStr, &device.ID)
			jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
			return
		}

		h.Logs.WriteDetail("info", "api.send", fmt.Sprintf("Berhasil kirim pesan ke phone %s: %s", phoneTarget, message), clientIP, reqDataStr, &device.ID)
		jsonOK(w, map[string]any{"message": "Message sent"})
		return
	}

	// PATH B: kirim by chat_id (existing flow)
	if chatIDStr == "" {
		h.Logs.WriteDetail("warn", "api.send", "Validasi gagal: chat_id or phone is required", clientIP, reqDataStr, &device.ID)
		jsonErrorTyped(w, "chat_id or phone is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}
	peerID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		h.Logs.WriteDetail("warn", "api.send", fmt.Sprintf("Validasi gagal: chat_id '%s' must be a numeric Telegram ID", chatIDStr), clientIP, reqDataStr, &device.ID)
		jsonErrorTyped(w, "chat_id must be a numeric Telegram ID", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}

	if peerType == "" {
		peerType = "user"
	}

	if mediaReader != nil {
		var err error
		if isBot {
			err = h.Telegram.SendBotTelegramMedia(r.Context(), device.ID, peerType, peerID, accessHash, mediaReader, mediaName, mtype, caption, replyMarkup)
		} else {
			err = h.Telegram.SendTelegramMedia(r.Context(), device.ID, peerType, peerID, accessHash, mediaReader, mediaName, mtype, caption, replyMarkup)
		}
		if err != nil {
			apiErr := telegramErrorToAPI(err)
			h.Logs.WriteDetail("error", "api.send", fmt.Sprintf("Gagal kirim %s ke chat %s (%s): %s", mtype, chatIDStr, peerType, apiErr.Message), clientIP, reqDataStr, &device.ID)
			jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
			return
		}

		h.Logs.WriteDetail("info", "api.send", fmt.Sprintf("Berhasil kirim %s ke chat %s (%s, file: %s)", mtype, chatIDStr, peerType, mediaName), clientIP, reqDataStr, &device.ID)
		jsonOK(w, map[string]any{"message": fmt.Sprintf("%s sent", mtype)})
		return
	}

	if message == "" {
		h.Logs.WriteDetail("warn", "api.send", fmt.Sprintf("Validasi gagal untuk chat %s: message or media file is required", chatIDStr), clientIP, reqDataStr, &device.ID)
		jsonErrorTyped(w, "message or media file is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}

	if isBot {
		err = h.Telegram.SendBotTelegramMessage(r.Context(), device.ID, peerType, peerID, accessHash, message, replyMarkup)
	} else {
		err = h.Telegram.SendTelegramMessage(r.Context(), device.ID, peerType, peerID, accessHash, message, replyMarkup)
	}
	if err != nil {
		apiErr := telegramErrorToAPI(err)
		h.Logs.WriteDetail("error", "api.send", fmt.Sprintf("Gagal kirim pesan ke chat %s (%s): %s", chatIDStr, peerType, apiErr.Message), clientIP, reqDataStr, &device.ID)
		jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
		return
	}

	h.Logs.WriteDetail("info", "api.send", fmt.Sprintf("Berhasil kirim pesan ke chat %s (%s): %s", chatIDStr, peerType, message), clientIP, reqDataStr, &device.ID)
	jsonOK(w, map[string]any{"message": "Message sent"})
}

// APICallbackHandler menangani klik URL callback dari tombol Telegram dan memicu webhook callback_query ke server Anda
func (h *Handlers) APICallbackHandler(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := r.URL.Query().Get("device")
	dataVal := r.URL.Query().Get("data")
	chatIDStr := r.URL.Query().Get("chat_id")

	deviceIDNum, _ := strconv.ParseUint(deviceIDStr, 10, 64)
	deviceID := uint(deviceIDNum)

	if deviceID == 0 {
		http.Error(w, "Parameter device tidak valid", http.StatusBadRequest)
		return
	}

	device, err := h.Devices.Find(deviceID)
	if err != nil {
		http.Error(w, "Device tidak ditemukan", http.StatusNotFound)
		return
	}

	chatIDNum, _ := strconv.ParseInt(chatIDStr, 10, 64)

	// Buat payload callback query
	now := time.Now()
	payload := &telegram.WebhookCallbackQueryPayload{
		Event:     "callback_query",
		DeviceID:  deviceID,
		IsBot:     true,
		QueryID:   now.UnixNano(),
		Date:      now.Unix(),
		DateTime:  now.Format("2006-01-02 15:04:05"),
		Data:      dataVal,
		Sender: &telegram.WebhookSenderInfo{
			ID: chatIDNum,
		},
		Chat: &telegram.WebhookChatInfo{
			ID:   chatIDNum,
			Type: "user",
		},
	}

	// Dispatch ke webhook Laravel secara asinkron
	if device.WebhookEnabled && strings.TrimSpace(device.WebhookURL) != "" {
		go func() {
			_, _ = telegram.SendWebhookPayload(nil, device.WebhookURL, device.WebhookSecret, deviceID, payload)
		}()
	}

	cbPayloadMap := map[string]any{
		"client_ip": GetClientIP(r),
		"device_id": deviceID,
		"chat_id":   chatIDStr,
		"data":      dataVal,
	}
	cbDataBytes, _ := json.Marshal(cbPayloadMap)
	h.Logs.WriteDetail("info", "webhook.callback", fmt.Sprintf("Callback URL '%s' diklik oleh penerima ID %s", dataVal, chatIDStr), GetClientIP(r), string(cbDataBytes), &deviceID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Aksi Diproses</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background: #0f172a; color: #fff; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; padding: 20px; box-sizing: border-box; text-align: center; }
    .card { background: #1e293b; border-radius: 16px; padding: 36px 24px; max-width: 400px; width: 100%%; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.4); }
    .icon { font-size: 52px; margin-bottom: 16px; }
    h2 { margin: 0 0 10px; font-size: 22px; font-weight: 700; color: #38bdf8; }
    p { margin: 0 0 24px; color: #94a3b8; font-size: 14px; line-height: 1.6; }
    .data-badge { display: inline-block; background: #334155; color: #f1f5f9; padding: 6px 14px; border-radius: 9999px; font-family: monospace; font-size: 13px; margin-bottom: 20px; word-break: break-all; }
    .btn { display: block; width: 100%%; box-sizing: border-box; background: #2563eb; color: #fff; padding: 12px 20px; border-radius: 10px; text-decoration: none; font-weight: 600; font-size: 15px; transition: background 0.2s; }
    .btn:hover { background: #1d4ed8; }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">✅</div>
    <h2>Pilihan Diterima</h2>
    <div class="data-badge">%s</div>
    <p>Aksi Anda telah berhasil dikirim.<br>Anda dapat kembali ke aplikasi Telegram.</p>
    <a href="tg://resolve" class="btn" onclick="window.close()">Kembali ke Telegram</a>
  </div>
  <script>
    setTimeout(function() {
      window.location.href = "tg://resolve";
    }, 1200);
  </script>
</body>
</html>`, dataVal)
	w.Write([]byte(html))
}

// classifyMedia returns "photo", "video", "audio", or "document"
func classifyMedia(ext string) string {
	switch ext {
	case "jpg", "jpeg", "png", "webp", "bmp", "gif":
		return "photo"
	case "mp4", "mov", "avi", "mkv", "webm":
		return "video"
	case "mp3", "ogg", "flac", "wav", "aac", "opus":
		return "audio"
	default:
		return "document"
	}
}

func previewText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

