package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/tiar/telegram-sender/internal/config"
	"github.com/tiar/telegram-sender/internal/models"
	"github.com/tiar/telegram-sender/internal/services"
	"github.com/tiar/telegram-sender/internal/utils"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type Service struct {
	cfg     config.Config
	db      *gorm.DB
	devices *services.DeviceService
	logs    *services.LogService
	waiter  *floodwait.Waiter
}

func NewService(cfg config.Config, db *gorm.DB, devices *services.DeviceService, logs *services.LogService) *Service {
	return &Service{
		cfg:     cfg,
		db:      db,
		devices: devices,
		logs:    logs,
		waiter:  floodwait.NewWaiter(),
	}
}

func (s *Service) client(deviceID uint) *telegram.Client {
	return telegram.NewClient(s.cfg.AppID, s.cfg.AppHash, telegram.Options{
		SessionStorage: NewDeviceSessionStorage(s.db, deviceID),
		Middlewares: []telegram.Middleware{
			s.waiter,
			ratelimit.New(rate.Every(100*time.Millisecond), 5),
		},
	})
}

func (s *Service) Run(ctx context.Context, deviceID uint, fn func(ctx context.Context, client *telegram.Client, api *tg.Client) error) error {
	client := s.client(deviceID)
	return s.waiter.Run(ctx, func(ctx context.Context) error {
		return client.Run(ctx, func(ctx context.Context) error {
			return fn(ctx, client, client.API())
		})
	})
}

func (s *Service) SendCode(ctx context.Context, deviceID uint, phone string) (string, error) {
	var codeHash string
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_ = api
		sent, err := client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
		if err != nil {
			return err
		}
		sc, ok := sent.(*tg.AuthSentCode)
		if !ok {
			return fmt.Errorf("unexpected sent code type: %T", sent)
		}
		codeHash = sc.PhoneCodeHash
		return nil
	})
	if err != nil {
		id := deviceID
		s.logs.Write("error", "session.send_code", err.Error(), &id)
	}
	return codeHash, err
}

func (s *Service) SignIn(ctx context.Context, deviceID uint, phone, code, codeHash string) (needsPassword bool, err error) {
	err = s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_ = api
		authResult, err := client.Auth().SignIn(ctx, phone, code, codeHash)
		if errors.Is(err, auth.ErrPasswordAuthNeeded) {
			needsPassword = true
			return nil
		}
		if err != nil {
			return err
		}
		_ = authResult
		return s.persistDevice(ctx, deviceID, client)
	})
	if err != nil {
		id := deviceID
		s.logs.Write("error", "session.sign_in", err.Error(), &id)
	}
	return needsPassword, err
}

func (s *Service) SignInPassword(ctx context.Context, deviceID uint, password string) error {
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_ = api
		authResult, err := client.Auth().Password(ctx, password)
		if err != nil {
			return err
		}
		_ = authResult
		return s.persistDevice(ctx, deviceID, client)
	})
	if err != nil {
		id := deviceID
		s.logs.Write("error", "session.password", err.Error(), &id)
	}
	return err
}

func (s *Service) RefreshProfile(ctx context.Context, deviceID uint) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_ = api
		return s.persistDevice(ctx, deviceID, client)
	})
}
func (s *Service) persistDevice(ctx context.Context, deviceID uint, client *telegram.Client) error {
	user, err := client.Self(ctx)
	if err != nil {
		return err
	}

	storage := NewDeviceSessionStorage(s.db, deviceID)
	data, err := storage.LoadSession(ctx)
	if err != nil {
		return err
	}
	var telegramColors = map[int]string{
		0: "#F44336",
		1: "#FF9800",
		2: "#9C27B0",
		3: "#4CAF50",
		4: "#00BCD4",
		5: "#2196F3",
		6: "#E91E63",
	}

	avatarColor := "#1677ff"

	if peerColor, ok := user.GetColor(); ok {
		if c, ok := peerColor.(*tg.PeerColor); ok {
			if color, exists := telegramColors[c.Color]; exists {
				avatarColor = color
			}
		}
	}

	if err := s.devices.UpdateSession(deviceID, data, user.ID, user.FirstName, user.LastName, avatarColor, user.Phone); err != nil {
		return err
	}

	s.logs.Write("info", "session.created", fmt.Sprintf("Session Telegram: %s (%s)", user.FirstName, user.Phone), &deviceID)
	return nil
}

