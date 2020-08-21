package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/ebracho/spotifysync"

	"golang.org/x/crypto/acme/autocert"
)

func serveLocal() {
	cfg, err := spotifysync.NewConfigFromFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	ss := spotifysync.SpotifySyncServer{
		SpotifyClient: spotifysync.NewSpotifyClient(cfg),
		Cfg:           cfg,
	}
	ss.RegisterHandlers()
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		log.Fatal(err)
	}
}

func serveProd() {
	cfg, err := spotifysync.NewConfigFromFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	ss := spotifysync.SpotifySyncServer{
		SpotifyClient: spotifysync.NewSpotifyClient(cfg),
		Cfg:           cfg,
	}
	ss.RegisterHandlers()

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("example.com"), //Your domain here
		Cache:      autocert.DirCache("certs"),            //Folder for storing certificates
	}
	server := &http.Server{
		Addr:    ":https",
		Handler: http.DefaultServeMux,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}
	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if _, err := os.Stat("certs"); err == nil {
		serveProd()
	} else {
		serveLocal()
	}
}
