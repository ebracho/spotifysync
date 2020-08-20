package spotifysync

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type SpotifySyncServer struct {
	SpotifyClient *SpotifyClient
	Cfg           Config
}

func (s *SpotifySyncServer) CurrentTrack(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("user")
	if userID == "" {
		http.Error(w, "missing query param 'user'", http.StatusBadRequest)
		return
	}
	user, ok := s.Cfg.RegisteredUsers[userID]
	if !ok {
		http.Error(w, "user not registered", http.StatusNotFound)
		return
	}
	currentTrack, err := s.SpotifyClient.CurrentlyPlaying(req.Context(), user)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("ERROR fetching current track for %s: %s", userID, err)
		return
	}
	if err := json.NewEncoder(w).Encode(currentTrack); err != nil {
		log.Printf("ERROR encoding response for current track for %s: %s", userID, err)
		return
	}
}

func (s *SpotifySyncServer) Login(w http.ResponseWriter, req *http.Request) {
	state := generateRandomState()
	http.SetCookie(w, &http.Cookie{
		Name:    "state",
		Value:   state,
		Expires: time.Now().Add(10 * time.Minute),
	})
	http.Redirect(w, req, s.Cfg.Oauth2Cfg.AuthCodeURL(state), http.StatusSeeOther)
}

func (s *SpotifySyncServer) Callback(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code query param", http.StatusBadRequest)
		return
	}
	state := req.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state query param", http.StatusBadRequest)
		return
	}
	storedState, err := req.Cookie("state")
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}
	if state != storedState.Value {
		http.Error(w, "oauth state mismatch", http.StatusUnauthorized)
		return
	}

	token, err := s.Cfg.Oauth2Cfg.Exchange(req.Context(), code)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("error exchanging code for token: %s\n", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "accessToken",
		Value: token.AccessToken,
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "refreshToken",
		Value: token.RefreshToken,
	})

	http.Redirect(w, req, "/", http.StatusSeeOther)
}

func (s *SpotifySyncServer) Home(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello world")
}

func (s *SpotifySyncServer) RegisterHandlers() {
	http.HandleFunc("/", s.Home)
	http.HandleFunc("/currentTrack", s.CurrentTrack)
	http.HandleFunc("/login", s.Login)
	http.HandleFunc("/spotifyCallback", s.Callback)
}

func generateRandomState() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