func (s *Service) CheckOnline(ctx context.Context, deviceID uint) (string, error) {
	device, err := s.devices.Find(deviceID)
	if err != nil {
		return "", err
	}
	if !device.HasSession() {
		return models.DeviceStatusNoSession, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	err = s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_, err := client.Self(ctx)
		return err
	})

	status := models.DeviceStatusOnline
	if err != nil {
		status = models.DeviceStatusOffline
	}
	_ = s.devices.UpdateStatus(deviceID, status)
	return status, nil
}

type ChatItem struct {
	ID          int64  `json:"id,string"`
	Type        string `json:"type"` // "user", "chat", "channel"
	Name        string `json:"name"`
	Username    string `json:"username,omitempty"`
	Phone       string `json:"phone,omitempty"`
	LastMsg     string `json:"lastMessage,omitempty"`
	LastMsgTime string `json:"lastMessageTime,omitempty"`
	UnreadCnt   int    `json:"unreadCount"`
	AccessHash  int64  `json:"accessHash,string"`
}

func (s *Service) GetDialogs(ctx context.Context, deviceID uint) ([]ChatItem, error) {
	var items []ChatItem
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		dialogsRes, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			Limit:      100,
			OffsetPeer: &tg.InputPeerEmpty{},
		})
		if err != nil {
			return err
		}

		var dialogs []tg.DialogClass
		var messages []tg.MessageClass
		var chats []tg.ChatClass
		var users []tg.UserClass

		switch dlg := dialogsRes.(type) {
		case *tg.MessagesDialogs:
			dialogs = dlg.Dialogs
			messages = dlg.Messages
			chats = dlg.Chats
			users = dlg.Users
		case *tg.MessagesDialogsSlice:
			dialogs = dlg.Dialogs
			messages = dlg.Messages
			chats = dlg.Chats
			users = dlg.Users
		}

		userMap := make(map[int64]*tg.User)
		for _, uClass := range users {
			if u, ok := uClass.(*tg.User); ok {
				userMap[u.ID] = u
			}
		}

		chatMap := make(map[int64]*tg.Chat)
		channelMap := make(map[int64]*tg.Channel)
		for _, cClass := range chats {
			switch c := cClass.(type) {
			case *tg.Chat:
				chatMap[c.ID] = c
			case *tg.Channel:
				channelMap[c.ID] = c
			}
		}

		messageMap := make(map[int]*tg.Message)
		for _, mClass := range messages {
			if m, ok := mClass.(*tg.Message); ok {
				messageMap[m.ID] = m
			}
		}

		for _, dlgClass := range dialogs {
			dlg, ok := dlgClass.(*tg.Dialog)
			if !ok {
				continue
			}
			var item ChatItem

			switch peer := dlg.Peer.(type) {
			case *tg.PeerUser:
				item.ID = peer.UserID
				item.Type = "user"
				if u, ok := userMap[peer.UserID]; ok {
					item.Name = strings.TrimSpace(u.FirstName + " " + u.LastName)
					if item.Name == "" {
						item.Name = u.Username
					}
					item.Username = u.Username
					item.Phone = u.Phone
					item.AccessHash = u.AccessHash
				}
			case *tg.PeerChat:
				item.ID = peer.ChatID
				item.Type = "chat"
				if c, ok := chatMap[peer.ChatID]; ok {
					item.Name = c.Title
				}
			case *tg.PeerChannel:
				item.ID = peer.ChannelID
				item.Type = "channel"
				if c, ok := channelMap[peer.ChannelID]; ok {
					item.Name = c.Title
					item.Username = c.Username
					item.AccessHash = c.AccessHash
				}
			default:
				continue
			}

			if item.Name == "" {
				item.Name = fmt.Sprintf("Chat %d", item.ID)
			}

			item.UnreadCnt = dlg.UnreadCount

			if msg, ok := messageMap[dlg.TopMessage]; ok {
				item.LastMsg = msg.Message
				item.LastMsgTime = time.Unix(int64(msg.Date), 0).Format("15:04")
			}

			items = append(items, item)
		}

		return nil
	})

	return items, err
}

