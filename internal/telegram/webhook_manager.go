package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/tiar/telegram-sender/internal/config"
	"github.com/tiar/telegram-sender/internal/services"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type WebhookManager struct {
	cfg        config.Config
	db         *gorm.DB
	devices    *services.DeviceService
	logs       *services.LogService
	httpClient *http.Client
	workers    map[uint]context.CancelFunc
	mu         sync.Mutex
	rootCtx    context.Context
}

func NewWebhookManager(cfg config.Config, db *gorm.DB, devices *services.DeviceService, logs *services.LogService) *WebhookManager {
	return &WebhookManager{
		cfg:        cfg,
		db:         db,
		devices:    devices,
		logs:       logs,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		workers:    make(map[uint]context.CancelFunc),
	}
}

func (m *WebhookManager) StartAll(ctx context.Context) {
	m.mu.Lock()
	m.rootCtx = ctx
	m.mu.Unlock()

	devices, err := m.devices.ListActiveWebhookDevices()
	fmt.Println("Jumlah active webhook device", len(devices))
	if err != nil {
		m.logs.Write("error", "webhook.startup", "Gagal memuat list device webhook: "+err.Error(), nil)
		return
	}

	for _, d := range devices {
		m.RestartDevice(d.ID)
	}
}

func (m *WebhookManager) RestartDevice(deviceID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing worker if running
	if cancel, exists := m.workers[deviceID]; exists {
		cancel()
		delete(m.workers, deviceID)
	}

	if m.rootCtx == nil {
		m.rootCtx = context.Background()
	}

	device, err := m.devices.Find(deviceID)
	if err != nil || !device.WebhookEnabled || strings.TrimSpace(device.WebhookURL) == "" {
		return
	}

	hasUser := device.HasSession()
	hasBot := device.HasBot() && device.BotEnabled

	if !hasUser && !hasBot {
		return
	}

	ctx, cancel := context.WithCancel(m.rootCtx)
	m.workers[deviceID] = cancel

	if hasUser {
		go m.runUserDeviceWorker(ctx, deviceID)
	}
	if hasBot {
		go m.runBotDeviceWorker(ctx, deviceID)
	}
}

func (m *WebhookManager) StopDevice(deviceID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, exists := m.workers[deviceID]; exists {
		cancel()
		delete(m.workers, deviceID)
	}
}

