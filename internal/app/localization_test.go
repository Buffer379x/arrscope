package app

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocaleSelectionAndMessages(t *testing.T) {
	cases := []struct{ header, query, want string }{{"es-MX,en;q=0.5", "", "es"}, {"it;q=0.8,de;q=0.9", "", "de"}, {"de;q=0,en;q=1", "", "en"}, {"fr-FR", "", "en"}, {"es", "?lang=it", "it"}, {"de", "?lang=../../etc/passwd", "de"}}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/api/status"+c.query, nil)
		r.Header.Set("Accept-Language", c.header)
		if got := requestLocale(r); got != c.want {
			t.Fatalf("%s %s: %s", c.header, c.query, got)
		}
	}
	for _, locale := range []string{"de", "en", "es", "it"} {
		if localizedMessage(locale, "API key was rejected") == "" {
			t.Fatal(locale)
		}
		if localizedMessage(locale, "Service returned HTTP 503") == "Service returned HTTP %d" {
			t.Fatal("format argument lost")
		}
	}
	if got := localizedMessage("es", "API key was rejected"); got != "Clave API rechazada" {
		t.Fatal(got)
	}
	if got := localizedMessage("de", "Service returned HTTP 503"); got != "Dienst meldet HTTP 503" {
		t.Fatal(got)
	}
	if got := localizedMessage("it", "Emby enrichment failed: API key was rejected"); got != "Arricchimento Emby fallito: Chiave API rifiutata" {
		t.Fatal(got)
	}
	if got := localizedMessage("en", "API-Key wurde abgelehnt"); got != "API key was rejected" {
		t.Fatal("legacy persisted error", got)
	}
}

func TestLocalizedHTTPErrorAndCSV(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/files", nil)
	request.Header.Set("Accept-Language", "es")
	response := httptest.NewRecorder()
	failure(response, request, http.StatusUnauthorized, errors.New("API key was rejected"))
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusUnauthorized || body["error"] != "Clave API rechazada" {
		t.Fatal(response.Code, body)
	}
	store := testStore(t)
	file := sample(1, "Demo.German.English.1080p.x264.mkv")
	if err := store.Replace("radarr", []File{file}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: store}
	response = httptest.NewRecorder()
	server.export(response, httptest.NewRequest("GET", "/api/export?service=radarr&lang=IT", nil))
	if response.Code != http.StatusOK {
		t.Fatal(response.Code, response.Body.String())
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(response.Body.String(), "\xef\xbb\xbf")))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0][0] != "Titolo" || records[1][4] != file.Filename || records[1][14] != file.Language {
		t.Fatal(records)
	}
}