type MessageItem struct {
	ID         int    `json:"id"`
	SenderID   int64  `json:"senderId,string"`
	SenderName string `json:"senderName"`
	Text       string `json:"text"`
	Out        bool   `json:"out"`
	Time       string `json:"time"`
	MediaType  string `json:"mediaType,omitempty"` // "photo", "video", "document", "sticker",  "voice", "audio"
	MediaID    int64  `json:"mediaId,omitempty,string"`
}

func (s *Service) GetMessages(ctx context.Context, deviceID uint, peerType string, peerID int64, accessHash int64, limit int, offsetID int) ([]MessageItem, error) {
	var items []MessageItem
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		var inputPeer tg.InputPeerClass
		switch peerType {
		case "user":
			inputPeer = &tg.InputPeerUser{UserID: peerID, AccessHash: accessHash}
		case "channel":
			inputPeer = &tg.InputPeerChannel{ChannelID: peerID, AccessHash: accessHash}
		case "chat":
			inputPeer = &tg.InputPeerChat{ChatID: peerID}
		default:
			return fmt.Errorf("invalid peer type: %s", peerType)
		}

		res, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			Limit:    limit,
			OffsetID: offsetID, // 0 = dari pesan terbaru, >0 = ambil pesan sebelum ID ini
		})
		if err != nil {
			return err
		}

		var messages []tg.MessageClass
		var chats []tg.ChatClass
		var users []tg.UserClass

		switch history := res.(type) {
		case *tg.MessagesMessages:
			messages = history.Messages
			chats = history.Chats
			users = history.Users
		case *tg.MessagesMessagesSlice:
			messages = history.Messages
			chats = history.Chats
			users = history.Users
		case *tg.MessagesChannelMessages:
			messages = history.Messages
			chats = history.Chats
			users = history.Users
		}

		userMap := make(map[int64]*tg.User)
		for _, uClass := range users {
			if u, ok := uClass.(*tg.User); ok {
				userMap[u.ID] = u
			}
		}

		chatMap := make(map[int64]*tg.Chat)
		channelMap := make(map[int64]*tg.Channel)
		for _, cClass := range chats {
			switch c := cClass.(type) {
			case *tg.Chat:
				chatMap[c.ID] = c
			case *tg.Channel:
				channelMap[c.ID] = c
			}
		}

		for _, mClass := range messages {
			m, ok := mClass.(*tg.Message)
			if !ok {
				continue
			}

			var item MessageItem
			item.ID = m.ID
			item.Text = m.Message
			item.Out = m.Out
			item.Time = time.Unix(int64(m.Date), 0).Format("15:04")

			if m.Media != nil {
				switch media := m.Media.(type) {
				case *tg.MessageMediaPhoto:
					item.MediaType = "photo"
					if photo, ok := media.Photo.(*tg.Photo); ok {
						item.MediaID = photo.ID
					}
				case *tg.MessageMediaDocument:
					if doc, ok := media.Document.(*tg.Document); ok {
						item.MediaID = doc.ID
						for _, attr := range doc.Attributes {
							switch attr.(type) {
							case *tg.DocumentAttributeVideo:
								item.MediaType = "video"
							case *tg.DocumentAttributeAudio:
								a := attr.(*tg.DocumentAttributeAudio)
								if a.Voice {
									item.MediaType = "voice"
								} else {
									item.MediaType = "audio"
								}
							case *tg.DocumentAttributeSticker:
								item.MediaType = "sticker"
							case *tg.DocumentAttributeAnimated:
								item.MediaType = "gif"
							}
						}
						if item.MediaType == "" {
							item.MediaType = "document"
						}
					}
				}
			}

			if m.FromID != nil {
				switch p := m.FromID.(type) {
				case *tg.PeerUser:
					item.SenderID = p.UserID
					if u, ok := userMap[p.UserID]; ok {
						item.SenderName = strings.TrimSpace(u.FirstName + " " + u.LastName)
						if item.SenderName == "" {
							item.SenderName = u.Username
						}
					}
				case *tg.PeerChat:
					item.SenderID = p.ChatID
					if c, ok := chatMap[p.ChatID]; ok {
						item.SenderName = c.Title
					}
				case *tg.PeerChannel:
					item.SenderID = p.ChannelID
					if c, ok := channelMap[p.ChannelID]; ok {
						item.SenderName = c.Title
					}
				}
			}

			if item.SenderName == "" {
				if item.Out {
					item.SenderName = "Anda"
				} else {
					item.SenderName = "System/Unknown"
				}
			}

			items = append(items, item)
		}

		return nil
	})

	// Reverse: oldest first
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	return items, err
}

