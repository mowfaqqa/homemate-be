package waitlist

import (
	"crypto/subtle"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type AdminHandler struct {
	store    Store
	password string
}

func NewAdminHandler(store Store, password string) http.Handler {
	return &AdminHandler{store: store, password: password}
}

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.password == "" {
		writeError(w, http.StatusServiceUnavailable, "Admin access is not configured.")
		return
	}
	provided := r.Header.Get("X-Admin-Password")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.password)) != 1 {
		writeError(w, http.StatusUnauthorized, "Incorrect admin password.")
		return
	}

	filters, err := filtersFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entries, err := h.store.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "The waitlist could not be loaded.")
		return
	}

	if strings.HasSuffix(r.URL.Path, "/export") {
		writeCSV(w, entries)
		return
	}

	assistantCount := 0
	cities := make(map[string]struct{})
	for _, entry := range entries {
		if entry.AssistantInterest {
			assistantCount++
		}
		cities[strings.ToLower(entry.City)] = struct{}{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries":        entries,
		"total":          len(entries),
		"assistantCount": assistantCount,
		"cityCount":      len(cities),
	})
}

func filtersFromRequest(r *http.Request) (Filters, error) {
	filters := Filters{Search: r.URL.Query().Get("search")}
	value := r.URL.Query().Get("assistantInterest")
	if value == "" || value == "all" {
		return filters, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return filters, fmt.Errorf("assistantInterest must be true, false, or all")
	}
	filters.AssistantInterest = &parsed
	return filters, nil
}

func writeCSV(w http.ResponseWriter, entries []Entry) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="homemate-waitlist.csv"`)
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"First name", "Email", "City", "Assistant interest", "Joined at"})
	for _, entry := range entries {
		_ = writer.Write([]string{
			entry.FirstName,
			entry.Email,
			entry.City,
			strconv.FormatBool(entry.AssistantInterest),
			entry.CreatedAt.Format("2006-01-02 15:04:05 MST"),
		})
	}
	writer.Flush()
}
