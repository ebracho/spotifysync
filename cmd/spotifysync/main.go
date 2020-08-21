package main

import (
	"log"
	"net/http"

	"github.com/ebracho/spotifysync"
)

func main() {
	cfg, err := spotifysync.NewConfigFromFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		log.Println(err)
	}
	srv := spotifysync.SpotifySyncServer{
		SpotifyClient: spotifysync.NewSpotifyClient(cfg),
		Cfg:           cfg,
	}
	srv.RegisterHandlers()
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
