package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckForUpdateWithClientRequiresSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if got, err := checkForUpdateWithClient(server.Client(), server.URL); err == nil {
		t.Fatalf("checkForUpdateWithClient returned (%q, nil), want error", got)
	}
}

func TestCheckForUpdateWithClientRejectsInvalidTags(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: `{"tag_name":""}`},
		{name: "missing v prefix", body: `{"tag_name":"0.4.10"}`},
		{name: "non numeric", body: `{"tag_name":"v0.4.latest"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			if got, err := checkForUpdateWithClient(server.Client(), server.URL); err == nil {
				t.Fatalf("checkForUpdateWithClient returned (%q, nil), want error", got)
			}
		})
	}
}

func TestCheckForUpdateWithClientReturnsTrimmedValidTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":" v0.4.10 "}`))
	}))
	defer server.Close()

	got, err := checkForUpdateWithClient(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("checkForUpdateWithClient returned error: %v", err)
	}
	if got != "v0.4.10" {
		t.Fatalf("tag = %q, want v0.4.10", got)
	}
}

func TestUpdateAvailableMsgIgnoresInvalidAndCurrentTags(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{name: "empty", tag: ""},
		{name: "invalid", tag: "latest"},
		{name: "current", tag: "v0.4.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel()
			m.version = "v0.4.10"

			updated, _ := m.Update(updateAvailableMsg{latestVersion: tt.tag})
			m = modelFromUpdate(t, updated)
			if m.updateAvailable != "" {
				t.Fatalf("updateAvailable = %q, want empty", m.updateAvailable)
			}
		})
	}
}

func TestUpdateAvailableMsgStoresNewValidTag(t *testing.T) {
	m := testModel()
	m.version = "v0.4.10"

	updated, _ := m.Update(updateAvailableMsg{latestVersion: " v0.4.11 "})
	m = modelFromUpdate(t, updated)
	if m.updateAvailable != "v0.4.11" {
		t.Fatalf("updateAvailable = %q, want v0.4.11", m.updateAvailable)
	}
}