func (s *Service) SendTelegramMessage(ctx context.Context, deviceID uint, peerType string, peerID int64, accessHash int64, text string) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		var peer tg.InputPeerClass
		switch peerType {
		case "user":
			peer = &tg.InputPeerUser{
				UserID:     peerID,
				AccessHash: accessHash,
			}
		case "channel":
			peer = &tg.InputPeerChannel{
				ChannelID:  peerID,
				AccessHash: accessHash,
			}
		case "chat":
			peer = &tg.InputPeerChat{
				ChatID: peerID,
			}
		default:
			return fmt.Errorf("invalid peer type: %s", peerType)
		}

		var b [8]byte
		_, _ = rand.Read(b[:])
		randomID := int64(binary.BigEndian.Uint64(b[:]))

		_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:     peer,
			Message:  text,
			RandomID: randomID,
		})
		return err
	})

}

// Tambahkan ke internal/telegram/service.go

func (s *Service) SendTelegramMedia(
	ctx context.Context,
	deviceID uint,
	peerType string,
	peerID int64,
	accessHash int64,
	data io.Reader,
	filename string,
	mtype string,
	caption string,
) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		var peer tg.InputPeerClass
		switch peerType {
		case "user":
			peer = &tg.InputPeerUser{UserID: peerID, AccessHash: accessHash}
		case "channel":
			peer = &tg.InputPeerChannel{ChannelID: peerID, AccessHash: accessHash}
		case "chat":
			peer = &tg.InputPeerChat{ChatID: peerID}
		default:
			return fmt.Errorf("invalid peer type: %s", peerType)
		}

		raw, err := io.ReadAll(data)
		if err != nil {
			return fmt.Errorf("read media: %w", err)
		}

		u := uploader.NewUploader(api)
		upload, err := u.FromBytes(ctx, filename, raw)
		if err != nil {
			return fmt.Errorf("upload media: %w", err)
		}

		var inputMedia tg.InputMediaClass
		switch mtype {
		case "photo":
			inputMedia = &tg.InputMediaUploadedPhoto{
				File: upload,
			}
		case "video":
			inputMedia = &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeVideo{
						SupportsStreaming: true,
					},
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			}
		case "audio":
			inputMedia = &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeAudio{},
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			}
		default: // document
			inputMedia = &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			}
		}

		var b [8]byte
		_, _ = rand.Read(b[:])
		randomID := int64(binary.BigEndian.Uint64(b[:]))

		_, err = api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
			Peer:     peer,
			Media:    inputMedia,
			Message:  caption, // caption
			RandomID: randomID,
		})
		return err
	})
}

// SendTelegramMessageByPhone resolves phone number via contacts.ImportContacts
// lalu kirim pesan. Contact sementara, dihapus setelah send.
func (s *Service) SendTelegramMessageByPhone(ctx context.Context, deviceID uint, phone, text string) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		// 1. normalize: pastikan ada prefix +
		if !strings.HasPrefix(phone, "+") {
			phone = "+" + phone
		}

		// 2. import contact sementara untuk resolve phone → user
		imported, err := api.ContactsImportContacts(ctx, []tg.InputPhoneContact{
			{
				ClientID:  0,
				Phone:     phone,
				FirstName: "tmp",
				LastName:  "",
			},
		})
		if err != nil {
			return fmt.Errorf("resolve phone: %w", err)
		}

		// 3. ambil user dari hasil import
		var target *tg.User
		for _, uClass := range imported.Users {
			if u, ok := uClass.(*tg.User); ok {
				target = u
				break
			}
		}
		if target == nil {
			return fmt.Errorf("nomor %s tidak ditemukan di Telegram", phone)
		}

		text, entities := parseWAFormat(text)

		if err != nil {
			return err
		}
		// 4. kirim pesan
		var b [8]byte
		_, _ = rand.Read(b[:])
		randomID := int64(binary.BigEndian.Uint64(b[:]))

		_, sendErr := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer: &tg.InputPeerUser{
				UserID:     target.ID,
				AccessHash: target.AccessHash,
			},
			Message:  text,
			RandomID: randomID,
			Entities: entities,
		})

		// 5. cleanup: hapus contact sementara (best effort, ignore error)
		_, _ = api.ContactsDeleteContacts(ctx, []tg.InputUserClass{
			&tg.InputUser{
				UserID:     target.ID,
				AccessHash: target.AccessHash,
			},
		})

		return sendErr
	})
}

