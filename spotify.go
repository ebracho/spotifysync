package spotifysync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Refresh current track data only after 10 seconds
const currentTrackTTL = 10 * time.Second

type Track struct {
	URI string `json:"uri"`
}

type CurrentTrack struct {
	Track                Track  `json:"item"`
	IsPlaying            bool   `json:"is_playing"`
	CurrentlyPlayingType string `json:"currently_playing_type"`
	ProgressMS           int    `json:"progress_ms"`
	Timestamp            int64  `json:"timestamp"` // unix millis
	Fetched              int64  `json:"fetched"`   // unix seconds
}

type SpotifyClient struct {
	Cfg Config

	cacheMu sync.Mutex
	cache   map[string]CurrentTrack
}

func NewSpotifyClient(cfg Config) *SpotifyClient {
	return &SpotifyClient{
		Cfg:   cfg,
		cache: make(map[string]CurrentTrack),
	}
}

func (c *SpotifyClient) currentlyPlaying(ctx context.Context, user User) (CurrentTrack, error) {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
	client := c.Cfg.Oauth2Cfg.Client(ctx, user.Token)
	res, err := client.Do(req)
	if err != nil {
		return CurrentTrack{}, fmt.Errorf("error fetching currently-playing: %w", err)
	}
	if res.StatusCode == 204 { // no content
		return CurrentTrack{}, nil
	}
	if res.StatusCode != 200 {
		return CurrentTrack{}, fmt.Errorf("received non-200 code from currently-playing: %d", res.StatusCode)
	}

	ct := CurrentTrack{}
	if err := json.NewDecoder(res.Body).Decode(&ct); err != nil {
		return CurrentTrack{}, fmt.Errorf("error decoding currently-playing response: %w", err)
	}
	ct.Fetched = time.Now().Unix()
	return ct, nil
}

func (c *SpotifyClient) CurrentlyPlaying(ctx context.Context, user User) (CurrentTrack, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	cached, ok := c.cache[user.ID]
	if ok && !c.shouldRefreshCurrentTrack(cached) {
		// cached, return early
		return cached, nil
	}
	// fetch new data
	ct, err := c.currentlyPlaying(ctx, user)
	if err != nil {
		return CurrentTrack{}, err
	}
	c.cache[user.ID] = ct
	return ct, nil
}

func (c *SpotifyClient) shouldRefreshCurrentTrack(t CurrentTrack) bool {
	return time.Unix(t.Fetched, 0).Add(currentTrackTTL).Before(time.Now())
}
