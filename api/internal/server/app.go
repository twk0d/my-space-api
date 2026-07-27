package server

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Addr        string
	AdminToken  string
	AllowOrigin string
	DataPath    string
	Spotify     SpotifyConfig
}

func ConfigFromEnv() Config {
	loadDotEnv()

	return Config{
		Addr:        envWithDefault("API_ADDR", ":8080"),
		AdminToken:  os.Getenv("API_ADMIN_TOKEN"),
		AllowOrigin: envWithDefault("API_ALLOW_ORIGIN", "http://localhost:3000"),
		DataPath:    envWithDefault("API_DATA_PATH", ".data/api.json"),
		Spotify: SpotifyConfig{
			ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
			ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
			RefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
			Market:       envWithDefault("SPOTIFY_MARKET", "BR"),
		},
	}
}

type Server struct {
	config  Config
	logger  *slog.Logger
	server  *http.Server
	store   DataStore
	spotify *SpotifyClient
}

func New(config Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.DataPath == "" {
		config.DataPath = ".data/api.json"
	}

	store, err := NewJSONStore(config.DataPath)
	if err != nil {
		panic(err)
	}

	app := &Server{
		config:  config,
		logger:  logger,
		store:   store,
		spotify: NewSpotifyClient(config.Spotify),
	}

	app.server = &http.Server{
		Addr:    config.Addr,
		Handler: app.withCORS(app.routes()),
	}

	return app
}

func (app *Server) Addr() string {
	return app.server.Addr
}

func (app *Server) ListenAndServe() error {
	return app.server.ListenAndServe()
}

func (app *Server) Close() {
	app.store.Close()
}

func (app *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", app.handleHealthz)
	mux.HandleFunc("GET /status/currently", app.handleGetCurrently)
	mux.HandleFunc("GET /spotify/top-tracks", app.handleTopTracks)
	mux.HandleFunc("GET /stats/visitors", app.handleGetVisitors)
	mux.HandleFunc("POST /visits", app.handleCreateVisit)
	mux.HandleFunc("POST /guestbook/signatures", app.handleCreateSignature)
	mux.HandleFunc("GET /guestbook/signatures", app.handleListApprovedSignatures)
	mux.HandleFunc("GET /guestbook/signatures/latest", app.handleLatestSignatures)
	mux.HandleFunc("GET /widgets/home", app.handleHomeWidgets)
	mux.HandleFunc("GET /admin/guestbook/signatures/pending", app.withAdmin(app.handleListPendingSignatures))
	mux.HandleFunc("PATCH /admin/guestbook/signatures/{id}/approve", app.withAdmin(app.handleApproveSignature))
	mux.HandleFunc("PATCH /admin/guestbook/signatures/{id}/reject", app.withAdmin(app.handleRejectSignature))
	mux.HandleFunc("PUT /admin/status/currently", app.withAdmin(app.handleUpdateCurrently))

	return mux
}

func (app *Server) withCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.AllowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", app.config.AllowOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func (app *Server) withAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.config.AdminToken == "" {
			writeError(w, http.StatusServiceUnavailable, "admin token is not configured")
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != app.config.AdminToken {
			writeError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}

		handler(w, r)
	}
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func queryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		return fallback
	}

	return value
}

func envWithDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func loadDotEnv() {
	path := findDotEnv()
	if path == "" {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
