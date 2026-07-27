package server

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func (app *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (app *Server) handleCreateVisit(w http.ResponseWriter, r *http.Request) {
	count, err := app.store.IncrementVisitors()
	if err != nil {
		app.logger.Error("increment visitors", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to record visit")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"visitors": count})
}

func (app *Server) handleGetVisitors(w http.ResponseWriter, r *http.Request) {
	visitors, err := app.store.VisitorCount()
	if err != nil {
		app.logger.Error("get visitors", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load visitors")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"visitors": visitors})
}

func (app *Server) handleHomeWidgets(w http.ResponseWriter, r *http.Request) {
	currently, err := app.store.Currently()
	if err != nil {
		app.logger.Error("get home currently", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load widgets")
		return
	}
	currently = app.withSpotifyListening(r, currently)

	signatures, err := app.store.Signatures(SignatureApproved, 3, 0)
	if err != nil {
		app.logger.Error("get home signatures", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load widgets")
		return
	}

	visitors, err := app.store.VisitorCount()
	if err != nil {
		app.logger.Error("get home visitors", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load widgets")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"currently":        currently,
		"latestSignatures": signatures,
		"visitors":         visitors,
	})
}

func (app *Server) handleGetCurrently(w http.ResponseWriter, r *http.Request) {
	currently, err := app.store.Currently()
	if err != nil {
		app.logger.Error("get currently", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load currently")
		return
	}

	writeJSON(w, http.StatusOK, app.withSpotifyListening(r, currently))
}

func (app *Server) handleUpdateCurrently(w http.ResponseWriter, r *http.Request) {
	var request Currently
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	request.Listening = strings.TrimSpace(request.Listening)
	if request.Listening == "" {
		writeError(w, http.StatusBadRequest, "listening is required")
		return
	}

	if err := app.store.UpdateCurrently(request); err != nil {
		app.logger.Error("update currently", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update currently")
		return
	}

	writeJSON(w, http.StatusOK, request)
}

func (app *Server) handleTopTracks(w http.ResponseWriter, r *http.Request) {
	if app.spotify == nil {
		writeJSON(w, http.StatusOK, map[string]any{"tracks": []MusicTrack{}})
		return
	}

	tracks, err := app.spotify.TopTracks(r.Context(), queryInt(r, "limit", 5))
	if err != nil {
		app.logger.Error("get spotify top tracks", "error", err)
		writeJSON(w, http.StatusOK, map[string]any{"tracks": []MusicTrack{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tracks": tracks})
}

func (app *Server) withSpotifyListening(r *http.Request, currently Currently) Currently {
	if app.spotify == nil {
		return currently
	}

	track, err := app.spotify.CurrentlyPlaying(r.Context())
	if err != nil {
		app.logger.Error("get spotify currently playing", "error", err)
		return currently
	}
	if track == nil {
		return currently
	}

	currently.Listening = formatTrackLabel(*track)
	currently.Track = track

	return currently
}

func (app *Server) handleCreateSignature(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(request.Name)
	message := strings.TrimSpace(request.Message)
	if name == "" || message == "" {
		writeError(w, http.StatusBadRequest, "name and message are required")
		return
	}
	if len(name) > 48 {
		writeError(w, http.StatusBadRequest, "name is too long")
		return
	}
	if len(message) > 600 {
		writeError(w, http.StatusBadRequest, "message is too long")
		return
	}

	signature, err := app.store.CreateSignature(name, message, time.Now())
	if err != nil {
		app.logger.Error("create signature", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create signature")
		return
	}

	writeJSON(w, http.StatusCreated, signature)
}

func formatTrackLabel(track MusicTrack) string {
	if len(track.Artists) == 0 {
		return track.Name
	}

	return track.Name + " - " + strings.Join(track.Artists, ", ")
}

func (app *Server) handleListApprovedSignatures(w http.ResponseWriter, r *http.Request) {
	signatures, err := app.store.Signatures(SignatureApproved, queryInt(r, "limit", 10), queryInt(r, "offset", 0))
	if err != nil {
		app.logger.Error("list approved signatures", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load signatures")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"signatures": signatures,
	})
}

func (app *Server) handleLatestSignatures(w http.ResponseWriter, r *http.Request) {
	signatures, err := app.store.Signatures(SignatureApproved, queryInt(r, "limit", 3), 0)
	if err != nil {
		app.logger.Error("list latest signatures", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load signatures")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"signatures": signatures,
	})
}

func (app *Server) handleListPendingSignatures(w http.ResponseWriter, r *http.Request) {
	signatures, err := app.store.Signatures(SignaturePending, queryInt(r, "limit", 20), queryInt(r, "offset", 0))
	if err != nil {
		app.logger.Error("list pending signatures", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load signatures")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"signatures": signatures,
	})
}

func (app *Server) handleApproveSignature(w http.ResponseWriter, r *http.Request) {
	app.updateSignatureStatus(w, r, SignatureApproved)
}

func (app *Server) handleRejectSignature(w http.ResponseWriter, r *http.Request) {
	app.updateSignatureStatus(w, r, SignatureRejected)
}

func (app *Server) updateSignatureStatus(w http.ResponseWriter, r *http.Request, status SignatureStatus) {
	signature, err := app.store.UpdateSignatureStatus(r.PathValue("id"), status, time.Now())
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "signature not found")
		return
	}
	if err != nil {
		app.logger.Error("update signature status", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update signature")
		return
	}

	writeJSON(w, http.StatusOK, signature)
}
