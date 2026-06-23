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
}

func telegramErrorToAPI(err error) apiTelegramError {
	if err == nil {
		return apiTelegramError{Message: "", Type: "terserah_yang_penting_sesuai_dengan_telegram", Detail: nil, HTTPStatus: http.StatusInternalServerError}
	}

	// If telegram service returns a typed error, use it.
	type typed interface {
		APIError() (msg string, errType string, detail map[string]any, httpStatus int)
	}
	if te, ok := err.(typed); ok {
		msg, errType, detail, httpStatus := te.APIError()
		if msg == "" {
			msg = err.Error()
		}
		if errType == "" {
			errType = "terserah_yang_penting_sesuai_dengan_telegram"
		}
		if httpStatus == 0 {
			httpStatus = http.StatusInternalServerError
		}
		return apiTelegramError{Message: msg, Type: errType, Detail: detail, HTTPStatus: httpStatus}
	}

	return apiTelegramError{Message: err.Error(), Type: "terserah_yang_penting_sesuai_dengan_telegram", Detail: nil, HTTPStatus: http.StatusInternalServerError}
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
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiKey == "" {
		jsonErrorTyped(w, "Missing API key", "invalid_api_key", nil, http.StatusUnauthorized)
		return
	}

	device, err := h.Devices.FindByAPIKey(apiKey)
	if err != nil {
		jsonErrorTyped(w, "Invalid API key", "invalid_api_key", nil, http.StatusUnauthorized)
		return
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
	)

	ct := r.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "multipart/form-data") {
		// ── multipart ──────────────────────────────────────────────────────────
		if err := r.ParseMultipartForm(64 << 20); err != nil { // 64 MB
			jsonErrorTyped(w, "Failed to parse multipart form", "bad_request", nil, http.StatusBadRequest)
			return
		}

		chatIDStr = r.FormValue("chat_id")
		peerType = r.FormValue("peer_type")
		caption = r.FormValue("caption")
		message = r.FormValue("message")
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
			Phone         string `json:"phone"`
			ChatID        string `json:"chat_id"`
			Message       string `json:"message"`
			PeerType      string `json:"peer_type"`
			AccessHash    int64  `json:"access_hash"`
			Caption       string `json:"caption"`
			MediaURL      string `json:"media_url"`
			MediaBase64   string `json:"media_base64"`
			MediaFilename string `json:"media_filename"`
		}
		req, err := Bind[Req](r)
		if err != nil {
			jsonErrorTyped(w, "Invalid request body", "bad_request", nil, http.StatusBadRequest)
			return
		}

		chatIDStr = req.ChatID
		phoneTarget = req.Phone
		peerType = req.PeerType
		accessHash = req.AccessHash
		message = req.Message
		caption = req.Caption

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

	// ── dispatch by phone (shortcut, skip peer resolution) ────────────────────
	if strings.HasPrefix(ct, "multipart/form-data") {
		phoneTarget = r.FormValue("phone")
	}

	// ── validate & dispatch ────────────────────────────────────────────────────
	mtype := classifyMedia(strings.ToLower(strings.TrimPrefix(filepath.Ext(mediaName), ".")))

	// PATH A: kirim by phone
	if phoneTarget != "" {
		if mediaReader != nil {
			if err := h.Telegram.SendTelegramMediaByPhone(r.Context(), device.ID, phoneTarget, mediaReader, mediaName, mtype, caption); err != nil {
				apiErr := telegramErrorToAPI(err)
				jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
				return
			}

			jsonOK(w, map[string]any{"message": fmt.Sprintf("%s sent", mtype)})
			return
		}
		if message == "" {
			jsonErrorTyped(w, "message or media file is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
			return
		}
		if err := h.Telegram.SendTelegramMessageByPhone(r.Context(), device.ID, phoneTarget, message); err != nil {
			apiErr := telegramErrorToAPI(err)
			jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
			return
		}

		jsonOK(w, map[string]any{"message": "Message sent"})
		return
	}

	// PATH B: kirim by chat_id (existing flow)
	if chatIDStr == "" {
		jsonErrorTyped(w, "chat_id or phone is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}
	peerID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		jsonErrorTyped(w, "chat_id must be a numeric Telegram ID", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}

	if peerType == "" {
		peerType = "user"
	}

	if mediaReader != nil {
		if err := h.Telegram.SendTelegramMedia(r.Context(), device.ID, peerType, peerID, accessHash, mediaReader, mediaName, mtype, caption); err != nil {
			apiErr := telegramErrorToAPI(err)
			jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
			return
		}

		jsonOK(w, map[string]any{"message": fmt.Sprintf("%s sent", mtype)})
		return
	}

	if message == "" {
		jsonErrorTyped(w, "message or media file is required", "unprocessable_entity", nil, http.StatusUnprocessableEntity)
		return
	}
	if err := h.Telegram.SendTelegramMessage(r.Context(), device.ID, peerType, peerID, accessHash, message); err != nil {
		apiErr := telegramErrorToAPI(err)
		jsonErrorTyped(w, apiErr.Message, apiErr.Type, apiErr.Detail, apiErr.HTTPStatus)
		return
	}

	jsonOK(w, map[string]any{"message": "Message sent"})
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
