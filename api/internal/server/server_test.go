package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHealthz(t *testing.T) {
	app := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	app.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected healthy response, got %q", body["status"])
	}
}

func TestGuestbookModerationFlow(t *testing.T) {
	app := newTestServer(t)

	createRes := request(app, http.MethodPost, "/guestbook/signatures", map[string]string{
		"name":    "Kao",
		"message": "Hello guestbook",
	})
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createRes.Code)
	}

	var signature Signature
	decodeBody(t, createRes, &signature)
	if signature.Status != SignaturePending {
		t.Fatalf("expected pending signature, got %q", signature.Status)
	}

	latestBeforeApproval := request(app, http.MethodGet, "/guestbook/signatures/latest", nil)
	var latestPayload struct {
		Signatures []Signature `json:"signatures"`
	}
	decodeBody(t, latestBeforeApproval, &latestPayload)
	if len(latestPayload.Signatures) != 0 {
		t.Fatalf("expected no public signatures before approval, got %d", len(latestPayload.Signatures))
	}

	approveRes := requestWithAdmin(app, http.MethodPatch, "/admin/guestbook/signatures/"+signature.ID+"/approve", nil)
	if approveRes.Code != http.StatusOK {
		t.Fatalf("expected approve status %d, got %d", http.StatusOK, approveRes.Code)
	}

	latestAfterApproval := request(app, http.MethodGet, "/guestbook/signatures/latest?limit=3", nil)
	decodeBody(t, latestAfterApproval, &latestPayload)
	if len(latestPayload.Signatures) != 1 {
		t.Fatalf("expected one public signature after approval, got %d", len(latestPayload.Signatures))
	}
	if latestPayload.Signatures[0].Name != "Kao" {
		t.Fatalf("expected signature from Kao, got %q", latestPayload.Signatures[0].Name)
	}
}

func TestCurrentlyAndVisitors(t *testing.T) {
	app := newTestServer(t)

	currentlyRes := request(app, http.MethodGet, "/status/currently", nil)
	var currently Currently
	decodeBody(t, currentlyRes, &currently)
	if currently.Listening == "" || !currently.IsOnline {
		t.Fatalf("expected default currently payload, got %+v", currently)
	}

	updateRes := requestWithAdmin(app, http.MethodPut, "/admin/status/currently", Currently{
		Listening: "New song.mp3",
		IsOnline:  true,
	})
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected currently update status %d, got %d", http.StatusOK, updateRes.Code)
	}

	visitRes := request(app, http.MethodPost, "/visits", nil)
	if visitRes.Code != http.StatusCreated {
		t.Fatalf("expected visit status %d, got %d", http.StatusCreated, visitRes.Code)
	}

	widgetsRes := request(app, http.MethodGet, "/widgets/home", nil)
	var widgets struct {
		Currently Currently `json:"currently"`
		Visitors  int64     `json:"visitors"`
	}
	decodeBody(t, widgetsRes, &widgets)
	if widgets.Currently.Listening != "New song.mp3" {
		t.Fatalf("expected updated currently in widgets, got %+v", widgets.Currently)
	}
	if widgets.Visitors != 124 {
		t.Fatalf("expected visitor count 124, got %d", widgets.Visitors)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	return New(Config{
		Addr:       ":0",
		AdminToken: "test-token",
		DataPath:   filepath.Join(t.TempDir(), "api.json"),
	}, slog.Default())
}

func request(app *Server, method, path string, body any) *httptest.ResponseRecorder {
	req := newJSONRequest(method, path, body)
	res := httptest.NewRecorder()

	app.routes().ServeHTTP(res, req)

	return res
}

func requestWithAdmin(app *Server, method, path string, body any) *httptest.ResponseRecorder {
	req := newJSONRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer test-token")

	res := httptest.NewRecorder()
	app.routes().ServeHTTP(res, req)

	return res
}

func newJSONRequest(method, path string, body any) *http.Request {
	if body == nil {
		return httptest.NewRequest(method, path, nil)
	}

	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(body); err != nil {
		panic(err)
	}

	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")

	return req
}

func decodeBody(t *testing.T, res *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