// ParseWAFormat convert WA-style markdown (*bold*, _italic_, ~strike~, ```mono```)
// jadi plain text + Telegram entities (offset/length dalam UTF-16 code units).
// Catatan: gak support nested marker, dan marker yang gak ketutup bakal hilang (silently stripped).
func parseWAFormat(input string) (string, []tg.MessageEntityClass) {
	runes := []rune(input)
	var out []rune
	var entities []tg.MessageEntityClass

	type openSpan struct {
		sym   rune
		start int
	}
	var stack []openSpan
	pos := 0

	utf16Len := func(r rune) int {
		if r > 0xFFFF {
			return 2
		}
		return 1
	}

	i := 0
	for i < len(runes) {
		// triple backtick block (mono)
		if runes[i] == '`' && i+2 < len(runes) && runes[i+1] == '`' && runes[i+2] == '`' {
			end := -1
			for j := i + 3; j+2 < len(runes); j++ {
				if runes[j] == '`' && runes[j+1] == '`' && runes[j+2] == '`' {
					end = j
					break
				}
			}
			if end != -1 {
				start := pos
				length := 0
				for _, r := range runes[i+3 : end] {
					out = append(out, r)
					length += utf16Len(r)
				}
				entities = append(entities, &tg.MessageEntityPre{Offset: start, Length: length})
				pos += length
				i = end + 3
				continue
			}
		}

		r := runes[i]
		if r == '*' || r == '_' || r == '~' {
			if n := len(stack); n > 0 && stack[n-1].sym == r {
				span := stack[n-1]
				stack = stack[:n-1]
				length := pos - span.start
				var ent tg.MessageEntityClass
				switch r {
				case '*':
					ent = &tg.MessageEntityBold{Offset: span.start, Length: length}
				case '_':
					ent = &tg.MessageEntityItalic{Offset: span.start, Length: length}
				case '~':
					ent = &tg.MessageEntityStrike{Offset: span.start, Length: length}
				}
				entities = append(entities, ent)
				i++
				continue
			}
			stack = append(stack, openSpan{sym: r, start: pos})
			i++
			continue
		}

		out = append(out, r)
		pos += utf16Len(r)
		i++
	}

	return string(out), entities
}

// SendTelegramMediaByPhone — sama tapi kirim media
func (s *Service) SendTelegramMediaByPhone(ctx context.Context, deviceID uint, phone string, data io.Reader, filename, mtype, caption string) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		if !strings.HasPrefix(phone, "+") {
			phone = "+" + phone
		}

		imported, err := api.ContactsImportContacts(ctx, []tg.InputPhoneContact{
			{ClientID: 0, Phone: phone, FirstName: "tmp"},
		})
		if err != nil {
			return fmt.Errorf("resolve phone: %w", err)
		}

		var target *tg.User
		for _, uClass := range imported.Users {
			if u, ok := uClass.(*tg.User); ok {
				target = u
				break
			}
		}
		if target == nil {
			return fmt.Errorf("nomor %s tidak ditemukan di Telegram", phone)
		}

		raw, err := io.ReadAll(data)
		if err != nil {
			return fmt.Errorf("read media: %w", err)
		}

		u := uploader.NewUploader(api)
		upload, err := u.FromBytes(ctx, filename, raw)
		if err != nil {
			return fmt.Errorf("upload media: %w", err)
		}

		var inputMedia tg.InputMediaClass
		switch mtype {
		case "photo":
			inputMedia = &tg.InputMediaUploadedPhoto{File: upload}
		case "video":
			inputMedia = &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeVideo{SupportsStreaming: true},
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			}
		case "audio":
			inputMedia = &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeAudio{},
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			}
		default:
			inputMedia = &tg.InputMediaUploadedDocument{
				File:       upload,
				MimeType:   utils.MimeByExt(filename),
				Attributes: []tg.DocumentAttributeClass{&tg.DocumentAttributeFilename{FileName: filename}},
			}
		}

		var b [8]byte
		_, _ = rand.Read(b[:])
		randomID := int64(binary.BigEndian.Uint64(b[:]))

		_, sendErr := api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
			Peer:     &tg.InputPeerUser{UserID: target.ID, AccessHash: target.AccessHash},
			Media:    inputMedia,
			Message:  caption,
			RandomID: randomID,
		})

		_, _ = api.ContactsDeleteContacts(ctx, []tg.InputUserClass{
			&tg.InputUser{UserID: target.ID, AccessHash: target.AccessHash},
		})

		return sendErr
	})
}

