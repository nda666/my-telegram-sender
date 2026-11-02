package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/tg"
	"github.com/tiar/telegram-sender/internal/models"
	"gorm.io/gorm"
)

type botMenuCacheEntry struct {
	commands []BotCommandItem
	expireAt time.Time
	synced   bool
}

type botMenuCacheManager struct {
	mu      sync.RWMutex
	entries map[string]botMenuCacheEntry
}

var globalBotMenuCache = &botMenuCacheManager{
	entries: make(map[string]botMenuCacheEntry),
}

func (c *botMenuCacheManager) get(key string) (botMenuCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expireAt) {
		return botMenuCacheEntry{}, false
	}
	return entry, true
}

func (c *botMenuCacheManager) set(key string, commands []BotCommandItem, ttl time.Duration, synced bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = botMenuCacheEntry{
		commands: commands,
		expireAt: time.Now().Add(ttl),
		synced:   synced,
	}
}

func (c *botMenuCacheManager) markSynced(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[key]; ok {
		entry.synced = true
		c.entries[key] = entry
	}
}

func DefaultBotCommands() []BotCommandItem {
	return []BotCommandItem{
		{Command: "start", Description: "Mulai bot dan buka menu utama"},
	}
}

// FetchBotMenuFromURL mengambil daftar menu commands dari URL eksternal dengan parameter ?phone=...&user_id=...&device_id=...
func FetchBotMenuFromURL(ctx context.Context, httpClient *http.Client, rawURL string, deviceID uint, userID int64, phone string) ([]BotCommandItem, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid menu URL: %w", err)
	}

	q := u.Query()
	q.Set("phone", phone)
	q.Set("user_id", strconv.FormatInt(userID, 10))
	q.Set("device_id", strconv.FormatUint(uint64(deviceID), 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Telegram-Sender-Bot-Menu/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to menu url failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("menu url returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 1. Coba format langsung array: [{"command":"start","description":"..."}, ...]
	var list []map[string]any
	if err := json.Unmarshal(body, &list); err != nil {
		// 2. Coba format object wrapper: {"commands": [...], "data": [...], "menu": [...]}
		var obj map[string]any
		if err2 := json.Unmarshal(body, &obj); err2 == nil {
			for _, key := range []string{"commands", "data", "menu", "items"} {
				if sub, ok := obj[key]; ok {
					if subArr, ok := sub.([]any); ok {
						subBytes, _ := json.Marshal(subArr)
						_ = json.Unmarshal(subBytes, &list)
						break
					}
				}
			}
		}
	}

	if len(list) == 0 {
		return DefaultBotCommands(), nil
	}

	var items []BotCommandItem
	for _, item := range list {
		cmdVal := ""
		for _, k := range []string{"command", "cmd", "name", "action"} {
			if v, ok := item[k].(string); ok && strings.TrimSpace(v) != "" {
				cmdVal = v
				break
			}
		}

		descVal := ""
		for _, k := range []string{"description", "desc", "title", "label", "text"} {
			if v, ok := item[k].(string); ok && strings.TrimSpace(v) != "" {
				descVal = v
				break
			}
		}

		cleanCmd := strings.TrimPrefix(strings.TrimSpace(cmdVal), "/")
		cleanCmd = strings.ToLower(cleanCmd)
		if cleanCmd == "" {
			continue
		}
		if descVal == "" {
			descVal = cleanCmd
		}

		items = append(items, BotCommandItem{
			Command:     cleanCmd,
			Description: descVal,
		})
	}

	if len(items) == 0 {
		return DefaultBotCommands(), nil
	}

	return items, nil
}

// GetUserBotCommands mengambil daftar bot commands (baik dari cache 1 menit, URL, atau DB manual)
func GetUserBotCommands(ctx context.Context, httpClient *http.Client, db *gorm.DB, device *models.Device, userID int64, phone string) ([]BotCommandItem, bool) {
	if device == nil {
		return DefaultBotCommands(), false
	}

	// 1. Mode Manual
	if strings.ToLower(strings.TrimSpace(device.BotMenuSource)) != "url" || strings.TrimSpace(device.BotMenuURL) == "" {
		if strings.TrimSpace(device.BotCommandsJSON) != "" {
			var manualCmds []BotCommandItem
			if err := json.Unmarshal([]byte(device.BotCommandsJSON), &manualCmds); err == nil && len(manualCmds) > 0 {
				return manualCmds, false
			}
		}
		return DefaultBotCommands(), false
	}

	// 2. Mode URL: Cek Cache 1 Menit
	cacheKey := fmt.Sprintf("%d:%d", device.ID, userID)
	if entry, ok := globalBotMenuCache.get(cacheKey); ok {
		return entry.commands, entry.synced
	}

	// Ambil phone dari DB jika belum ada
	if phone == "" && db != nil {
		p, _, _, _ := FindSenderContactPhone(db, device.ID, userID)
		phone = p
	}

	// Request ke API URL
	cmds, err := FetchBotMenuFromURL(ctx, httpClient, device.BotMenuURL, device.ID, userID, phone)
	if err != nil || len(cmds) == 0 {
		cmds = DefaultBotCommands()
	}

	// Cache selama 1 menit (60 detik)
	globalBotMenuCache.set(cacheKey, cmds, 1*time.Minute, false)
	return cmds, false
}

// SyncUserBotMenu menyinkronkan menu bot per user (BotCommandScopePeer) ke Telegram MTProto
func SyncUserBotMenu(ctx context.Context, httpClient *http.Client, db *gorm.DB, api *tg.Client, device *models.Device, userID int64, accessHash int64, phone string) {
	if api == nil || device == nil || userID == 0 {
		return
	}

	cacheKey := fmt.Sprintf("%d:%d", device.ID, userID)
	cmds, alreadySynced := GetUserBotCommands(ctx, httpClient, db, device, userID, phone)
	if alreadySynced {
		return
	}

	var tgCmds []tg.BotCommand
	for _, c := range cmds {
		cmd := strings.TrimPrefix(strings.TrimSpace(c.Command), "/")
		desc := strings.TrimSpace(c.Description)
		if cmd == "" || desc == "" {
			continue
		}
		tgCmds = append(tgCmds, tg.BotCommand{
			Command:     cmd,
			Description: desc,
		})
	}

	if len(tgCmds) == 0 {
		tgCmds = []tg.BotCommand{
			{Command: "start", Description: "Mulai bot dan buka menu utama"},
		}
	}

	_, err := api.BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
		Scope: &tg.BotCommandScopePeer{
			Peer: &tg.InputPeerUser{
				UserID:     userID,
				AccessHash: accessHash,
			},
		},
		LangCode: "",
		Commands: tgCmds,
	})

	if err == nil {
		globalBotMenuCache.markSynced(cacheKey)
	}
}
