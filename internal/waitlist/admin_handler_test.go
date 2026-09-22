package waitlist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminHandlerRequiresPasswordAndFiltersAssistants(t *testing.T) {
	store := NewMemoryStore()
	for _, entry := range []Entry{
		{FirstName: "Ada", Email: "ada@example.com", City: "Lagos", AssistantInterest: true},
		{FirstName: "Tobi", Email: "tobi@example.com", City: "Abuja", AssistantInterest: false},
	} {
		if err := store.Add(context.Background(), entry); err != nil {
			t.Fatalf("seed store: %v", err)
		}
	}
	handler := NewAdminHandler(store, "test-secret")

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/admin/waitlist", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/admin/waitlist?assistantInterest=true", nil)
	request.Header.Set("X-Admin-Password", "test-secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, want %d", response.Code, http.StatusOK)
	}

	var payload struct {
		Entries        []Entry `json:"entries"`
		Total          int     `json:"total"`
		AssistantCount int     `json:"assistantCount"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || payload.AssistantCount != 1 || payload.Entries[0].Email != "ada@example.com" {
		t.Fatalf("unexpected filtered response: %+v", payload)
	}
}

func TestMemoryStoreRejectsDuplicateEmail(t *testing.T) {
	store := NewMemoryStore()
	entry := Entry{FirstName: "Ada", Email: "ada@example.com", City: "Lagos"}
	if err := store.Add(context.Background(), entry); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := store.Add(context.Background(), entry); err != ErrDuplicateEmail {
		t.Fatalf("duplicate error = %v, want %v", err, ErrDuplicateEmail)
	}
}
