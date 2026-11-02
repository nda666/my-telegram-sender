package telegram

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

type WebhookSenderInfo struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Username  string `json:"username,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

type WebhookChatInfo struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"` // "user", "chat", "channel"
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type WebhookMediaInfo struct {
	Type     string `json:"type"` // "photo", "video", "audio", "voice", "document", "sticker", "gif"
	ID       int64  `json:"id"`
	MimeType string `json:"mimeType,omitempty"`
	FileName string `json:"fileName,omitempty"`
}

type WebhookMessagePayload struct {
	Event     string             `json:"event"`
	DeviceID  uint               `json:"deviceId"`
	IsBot     bool               `json:"isBot"`
	MessageID int                `json:"messageId"`
	Date      int64              `json:"date"`
	DateTime  string             `json:"dateTime"`
	Text      string             `json:"text"`
	Out       bool               `json:"out"`
	Sender    *WebhookSenderInfo `json:"sender,omitempty"`
	Chat      *WebhookChatInfo   `json:"chat,omitempty"`
	Media     *WebhookMediaInfo  `json:"media,omitempty"`
}

type WebhookCallbackQueryPayload struct {
	Event     string             `json:"event"`
	DeviceID  uint               `json:"deviceId"`
	IsBot     bool               `json:"isBot"`
	QueryID   int64              `json:"queryId"`
	MessageID int                `json:"messageId,omitempty"`
	Date      int64              `json:"date"`
	DateTime  string             `json:"dateTime"`
	Data      string             `json:"data"`
	Sender    *WebhookSenderInfo `json:"sender,omitempty"`
	Chat      *WebhookChatInfo   `json:"chat,omitempty"`
}

