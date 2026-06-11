package handlers

import (
	"net/http"
	"strconv"
)

func (h *Handlers) LogsIndex(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	logs, total, err := h.Logs.List(page, 20)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]any, len(logs))
	for i, l := range logs {
		items[i] = map[string]any{
			"id":        l.ID,
			"deviceId":  l.DeviceID,
			"level":     l.Level,
			"action":    l.Action,
			"message":   l.Message,
			"createdAt": l.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	h.render(w, r, "Logs/Index", map[string]any{
		"logs": items,
		"pagination": map[string]any{
			"page":  page,
			"total": total,
			"pages": (total + 19) / 20,
		},
	})
}