func (m *WebhookManager) runUserDeviceWorker(ctx context.Context, deviceID uint) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		device, err := m.devices.Find(deviceID)
		if err != nil || !device.HasSession() || !device.WebhookEnabled || strings.TrimSpace(device.WebhookURL) == "" {
			return
		}

		var client *telegram.Client
		dispatcher := tg.NewUpdateDispatcher()
		dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
			msg, ok := u.Message.(*tg.Message)
			if !ok || msg.Out {
				return nil
			}
			payload, accessHash := ParseMessageToPayload(deviceID, e, msg)
			payload.IsBot = false
			if client != nil {
				EnrichSenderInfo(ctx, m.db, client.API(), payload, accessHash)
			}

			senderDesc := ""
			if payload.Sender != nil {
				if payload.Sender.FirstName != "" {
					senderDesc = payload.Sender.FirstName
				}
				if payload.Sender.Phone != "" {
					if senderDesc != "" {
						senderDesc += " (" + payload.Sender.Phone + ")"
					} else {
						senderDesc = payload.Sender.Phone
					}
				}
			}
			if senderDesc == "" && payload.Chat != nil {
				senderDesc = fmt.Sprintf("ID %d", payload.Chat.ID)
			}

			m.logs.Write("info", "webhook.received", fmt.Sprintf("Pesan masuk #%d diterima dari %s, meneruskan ke webhook...", msg.ID, senderDesc), &deviceID)
			go m.dispatch(deviceID, payload, fmt.Sprintf("pesan #%d", msg.ID))
			return nil
		})

		dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
			msg, ok := u.Message.(*tg.Message)
			if !ok || msg.Out {
				return nil
			}
			payload, accessHash := ParseMessageToPayload(deviceID, e, msg)
			payload.IsBot = false
			if client != nil {
				EnrichSenderInfo(ctx, m.db, client.API(), payload, accessHash)
			}
			m.logs.Write("info", "webhook.received", fmt.Sprintf("Pesan channel masuk #%d diterima, meneruskan ke webhook...", msg.ID), &deviceID)
			go m.dispatch(deviceID, payload, fmt.Sprintf("pesan channel #%d", msg.ID))
			return nil
		})

		gaps := updates.New(updates.Config{
			Handler: dispatcher,
		})

		client = telegram.NewClient(m.cfg.AppID, m.cfg.AppHash, telegram.Options{
			SessionStorage: NewDeviceSessionStorage(m.db, deviceID),
			UpdateHandler:  gaps,
			Middlewares: []telegram.Middleware{
				ratelimit.New(rate.Every(100*time.Millisecond), 5),
			},
		})

		err = client.Run(ctx, func(ctx context.Context) error {
			self, err := client.Self(ctx)
			if err != nil {
				return fmt.Errorf("failed to get self: %w", err)
			}

			return gaps.Run(ctx, client.API(), self.ID, updates.AuthOptions{
				OnStart: func(ctx context.Context) {
					m.logs.Write("info", "webhook.connected", fmt.Sprintf("Listener webhook Userbot aktif untuk device %s (%s)", device.Name, device.Phone), &deviceID)
				},
			})
		})

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			m.logs.Write("warning", "webhook.reconnecting", fmt.Sprintf("Listener Userbot device %s terputus (%v), mencoba menghubungkan kembali...", device.Name, err), &deviceID)
		}

		// Backoff sebelum auto reconnect
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (m *WebhookManager) runBotDeviceWorker(ctx context.Context, deviceID uint) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		device, err := m.devices.Find(deviceID)
		if err != nil || !device.HasBot() || !device.BotEnabled || !device.WebhookEnabled || strings.TrimSpace(device.WebhookURL) == "" {
			return
		}

		var client *telegram.Client
		dispatcher := tg.NewUpdateDispatcher()

		dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
			msg, ok := u.Message.(*tg.Message)
			if !ok || msg.Out {
				return nil
			}
			payload, accessHash := ParseMessageToPayload(deviceID, e, msg)
			payload.IsBot = true
			if client != nil {
				EnrichSenderInfo(ctx, m.db, client.API(), payload, accessHash)
			}

			// Otomatis Request Phone Number saat /start (hanya jika diaktifkan & nomor belum ada di kontak)
			if device.BotRequestPhoneOnStart {
				if peerUser, isUser := msg.PeerID.(*tg.PeerUser); isUser {
					// 1. User kirim /start -> Kirim prompt tombol Request Phone HANYA JIKA nomor HP belum diketahui
					if strings.HasPrefix(strings.TrimSpace(msg.Message), "/start") {
						hasPhone := false
						if payload.Sender != nil && strings.TrimSpace(payload.Sender.Phone) != "" {
							hasPhone = true
						} else {
							existingPhone, _, _, _ := FindSenderContactPhone(m.db, deviceID, peerUser.UserID)
							if strings.TrimSpace(existingPhone) != "" {
								hasPhone = true
							}
						}

						if !hasPhone {
							go func(uID int64, aHash int64) {
								startMsg := strings.TrimSpace(device.BotStartMessage)
								if startMsg == "" {
									startMsg = "👋 Halo! Selamat datang. Silakan tekan tombol di bawah ini untuk membagikan nomor telepon Anda:"
								}
								btnText := strings.TrimSpace(device.BotPhoneButtonText)
								if btnText == "" {
									btnText = "📱 Bagikan Nomor Telepon"
								}

								var b [8]byte
								_, _ = rand.Read(b[:])
								randomID := int64(binary.BigEndian.Uint64(b[:]))

								replyMarkup := &tg.ReplyKeyboardMarkup{
									Resize:     true,
									SingleUse:  true,
									Persistent: true,
									Rows: []tg.KeyboardButtonRow{
										{
											Buttons: []tg.KeyboardButtonClass{
												&tg.KeyboardButtonRequestPhone{Text: btnText},
											},
										},
									},
								}

								if client != nil {
									_, _ = client.API().MessagesSendMessage(context.Background(), &tg.MessagesSendMessageRequest{
										Peer: &tg.InputPeerUser{
											UserID:     uID,
											AccessHash: aHash,
										},
										Message:     startMsg,
										RandomID:    randomID,
										ReplyMarkup: replyMarkup,
									})
								}
							}(peerUser.UserID, accessHash)
						}
					}

					// 2. User bagikan contact / nomor telepon -> Kirim pesan konfirmasi & sembunyikan keyboard
					if msg.Media != nil {
						if _, isContact := msg.Media.(*tg.MessageMediaContact); isContact {
							go func(uID int64, aHash int64) {
								receivedMsg := strings.TrimSpace(device.BotPhoneReceivedMessage)
								if receivedMsg == "" {
									receivedMsg = "✅ Terima kasih! Nomor telepon Anda telah berhasil diterima."
								}

								var b [8]byte
								_, _ = rand.Read(b[:])
								randomID := int64(binary.BigEndian.Uint64(b[:]))

								if client != nil {
									_, _ = client.API().MessagesSendMessage(context.Background(), &tg.MessagesSendMessageRequest{
										Peer: &tg.InputPeerUser{
											UserID:     uID,
											AccessHash: aHash,
										},
										Message:     receivedMsg,
										RandomID:    randomID,
										ReplyMarkup: &tg.ReplyKeyboardHide{},
									})
								}
							}(peerUser.UserID, accessHash)
						}
					}
				}
			}

			// Sinkronisasi menu bot dinamis per user dari URL (cache 1 menit)
			if peerUser, isUser := msg.PeerID.(*tg.PeerUser); isUser {
				if device.BotMenuSource == "url" && strings.TrimSpace(device.BotMenuURL) != "" {
					phone := ""
					if payload.Sender != nil {
						phone = payload.Sender.Phone
					}
					go SyncUserBotMenu(context.Background(), m.httpClient, m.db, client.API(), device, peerUser.UserID, accessHash, phone)
				}
			}

			senderDesc := ""
			if payload.Sender != nil {
				if payload.Sender.FirstName != "" {
					senderDesc = payload.Sender.FirstName
				}
				if payload.Sender.Phone != "" {
					if senderDesc != "" {
						senderDesc += " (" + payload.Sender.Phone + ")"
					} else {
						senderDesc = payload.Sender.Phone
					}
				}
			}
			if senderDesc == "" && payload.Chat != nil {
				senderDesc = fmt.Sprintf("ID %d", payload.Chat.ID)
			}

			m.logs.Write("info", "webhook.received", fmt.Sprintf("Pesan Bot @%s masuk #%d diterima dari %s, meneruskan ke webhook...", device.BotUsername, msg.ID, senderDesc), &deviceID)
			go m.dispatch(deviceID, payload, fmt.Sprintf("pesan bot #%d", msg.ID))
			return nil
		})

		dispatcher.OnBotCallbackQuery(func(ctx context.Context, e tg.Entities, u *tg.UpdateBotCallbackQuery) error {
			if client != nil {
				go func(queryID int64) {
					_, _ = client.API().MessagesSetBotCallbackAnswer(context.Background(), &tg.MessagesSetBotCallbackAnswerRequest{
						QueryID: queryID,
					})
				}(u.QueryID)
			}

			payload, accessHash := ParseCallbackQueryToPayload(deviceID, e, u)
			payload.IsBot = true
			if client != nil {
				EnrichCallbackSenderInfo(ctx, m.db, client.API(), payload, accessHash)
			}

			// Sinkronisasi menu bot dinamis per user dari URL (cache 1 menit)
			if device.BotMenuSource == "url" && strings.TrimSpace(device.BotMenuURL) != "" && client != nil {
				phone := ""
				if payload.Sender != nil {
					phone = payload.Sender.Phone
				}
				go SyncUserBotMenu(context.Background(), m.httpClient, m.db, client.API(), device, u.UserID, accessHash, phone)
			}

			senderDesc := ""
			if payload.Sender != nil {
				if payload.Sender.FirstName != "" {
					senderDesc = payload.Sender.FirstName
				}
				if payload.Sender.Phone != "" {
					if senderDesc != "" {
						senderDesc += " (" + payload.Sender.Phone + ")"
					} else {
						senderDesc = payload.Sender.Phone
					}
				}
			}
			if senderDesc == "" && payload.Chat != nil {
				senderDesc = fmt.Sprintf("ID %d", payload.Chat.ID)
			}

			m.logs.Write("info", "webhook.callback", fmt.Sprintf("Bot callback query '%s' diterima dari %s, meneruskan ke webhook...", payload.Data, senderDesc), &deviceID)
			go m.dispatch(deviceID, payload, fmt.Sprintf("callback '%s'", payload.Data))
			return nil
		})

		dispatcher.OnInlineBotCallbackQuery(func(ctx context.Context, e tg.Entities, u *tg.UpdateInlineBotCallbackQuery) error {
			if client != nil {
				go func(queryID int64) {
					_, _ = client.API().MessagesSetBotCallbackAnswer(context.Background(), &tg.MessagesSetBotCallbackAnswerRequest{
						QueryID: queryID,
					})
				}(u.QueryID)
			}

			now := time.Now()
			payload := &WebhookCallbackQueryPayload{
				Event:    "callback_query",
				DeviceID: deviceID,
				IsBot:    true,
				QueryID:  u.QueryID,
				Date:     now.Unix(),
				DateTime: now.Format("2006-01-02 15:04:05"),
				Data:     string(u.Data),
			}

			sender := &WebhookSenderInfo{ID: u.UserID}
			var accessHash int64
			if user, ok := e.Users[u.UserID]; ok {
				accessHash = user.AccessHash
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

			if client != nil {
				EnrichCallbackSenderInfo(ctx, m.db, client.API(), payload, accessHash)
			}

			m.logs.Write("info", "webhook.callback", fmt.Sprintf("Bot inline callback query '%s' diterima, meneruskan ke webhook...", payload.Data), &deviceID)
			go m.dispatch(deviceID, payload, fmt.Sprintf("inline callback '%s'", payload.Data))
			return nil
		})

		gaps := updates.New(updates.Config{
			Handler: dispatcher,
		})

		client = telegram.NewClient(m.cfg.AppID, m.cfg.AppHash, telegram.Options{
			SessionStorage: NewDeviceBotSessionStorage(m.db, deviceID),
			UpdateHandler:  gaps,
			Middlewares: []telegram.Middleware{
				ratelimit.New(rate.Every(100*time.Millisecond), 5),
			},
		})

		err = client.Run(ctx, func(ctx context.Context) error {
			self, err := client.Self(ctx)
			if err != nil {
				return fmt.Errorf("failed to get bot self: %w", err)
			}

			return gaps.Run(ctx, client.API(), self.ID, updates.AuthOptions{
				OnStart: func(ctx context.Context) {
					m.logs.Write("info", "webhook.bot_connected", fmt.Sprintf("Listener webhook Bot MTProto aktif untuk @%s (Device #%d)", device.BotUsername, device.ID), &deviceID)
				},
			})
		})

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			m.logs.Write("warning", "webhook.bot_reconnecting", fmt.Sprintf("Listener Bot @%s terputus (%v), mencoba menghubungkan kembali...", device.BotUsername, err), &deviceID)
		}

		// Backoff sebelum auto reconnect
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (m *WebhookManager) dispatch(deviceID uint, payload any, desc string) {
	device, err := m.devices.Find(deviceID)
	if err != nil || !device.WebhookEnabled || strings.TrimSpace(device.WebhookURL) == "" {
		return
	}

	result, err := SendWebhookPayload(m.httpClient, device.WebhookURL, device.WebhookSecret, deviceID, payload)
	if err != nil || !result.Success {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else if result != nil {
			errMsg = result.Error
		}
		m.logs.Write("error", "webhook.failed", fmt.Sprintf("Gagal kirim webhook %s ke %s: %s", desc, device.WebhookURL, errMsg), &deviceID)
		return
	}

	m.logs.Write("info", "webhook.delivered", fmt.Sprintf("Webhook %s terkirim ke %s (HTTP %d, %dms)", desc, device.WebhookURL, result.StatusCode, result.DurationMs), &deviceID)
}

