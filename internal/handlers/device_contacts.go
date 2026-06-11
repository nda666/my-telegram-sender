package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

// IndexContacts — GET /contacts?device_id=1
// IndexContacts — GET /contacts?device_id=1&page=1&page_size=10&search_name=budi&search_username=budi&search_phone=628
func (h *Handlers) IndexContacts(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := parseID(r, "id")
	if !ok {
		jsonError(w, "device_id required", http.StatusBadRequest)
		return
	}

	// --- query params ---
	q := r.URL.Query()
	searchName := strings.ToLower(q.Get("search_name"))
	searchUsername := strings.ToLower(q.Get("search_username"))
	searchPhone := q.Get("search_phone")

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}

	// --- fetch all contacts from Telegram ---
	all, err := h.Telegram.GetContacts(r.Context(), deviceID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// --- filter ---
	filtered := all[:0:0] // same backing array, len=0
	for _, c := range all {
		if searchName != "" && !strings.Contains(strings.ToLower(c.FirstName+" "+c.LastName), searchName) {
			continue
		}
		if searchUsername != "" && !strings.Contains(strings.ToLower(c.Username), searchUsername) {
			continue
		}
		if searchPhone != "" && !strings.Contains(c.Phone, searchPhone) {
			continue
		}
		filtered = append(filtered, c)
	}

	// --- paginate ---
	total := len(filtered)
	offset := (page - 1) * pageSize
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	paged := filtered[offset:end]

	h.Inertia.Render(w, r, "Devices/Contacts", map[string]any{
		"contacts":       paged,
		"deviceID":       deviceID,
		"total":          total,
		"page":           page,
		"pageSize":       pageSize,
		"searchName":     q.Get("search_name"),
		"searchUsername": q.Get("search_username"),
		"searchPhone":    q.Get("search_phone"),
	})
}

// StoreContact — POST /contacts
// Body: { "device_id": 1, "phone": "+628xxx", "first_name": "Budi", "last_name": "S" }
func (h *Handlers) StoreContact(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		DeviceID  uint   `json:"device_id"`
		Phone     string `json:"phone"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	req, err := Bind[Req](r)
	if err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Phone == "" || req.FirstName == "" || req.DeviceID == 0 {
		jsonError(w, "device_id, phone, and first_name are required", http.StatusUnprocessableEntity)
		return
	}

	contact, err := h.Telegram.ImportContact(r.Context(), req.DeviceID, req.Phone, req.FirstName, req.LastName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, contact)
}

// UpdateContact — PUT /contacts/{user_id}
// Body: { "device_id": 1, "phone": "+628xxx", "first_name": "Budi", "last_name": "S" }
func (h *Handlers) UpdateContact(w http.ResponseWriter, r *http.Request) {
	// userTelegramId, ok := parseID(r, "user_telegram_id")
	// if !ok {
	// 	respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid contact ID"})
	// 	return
	// }
	type Req struct {
		DeviceID  int64  `json:"device_id"`
		Phone     string `json:"phone"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	req, err := Bind[Req](r)
	if err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	contact, err := h.Telegram.EditContact(r.Context(), uint(req.DeviceID), req.Phone, req.FirstName, req.LastName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, contact)
}

// DeleteContact — DELETE /contacts/{user_id}
// Body: { "device_id": 1, "access_hash": 123456 }
func (h *Handlers) DeleteContact(w http.ResponseWriter, r *http.Request) {
	userTelegramId, ok := parseID(r, "user_telegram_id")
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user_id"})
		return
	}
	type Req struct {
		DeviceID   int64 `json:"device_id"`
		AccessHash int64 `json:"access_hash"`
	}
	req, err := Bind[Req](r)
	if err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.Telegram.DeleteContact(r.Context(), uint(req.DeviceID), int64(userTelegramId), req.AccessHash); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"message": "Contact deleted"})
}

// --- helpers ---

func queryUint(r *http.Request, key string) (uint, error) {
	v, err := strconv.ParseUint(r.URL.Query().Get(key), 10, 64)
	return uint(v), err
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
