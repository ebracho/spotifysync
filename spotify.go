package spotifysync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SpotifyClient struct {
	Cfg Config
}

type Track struct {
	URI string `json:"uri"`
}

type CurrentTrack struct {
	Track                Track  `json:"item"`
	IsPlaying            bool   `json:"is_playing"`
	ProgressMS           int    `json:"progress_ms"`
	Timestamp            int64  `json:"timestamp"`
	CurrentlyPlayingType string `json:"currently_playing_type"`
}

func (c *SpotifyClient) CurrentlyPlaying(ctx context.Context, user User) (CurrentTrack, error) {
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
	return ct, nil
}
