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

// APISendMessage — POST /api/send
// Header: X-Api-Key: <uuid>
//
// JSON body (text only):
//
//	{ "chat_id": "123", "message": "Hello!", "peer_type": "user", "access_hash": 0 }
//
// JSON body (media via URL):
//
//	{ "chat_id": "123", "peer_type": "user", "access_hash": 0,
//	  "media_url": "https://example.com/photo.jpg", "caption": "look!" }
//
// JSON body (media via base64):
//
//	{ "chat_id": "123", "peer_type": "user", "access_hash": 0,
//	  "media_base64": "<base64>", "media_filename": "photo.jpg", "caption": "look!" }
//
// Multipart form-data:
//
//	fields: chat_id, peer_type, access_hash, caption, message
//	file:   field name "file"
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