type WebhookTestResult struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"statusCode"`
	DurationMs   int64  `json:"durationMs"`
	ResponseBody string `json:"responseBody,omitempty"`
	Error        string `json:"error,omitempty"`
}

func ParseMessageToPayload(deviceID uint, e tg.Entities, msg *tg.Message) (*WebhookMessagePayload, int64) {
	payload := &WebhookMessagePayload{
		Event:     "message.received",
		DeviceID:  deviceID,
		MessageID: msg.ID,
		Date:      int64(msg.Date),
		DateTime:  time.Unix(int64(msg.Date), 0).Format("2006-01-02 15:04:05"),
		Text:      msg.Message,
		Out:       msg.Out,
	}

	var accessHash int64

	// Chat / Peer Info
	if msg.PeerID != nil {
		switch p := msg.PeerID.(type) {
		case *tg.PeerUser:
			chat := &WebhookChatInfo{ID: p.UserID, Type: "user"}
			if u, ok := e.Users[p.UserID]; ok {
				accessHash = u.AccessHash
				chat.Title = strings.TrimSpace(u.FirstName + " " + u.LastName)
				if chat.Title == "" {
					chat.Title = u.Username
				}
				chat.Username = u.Username
			}
			payload.Chat = chat
		case *tg.PeerChat:
			chat := &WebhookChatInfo{ID: p.ChatID, Type: "chat"}
			if c, ok := e.Chats[p.ChatID]; ok {
				chat.Title = c.Title
			}
			payload.Chat = chat
		case *tg.PeerChannel:
			chat := &WebhookChatInfo{ID: p.ChannelID, Type: "channel"}
			if c, ok := e.Channels[p.ChannelID]; ok {
				chat.Title = c.Title
				chat.Username = c.Username
			}
			payload.Chat = chat
		}
	}

	// Sender Info
	if msg.FromID != nil {
		switch p := msg.FromID.(type) {
		case *tg.PeerUser:
			sender := &WebhookSenderInfo{ID: p.UserID}
			if u, ok := e.Users[p.UserID]; ok {
				if accessHash == 0 {
					accessHash = u.AccessHash
				}
				sender.FirstName = u.FirstName
				sender.LastName = u.LastName
				sender.Username = u.Username
				if u.Phone != "" {
					sender.Phone = u.Phone
					if !strings.HasPrefix(sender.Phone, "+") {
						sender.Phone = "+" + sender.Phone
					}
				}
			}
			payload.Sender = sender
		}
	} else if payload.Chat != nil && payload.Chat.Type == "user" {
		sender := &WebhookSenderInfo{ID: payload.Chat.ID}
		if u, ok := e.Users[payload.Chat.ID]; ok {
			if accessHash == 0 {
				accessHash = u.AccessHash
			}
			sender.FirstName = u.FirstName
			sender.LastName = u.LastName
			sender.Username = u.Username
			if u.Phone != "" {
				sender.Phone = u.Phone
				if !strings.HasPrefix(sender.Phone, "+") {
					sender.Phone = "+" + sender.Phone
				}
			}
		}
		payload.Sender = sender
	}

	// Media Info
	if msg.Media != nil {
		switch media := msg.Media.(type) {
		case *tg.MessageMediaPhoto:
			if photo, ok := media.Photo.(*tg.Photo); ok {
				payload.Media = &WebhookMediaInfo{
					Type: "photo",
					ID:   photo.ID,
				}
			}
		case *tg.MessageMediaDocument:
			if doc, ok := media.Document.(*tg.Document); ok {
				mType := "document"
				var fileName string
				for _, attr := range doc.Attributes {
					switch a := attr.(type) {
					case *tg.DocumentAttributeVideo:
						mType = "video"
					case *tg.DocumentAttributeAudio:
						if a.Voice {
							mType = "voice"
						} else {
							mType = "audio"
						}
					case *tg.DocumentAttributeSticker:
						mType = "sticker"
					case *tg.DocumentAttributeAnimated:
						mType = "gif"
					case *tg.DocumentAttributeFilename:
						fileName = a.FileName
					}
				}
				payload.Media = &WebhookMediaInfo{
					Type:     mType,
					ID:       doc.ID,
					MimeType: doc.MimeType,
					FileName: fileName,
				}
			}
		case *tg.MessageMediaContact:
			phone := media.PhoneNumber
			if !strings.HasPrefix(phone, "+") {
				phone = "+" + phone
			}
			contactName := strings.TrimSpace(media.FirstName + " " + media.LastName)
			if contactName == "" {
				contactName = phone
			}
			payload.Media = &WebhookMediaInfo{
				Type:     "contact",
				ID:       media.UserID,
				FileName: contactName,
			}
			if payload.Sender != nil && (payload.Sender.Phone == "" || payload.Sender.ID == media.UserID) {
				payload.Sender.Phone = phone
			}
		}
	}

	return payload, accessHash
}

func applyUser(payload *WebhookMessagePayload, u *tg.User) {
	if u == nil || payload.Sender == nil {
		return
	}
	if payload.Sender.FirstName == "" && u.FirstName != "" {
		payload.Sender.FirstName = u.FirstName
	}
	if payload.Sender.LastName == "" && u.LastName != "" {
		payload.Sender.LastName = u.LastName
	}
	if payload.Sender.Username == "" && u.Username != "" {
		payload.Sender.Username = u.Username
	}
	if u.Phone != "" {
		phone := u.Phone
		if !strings.HasPrefix(phone, "+") {
			phone = "+" + phone
		}
		payload.Sender.Phone = phone
	}
}

func SaveSenderContact(db *gorm.DB, deviceID uint, tgUserID int64, phone, firstName, lastName, username string) {
	if db == nil || deviceID == 0 || tgUserID == 0 || phone == "" {
		return
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}

	var contact models.DeviceContact
	err := db.Where("device_id = ? AND telegram_user_id = ?", deviceID, tgUserID).First(&contact).Error
	if err == nil {
		updates := map[string]any{"phone": phone}
		if firstName != "" {
			updates["first_name"] = firstName
		}
		if lastName != "" {
			updates["last_name"] = lastName
		}
		if username != "" {
			updates["username"] = username
		}
		_ = db.Model(&contact).Updates(updates).Error
		return
	}

	contact = models.DeviceContact{
		DeviceID:       deviceID,
		TelegramUserID: tgUserID,
		Phone:          phone,
		FirstName:      firstName,
		LastName:       lastName,
		Username:       username,
	}
	_ = db.Create(&contact).Error
}

func FindSenderContactPhone(db *gorm.DB, deviceID uint, tgUserID int64) (string, string, string, string) {
	if db == nil || tgUserID == 0 {
		return "", "", "", ""
	}
	var contact models.DeviceContact
	if deviceID > 0 {
		if err := db.Where("device_id = ? AND telegram_user_id = ?", deviceID, tgUserID).First(&contact).Error; err == nil && contact.Phone != "" {
			return contact.Phone, contact.FirstName, contact.LastName, contact.Username
		}
	}
	// Fallback across any device if this user was recorded previously
	var anyContact models.DeviceContact
	if err := db.Where("telegram_user_id = ? AND phone != ''", tgUserID).Order("id DESC").First(&anyContact).Error; err == nil && anyContact.Phone != "" {
		return anyContact.Phone, anyContact.FirstName, anyContact.LastName, anyContact.Username
	}
	return "", "", "", ""
}

// EnrichSenderInfo melengkapi nomor HP, Nama, dan Username pengirim dari DB lokal dan Telegram MTProto API jika belum lengkap
func EnrichSenderInfo(ctx context.Context, db *gorm.DB, api *tg.Client, payload *WebhookMessagePayload, accessHash int64) {
	if payload == nil || payload.Sender == nil || payload.Sender.ID == 0 {
		return
	}

	senderID := payload.Sender.ID

	// 1. Jika nomor telepon sudah ada pada payload, simpan / update ke database contact
	if payload.Sender.Phone != "" {
		SaveSenderContact(db, payload.DeviceID, senderID, payload.Sender.Phone, payload.Sender.FirstName, payload.Sender.LastName, payload.Sender.Username)
	} else {
		// 2. Jika nomor telepon belum ada, cari dari cache database DeviceContact
		if cachedPhone, cFn, cLn, cUn := FindSenderContactPhone(db, payload.DeviceID, senderID); cachedPhone != "" {
			payload.Sender.Phone = cachedPhone
			if payload.Sender.FirstName == "" && cFn != "" {
				payload.Sender.FirstName = cFn
			}
			if payload.Sender.LastName == "" && cLn != "" {
				payload.Sender.LastName = cLn
			}
			if payload.Sender.Username == "" && cUn != "" {
				payload.Sender.Username = cUn
			}
		}
	}

	if api != nil {
		// 3. Coba ambil dari MessagesGetMessages (membawa entitas user terkait message tersebut)
		if payload.MessageID > 0 {
			res, err := api.MessagesGetMessages(ctx, []tg.InputMessageClass{
				&tg.InputMessageID{ID: payload.MessageID},
			})
			if err == nil {
				var users []tg.UserClass
				switch m := res.(type) {
				case *tg.MessagesMessages:
					users = m.Users
				case *tg.MessagesMessagesSlice:
					users = m.Users
				case *tg.MessagesChannelMessages:
					users = m.Users
				}
				for _, uClass := range users {
					if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
						if accessHash == 0 {
							accessHash = user.AccessHash
						}
						applyUser(payload, user)
						break
					}
				}
			}
		}

		// 4. Jika phone masih kosong atau accessHash belum didapat, coba dari Contacts
		if payload.Sender.Phone == "" || accessHash == 0 {
			contacts, err := api.ContactsGetContacts(ctx, 0)
			if err == nil {
				if c, ok := contacts.(*tg.ContactsContacts); ok {
					for _, uClass := range c.Users {
						if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
							if accessHash == 0 {
								accessHash = user.AccessHash
							}
							applyUser(payload, user)
							break
						}
					}
				}
			}
		}

		// 5. Jika accessHash ada tapi phone masih kosong, coba UsersGetFullUser & UsersGetUsers
		if accessHash != 0 {
			inputUser := &tg.InputUser{UserID: senderID, AccessHash: accessHash}
			if payload.Sender.Phone == "" {
				full, err := api.UsersGetFullUser(ctx, inputUser)
				if err == nil {
					for _, uClass := range full.Users {
						if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
							applyUser(payload, user)
							break
						}
					}
				}
			}

			if payload.Sender.FirstName == "" || payload.Sender.Username == "" {
				users, err := api.UsersGetUsers(ctx, []tg.InputUserClass{inputUser})
				if err == nil {
					for _, uClass := range users {
						if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
							applyUser(payload, user)
							break
						}
					}
				}
			}
		}
	}

	// Simpan ke DB jika phone berhasil didapatkan dari MTProto API
	if payload.Sender.Phone != "" {
		SaveSenderContact(db, payload.DeviceID, senderID, payload.Sender.Phone, payload.Sender.FirstName, payload.Sender.LastName, payload.Sender.Username)
	}

	// Sinkronisasi data chat jika chat bertipe "user"
	if payload.Chat != nil && payload.Chat.Type == "user" {
		if payload.Chat.Title == "" {
			name := strings.TrimSpace(payload.Sender.FirstName + " " + payload.Sender.LastName)
			if name == "" {
				name = payload.Sender.Username
			}
			if name == "" {
				name = payload.Sender.Phone
			}
			payload.Chat.Title = name
		}
		if payload.Chat.Username == "" {
			payload.Chat.Username = payload.Sender.Username
		}
	}
}

func ParseCallbackQueryToPayload(deviceID uint, e tg.Entities, u *tg.UpdateBotCallbackQuery) (*WebhookCallbackQueryPayload, int64) {
	now := time.Now()
	payload := &WebhookCallbackQueryPayload{
		Event:     "callback_query",
		DeviceID:  deviceID,
		QueryID:   u.QueryID,
		MessageID: u.MsgID,
		Date:      now.Unix(),
		DateTime:  now.Format("2006-01-02 15:04:05"),
		Data:      string(u.Data),
	}

	var accessHash int64

	// Chat / Peer Info
	if u.Peer != nil {
		switch p := u.Peer.(type) {
		case *tg.PeerUser:
			chat := &WebhookChatInfo{ID: p.UserID, Type: "user"}
			if user, ok := e.Users[p.UserID]; ok {
				accessHash = user.AccessHash
				chat.Title = strings.TrimSpace(user.FirstName + " " + user.LastName)
				if chat.Title == "" {
					chat.Title = user.Username
				}
				chat.Username = user.Username
			}
			payload.Chat = chat
		case *tg.PeerChat:
			chat := &WebhookChatInfo{ID: p.ChatID, Type: "chat"}
			if c, ok := e.Chats[p.ChatID]; ok {
				chat.Title = c.Title
			}
			payload.Chat = chat
		case *tg.PeerChannel:
			chat := &WebhookChatInfo{ID: p.ChannelID, Type: "channel"}
			if c, ok := e.Channels[p.ChannelID]; ok {
				chat.Title = c.Title
				chat.Username = c.Username
			}
			payload.Chat = chat
		}
	}

	// Sender Info
	sender := &WebhookSenderInfo{ID: u.UserID}
	if user, ok := e.Users[u.UserID]; ok {
		if accessHash == 0 {
			accessHash = user.AccessHash
		}
		sender.FirstName = user.FirstName
		sender.LastName = user.LastName
		sender.Username = user.Username
		if user.Phone != "" {
			phone := user.Phone
			if !strings.HasPrefix(phone, "+") {
				phone = "+" + phone
			}
			sender.Phone = phone
		}
	}
	payload.Sender = sender

	return payload, accessHash
}

func EnrichCallbackSenderInfo(ctx context.Context, db *gorm.DB, api *tg.Client, payload *WebhookCallbackQueryPayload, accessHash int64) {
	if payload == nil || payload.Sender == nil || payload.Sender.ID == 0 {
		return
	}

	senderID := payload.Sender.ID

	// 1. Jika nomor telepon sudah ada pada payload, simpan / update ke database contact
	if payload.Sender.Phone != "" {
		SaveSenderContact(db, payload.DeviceID, senderID, payload.Sender.Phone, payload.Sender.FirstName, payload.Sender.LastName, payload.Sender.Username)
	} else {
		// 2. Jika nomor telepon belum ada, cari dari cache database DeviceContact
		if cachedPhone, cFn, cLn, cUn := FindSenderContactPhone(db, payload.DeviceID, senderID); cachedPhone != "" {
			payload.Sender.Phone = cachedPhone
			if payload.Sender.FirstName == "" && cFn != "" {
				payload.Sender.FirstName = cFn
			}
			if payload.Sender.LastName == "" && cLn != "" {
				payload.Sender.LastName = cLn
			}
			if payload.Sender.Username == "" && cUn != "" {
				payload.Sender.Username = cUn
			}
		}
	}

	if api != nil {
		// 3. Coba ambil dari MessagesGetMessages
		if payload.MessageID > 0 {
			res, err := api.MessagesGetMessages(ctx, []tg.InputMessageClass{
				&tg.InputMessageID{ID: payload.MessageID},
			})
			if err == nil {
				var users []tg.UserClass
				switch m := res.(type) {
				case *tg.MessagesMessages:
					users = m.Users
				case *tg.MessagesMessagesSlice:
					users = m.Users
				case *tg.MessagesChannelMessages:
					users = m.Users
				}
				for _, uClass := range users {
					if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
						if accessHash == 0 {
							accessHash = user.AccessHash
						}
						if payload.Sender.FirstName == "" && user.FirstName != "" {
							payload.Sender.FirstName = user.FirstName
						}
						if payload.Sender.LastName == "" && user.LastName != "" {
							payload.Sender.LastName = user.LastName
						}
						if payload.Sender.Username == "" && user.Username != "" {
							payload.Sender.Username = user.Username
						}
						if user.Phone != "" {
							phone := user.Phone
							if !strings.HasPrefix(phone, "+") {
								phone = "+" + phone
							}
							payload.Sender.Phone = phone
						}
						break
					}
				}
			}
		}

		// 4. Contacts & UsersGetFullUser
		if payload.Sender.Phone == "" || accessHash == 0 {
			contacts, err := api.ContactsGetContacts(ctx, 0)
			if err == nil {
				if c, ok := contacts.(*tg.ContactsContacts); ok {
					for _, uClass := range c.Users {
						if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
							if accessHash == 0 {
								accessHash = user.AccessHash
							}
							if payload.Sender.FirstName == "" && user.FirstName != "" {
								payload.Sender.FirstName = user.FirstName
							}
							if payload.Sender.LastName == "" && user.LastName != "" {
								payload.Sender.LastName = user.LastName
							}
							if payload.Sender.Username == "" && user.Username != "" {
								payload.Sender.Username = user.Username
							}
							if user.Phone != "" {
								phone := user.Phone
								if !strings.HasPrefix(phone, "+") {
									phone = "+" + phone
								}
								payload.Sender.Phone = phone
							}
							break
						}
					}
				}
			}
		}

		if accessHash != 0 && payload.Sender.Phone == "" {
			inputUser := &tg.InputUser{UserID: senderID, AccessHash: accessHash}
			full, err := api.UsersGetFullUser(ctx, inputUser)
			if err == nil {
				for _, uClass := range full.Users {
					if user, ok := uClass.(*tg.User); ok && user.ID == senderID {
						if user.Phone != "" {
							phone := user.Phone
							if !strings.HasPrefix(phone, "+") {
								phone = "+" + phone
							}
							payload.Sender.Phone = phone
						}
						break
					}
				}
			}
		}
	}

	// Simpan ke DB jika phone berhasil didapatkan
	if payload.Sender.Phone != "" {
		SaveSenderContact(db, payload.DeviceID, senderID, payload.Sender.Phone, payload.Sender.FirstName, payload.Sender.LastName, payload.Sender.Username)
	}

	// Sinkronisasi data chat jika chat bertipe "user"
	if payload.Chat != nil && payload.Chat.Type == "user" {
		if payload.Chat.Title == "" {
			name := strings.TrimSpace(payload.Sender.FirstName + " " + payload.Sender.LastName)
			if name == "" {
				name = payload.Sender.Username
			}
			if name == "" {
				name = payload.Sender.Phone
			}
			payload.Chat.Title = name
		}
		if payload.Chat.Username == "" {
			payload.Chat.Username = payload.Sender.Username
		}
	}
}

func SendWebhookPayload(client *http.Client, webhookURL, webhookSecret string, deviceID uint, payload any) (*WebhookTestResult, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Telegram-Sender-Webhook/1.0")
	req.Header.Set("X-Telegram-Sender-Device-Id", strconv.Itoa(int(deviceID)))

	// Extract event name
	if p, ok := payload.(*WebhookMessagePayload); ok {
		req.Header.Set("X-Telegram-Sender-Event", p.Event)
	} else if p, ok := payload.(*WebhookCallbackQueryPayload); ok {
		req.Header.Set("X-Telegram-Sender-Event", p.Event)
	} else if pMap, ok := payload.(map[string]any); ok {
		if ev, ok := pMap["event"].(string); ok {
			req.Header.Set("X-Telegram-Sender-Event", ev)
		}
	}

	if webhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(bodyBytes)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Telegram-Sender-Signature", "sha256="+signature)
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		return &WebhookTestResult{
			Success:    false,
			StatusCode: 0,
			DurationMs: duration,
			Error:      err.Error(),
		}, err
	}
	defer resp.Body.Close()

	respBodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	respBodyStr := string(respBodyBytes)

	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300
	result := &WebhookTestResult{
		Success:      isSuccess,
		StatusCode:   resp.StatusCode,
		DurationMs:   duration,
		ResponseBody: respBodyStr,
	}

	if !isSuccess {
		result.Error = fmt.Sprintf("HTTP server mengembalikan status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return result, nil
}
