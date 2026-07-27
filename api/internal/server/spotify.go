package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SpotifyConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	Market       string
}

type SpotifyClient struct {
	config SpotifyConfig
	http   *http.Client
}

type MusicTrack struct {
	Name            string   `json:"name"`
	Artists         []string `json:"artists"`
	Album           string   `json:"album,omitempty"`
	DurationMs      int      `json:"durationMs,omitempty"`
	DurationLabel   string   `json:"durationLabel,omitempty"`
	ProgressMs      int      `json:"progressMs,omitempty"`
	ProgressLabel   string   `json:"progressLabel,omitempty"`
	ProgressPercent int      `json:"progressPercent,omitempty"`
	ImageURL        string   `json:"imageUrl,omitempty"`
	ExternalURL     string   `json:"externalUrl,omitempty"`
	IsPlaying       bool     `json:"isPlaying,omitempty"`
}

func NewSpotifyClient(config SpotifyConfig) *SpotifyClient {
	if strings.TrimSpace(config.ClientID) == "" ||
		strings.TrimSpace(config.ClientSecret) == "" ||
		strings.TrimSpace(config.RefreshToken) == "" {
		return nil
	}
	if config.Market == "" {
		config.Market = "BR"
	}

	return &SpotifyClient{
		config: config,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (client *SpotifyClient) CurrentlyPlaying(ctx context.Context) (*MusicTrack, error) {
	token, err := client.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := "https://api.spotify.com/v1/me/player/currently-playing"
	query := url.Values{}
	query.Set("additional_types", "track")
	if client.config.Market != "" {
		query.Set("market", client.config.Market)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify currently playing status %d", response.StatusCode)
	}

	var payload spotifyCurrentlyResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if !payload.IsPlaying || payload.Item.Name == "" {
		return nil, nil
	}

	track := payload.Item.toMusicTrack()
	track.IsPlaying = true
	track.ProgressMs = payload.ProgressMs
	track.ProgressLabel = formatDuration(payload.ProgressMs)
	track.ProgressPercent = progressPercent(payload.ProgressMs, track.DurationMs)

	return &track, nil
}

func (client *SpotifyClient) TopTracks(ctx context.Context, limit int) ([]MusicTrack, error) {
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	token, err := client.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("time_range", "long_term")
	query.Set("limit", fmt.Sprint(limit))
	if client.config.Market != "" {
		query.Set("market", client.config.Market)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.spotify.com/v1/me/top/tracks?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify top tracks status %d", response.StatusCode)
	}

	var payload spotifyTopTracksResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	tracks := make([]MusicTrack, 0, len(payload.Items))
	for _, item := range payload.Items {
		tracks = append(tracks, item.toMusicTrack())
	}

	return tracks, nil
}

func (client *SpotifyClient) accessToken(ctx context.Context) (string, error) {
	body := url.Values{}
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", client.config.RefreshToken)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}

	credentials := client.config.ClientID + ":" + client.config.ClientSecret
	request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(credentials)))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("spotify token status %d", response.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", errors.New("spotify token response did not include access_token")
	}

	return payload.AccessToken, nil
}

type spotifyCurrentlyResponse struct {
	IsPlaying  bool         `json:"is_playing"`
	ProgressMs int          `json:"progress_ms"`
	Item       spotifyTrack `json:"item"`
}

type spotifyTopTracksResponse struct {
	Items []spotifyTrack `json:"items"`
}

type spotifyTrack struct {
	Name       string `json:"name"`
	DurationMs int    `json:"duration_ms"`
	Artists    []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name   string `json:"name"`
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"album"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

func (track spotifyTrack) toMusicTrack() MusicTrack {
	artists := make([]string, 0, len(track.Artists))
	for _, artist := range track.Artists {
		if artist.Name != "" {
			artists = append(artists, artist.Name)
		}
	}

	imageURL := ""
	if len(track.Album.Images) > 0 {
		imageURL = track.Album.Images[0].URL
	}

	return MusicTrack{
		Name:          track.Name,
		Artists:       artists,
		Album:         track.Album.Name,
		DurationMs:    track.DurationMs,
		DurationLabel: formatDuration(track.DurationMs),
		ImageURL:      imageURL,
		ExternalURL:   track.ExternalURLs.Spotify,
	}
}

func formatDuration(durationMs int) string {
	if durationMs <= 0 {
		return ""
	}

	totalSeconds := durationMs / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func progressPercent(progressMs, durationMs int) int {
	if progressMs <= 0 || durationMs <= 0 {
		return 0
	}

	percent := progressMs * 100 / durationMs
	if percent > 100 {
		return 100
	}

	return percent
}