func (m *WebhookManager) TestWebhook(deviceID uint, customURL, customSecret string) (*WebhookTestResult, error) {
	device, err := m.devices.Find(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found")
	}

	targetURL := strings.TrimSpace(customURL)
	if targetURL == "" {
		targetURL = strings.TrimSpace(device.WebhookURL)
	}

	if targetURL == "" {
		return nil, fmt.Errorf("URL Webhook belum diisi")
	}

	targetSecret := customSecret
	if targetSecret == "" && strings.TrimSpace(customURL) == "" {
		targetSecret = device.WebhookSecret
	}

	testPayload := map[string]any{
		"event":     "webhook.test",
		"deviceId":  device.ID,
		"isBot":     true,
		"messageId": 999999,
		"date":      time.Now().Unix(),
		"dateTime":  time.Now().Format("2006-01-02 15:04:05"),
		"text":      "Ini adalah pesan uji coba (test payload) dari Telegram Sender Webhook.",
		"out":       false,
		"sender": map[string]any{
			"id":        123456789,
			"firstName": "Test",
			"lastName":  "User",
			"username":  "testuser",
			"phone":     "+628123456789",
		},
		"chat": map[string]any{
			"id":    123456789,
			"type":  "user",
			"title": "Test User",
		},
	}

	result, err := SendWebhookPayload(m.httpClient, targetURL, targetSecret, device.ID, testPayload)
	if err != nil {
		m.logs.Write("error", "webhook.test", fmt.Sprintf("Tes webhook ke %s gagal: %s", targetURL, err.Error()), &device.ID)
		return result, err
	}

	if result.Success {
		m.logs.Write("info", "webhook.test", fmt.Sprintf("Tes webhook ke %s berhasil (HTTP %d, %dms)", targetURL, result.StatusCode, result.DurationMs), &device.ID)
	} else {
		m.logs.Write("error", "webhook.test", fmt.Sprintf("Tes webhook ke %s gagal: HTTP %d", targetURL, result.StatusCode), &device.ID)
	}

	return result, nil
}
