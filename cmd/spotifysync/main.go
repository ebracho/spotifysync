package main

import (
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ebracho/spotifysync"
)

func main() {
	cfg := spotifysync.Config{
		Oauth2Cfg: oauth2.Config{
			ClientID:     os.Getenv("CLIENT_ID"),
			ClientSecret: os.Getenv("CLIENT_SECRET"),
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://accounts.spotify.com/authorize",
				TokenURL: "https://accounts.spotify.com/api/token",
			},
			RedirectURL: "http://localhost:8999/spotifyCallback",
			Scopes: []string{
				"user-read-currently-playing",
			},
		},
		RegisteredUsers: map[string]spotifysync.User{
			"ebracho": spotifysync.User{
				ID: "ebracho",
				Token: &oauth2.Token{
					AccessToken:  os.Getenv("ACCESS_TOKEN"),
					RefreshToken: os.Getenv("REFRESH_TOKEN"),
					Expiry:       time.Now(), // force a refresh on startup
				},
			},
		},
		ListenAddress: "localhost:8999",
	}
	srv := spotifysync.SpotifySyncServer{
		SpotifyClient: &spotifysync.SpotifyClient{
			Cfg: cfg,
		},
		Cfg: cfg,
	}
	srv.RegisterHandlers()
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
