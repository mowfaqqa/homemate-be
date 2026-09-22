package waitlist

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Entry struct {
	ID                int64     `json:"id"`
	FirstName         string    `json:"firstName"`
	Email             string    `json:"email"`
	City              string    `json:"city"`
	AssistantInterest bool      `json:"assistantInterest"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Store interface {
	Add(ctx context.Context, entry Entry) error
	List(ctx context.Context, filters Filters) ([]Entry, error)
}

type Handler struct {
	store Store
}

func NewHandler(store Store) http.Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, http.StatusBadRequest, "Send valid waitlist details.")
		return
	}

	normalized, err := validate(entry)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.Add(r.Context(), normalized); err != nil {
		if !errors.Is(err, ErrDuplicateEmail) {
			writeError(w, http.StatusInternalServerError, "We could not save your place. Please try again.")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "You are on the HomeMate waitlist.",
	})
}

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func validate(entry Entry) (Entry, error) {
	entry.FirstName = strings.TrimSpace(entry.FirstName)
	entry.Email = strings.ToLower(strings.TrimSpace(entry.Email))
	entry.City = strings.TrimSpace(entry.City)

	if len(entry.FirstName) < 2 {
		return entry, errors.New("Enter your first name.")
	}

	if !emailPattern.MatchString(entry.Email) {
		return entry, errors.New("Enter a valid email address.")
	}

	if len(entry.City) < 2 {
		return entry, errors.New("Enter your city.")
	}

	return entry, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
