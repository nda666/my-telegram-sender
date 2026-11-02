package telegram

import (
	"encoding/json"
	"strings"

	"github.com/gotd/td/tg"
)

type ButtonItem struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	WebApp       any    `json:"web_app,omitempty"`
	WebAppURL    string `json:"web_app_url,omitempty"`
	Data         string `json:"data,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
	RequestPhone bool   `json:"request_phone,omitempty"`
}

func (b ButtonItem) GetData() string {
	if b.CallbackData != "" {
		return b.CallbackData
	}
	if b.Data != "" {
		return b.Data
	}
	return b.Text
}

func (b ButtonItem) GetWebAppURL() string {
	if strings.TrimSpace(b.WebAppURL) != "" {
		return strings.TrimSpace(b.WebAppURL)
	}
	if b.WebApp != nil {
		if s, ok := b.WebApp.(string); ok {
			return strings.TrimSpace(s)
		}
		if m, ok := b.WebApp.(map[string]any); ok {
			if u, ok := m["url"].(string); ok {
				return strings.TrimSpace(u)
			}
		}
	}
	return ""
}

// ExtractButtons mengurai input buttons (1D/2D array atau JSON string) menjadi [][]ButtonItem.
func ExtractButtons(raw any) [][]ButtonItem {
	if raw == nil {
		return nil
	}

	var rows [][]ButtonItem

	// Jika raw berupa string JSON (misal dari multipart/form-data)
	if s, ok := raw.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err != nil {
			return nil
		}
		raw = parsed
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	// Coba format 2D: [[{text, url}], [{text, url}]]
	if err := json.Unmarshal(b, &rows); err != nil {
		// Coba format 1D: [{text, url}, {text, url}]
		var singleList []ButtonItem
		if err := json.Unmarshal(b, &singleList); err == nil && len(singleList) > 0 {
			for _, item := range singleList {
				rows = append(rows, []ButtonItem{item})
			}
		} else {
			return nil
		}
	}

	return rows
}

// ParseReplyMarkup mengonversi data buttons menjadi tg.ReplyMarkupClass.
// - "inline": ReplyInlineMarkup (tombol inline URL / callback data / web_app).
// - "reply" / "keyboard": ReplyKeyboardMarkup (tombol keyboard native Telegram / web_app).
// - "force_reply": ReplyKeyboardForceReply.
func ParseReplyMarkup(raw any, buttonType string) tg.ReplyMarkupClass {
	rows := ExtractButtons(raw)
	if len(rows) == 0 {
		return nil
	}

	bType := strings.ToLower(strings.TrimSpace(buttonType))

	// Force Reply
	if bType == "force_reply" || bType == "forcereply" {
		placeholder := "Ketik balasan Anda..."
		if len(rows) > 0 && len(rows[0]) > 0 && strings.TrimSpace(rows[0][0].Text) != "" {
			placeholder = strings.TrimSpace(rows[0][0].Text)
		}
		return &tg.ReplyKeyboardForceReply{
			SingleUse:   true,
			Placeholder: placeholder,
		}
	}

	// Cek apakah ada callback data, URL button, atau WebApp button
	hasCallbackData := false
	hasURL := false
	hasWebApp := false
	for _, row := range rows {
		for _, btn := range row {
			if btn.CallbackData != "" || btn.Data != "" {
				hasCallbackData = true
			}
			if strings.TrimSpace(btn.URL) != "" {
				hasURL = true
			}
			if btn.GetWebAppURL() != "" {
				hasWebApp = true
			}
		}
	}

	// Jika tipe "reply" / "keyboard" / "request_phone", atau jika buttonType tidak diisi dan tidak ada callback_data & tidak ada url & tidak ada web_app:
	if (bType == "reply" || bType == "keyboard" || bType == "request_phone") || (bType == "" && !hasCallbackData && !hasURL && !hasWebApp) {
		var tgRows []tg.KeyboardButtonRow
		for _, row := range rows {
			var tgButtons []tg.KeyboardButtonClass
			for _, btn := range row {
				text := strings.TrimSpace(btn.Text)
				if text == "" {
					continue
				}
				if btn.RequestPhone || bType == "request_phone" {
					tgButtons = append(tgButtons, &tg.KeyboardButtonRequestPhone{Text: text})
				} else if webURL := btn.GetWebAppURL(); webURL != "" {
					tgButtons = append(tgButtons, &tg.KeyboardButtonSimpleWebView{
						Text: text,
						URL:  webURL,
					})
				} else {
					tgButtons = append(tgButtons, &tg.KeyboardButton{Text: text})
				}
			}
			if len(tgButtons) > 0 {
				tgRows = append(tgRows, tg.KeyboardButtonRow{Buttons: tgButtons})
			}
		}
		if len(tgRows) == 0 {
			return nil
		}
		return &tg.ReplyKeyboardMarkup{
			Resize:     true,
			SingleUse:  false,
			Persistent: true,
			Rows:       tgRows,
		}
	}

	// Default: Inline Keyboard Markup (untuk URL, WebApp, & Callback Data)
	var tgRows []tg.KeyboardButtonRow
	for _, row := range rows {
		var tgButtons []tg.KeyboardButtonClass
		for _, btn := range row {
			text := strings.TrimSpace(btn.Text)
			if text == "" {
				continue
			}
			if webURL := btn.GetWebAppURL(); webURL != "" {
				tgButtons = append(tgButtons, &tg.KeyboardButtonWebView{
					Text: text,
					URL:  webURL,
				})
			} else if btn.URL != "" {
				tgButtons = append(tgButtons, &tg.KeyboardButtonURL{
					Text: text,
					URL:  btn.URL,
				})
			} else {
				tgButtons = append(tgButtons, &tg.KeyboardButtonCallback{
					Text: text,
					Data: []byte(btn.GetData()),
				})
			}
		}
		if len(tgButtons) > 0 {
			tgRows = append(tgRows, tg.KeyboardButtonRow{Buttons: tgButtons})
		}
	}

	if len(tgRows) == 0 {
		return nil
	}

	return &tg.ReplyInlineMarkup{
		Rows: tgRows,
	}
}