func (s *Service) DownloadMedia(ctx context.Context, deviceID uint, peerType string, peerID int64, accessHash int64, msgID int, w io.Writer) error {
	// Buat client baru tapi bypass waiter dengan SimpleWaiter
	// karena s.waiter.Run() lifecycle tidak compatible dengan on-demand HTTP handler
	client := telegram.NewClient(s.cfg.AppID, s.cfg.AppHash, telegram.Options{
		SessionStorage: NewDeviceSessionStorage(s.db, deviceID),
		Middlewares: []telegram.Middleware{
			ratelimit.New(rate.Every(100*time.Millisecond), 5),
			// tidak pakai s.waiter di sini
		},
	})

	return client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		var inputPeer tg.InputPeerClass
		switch peerType {
		case "user":
			inputPeer = &tg.InputPeerUser{UserID: peerID, AccessHash: accessHash}
		case "channel":
			inputPeer = &tg.InputPeerChannel{ChannelID: peerID, AccessHash: accessHash}
		case "chat":
			inputPeer = &tg.InputPeerChat{ChatID: peerID}
		default:
			return fmt.Errorf("invalid peer type")
		}

		// Untuk channel, MinID/MaxID tidak reliable — pakai OffsetID saja
		res, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			OffsetID: msgID + 1, // ambil pesan sebelum msgID+1, jadi include msgID
			Limit:    1,
		})
		if err != nil {
			return err
		}

		var messages []tg.MessageClass
		switch r := res.(type) {
		case *tg.MessagesMessages:
			messages = r.Messages
		case *tg.MessagesMessagesSlice:
			messages = r.Messages
		case *tg.MessagesChannelMessages:
			messages = r.Messages
		}

		for _, mClass := range messages {
			m, ok := mClass.(*tg.Message)
			if !ok || m.Media == nil {
				continue
			}
			// cek ID — untuk channel kadang ID match, untuk chat juga
			if m.ID != msgID {
				continue
			}

			var location tg.InputFileLocationClass

			switch media := m.Media.(type) {
			case *tg.MessageMediaPhoto:
				photo, ok := media.Photo.(*tg.Photo)
				if !ok {
					return fmt.Errorf("invalid photo")
				}
				var biggest *tg.PhotoSize
				for _, sClass := range photo.Sizes {
					if sz, ok := sClass.(*tg.PhotoSize); ok {
						if biggest == nil || sz.Size > biggest.Size {
							biggest = sz
						}
					}
				}
				if biggest == nil {
					return fmt.Errorf("no photo size found")
				}
				location = &tg.InputPhotoFileLocation{
					ID:            photo.ID,
					AccessHash:    photo.AccessHash,
					FileReference: photo.FileReference,
					ThumbSize:     biggest.Type,
				}

			case *tg.MessageMediaDocument:
				doc, ok := media.Document.(*tg.Document)
				if !ok {
					return fmt.Errorf("invalid document")
				}
				// GIF, video, sticker, audio — semua pakai InputDocumentFileLocation
				location = &tg.InputDocumentFileLocation{
					ID:            doc.ID,
					AccessHash:    doc.AccessHash,
					FileReference: doc.FileReference,
				}

			default:
				return fmt.Errorf("unsupported media type: %T", m.Media)
			}

			_, err = client.Download(location).Stream(ctx, w)
			return err
		}

		return fmt.Errorf("message %d not found", msgID)
	})
}
