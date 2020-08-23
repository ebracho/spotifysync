package spotifysync

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

type SpotifySyncServer struct {
	SpotifyClient *SpotifyClient
	Cfg           *Config
}

func (s *SpotifySyncServer) CurrentTrack(w http.ResponseWriter, req *http.Request) {
	userID := req.URL.Query().Get("user")
	if userID == "" {
		http.Error(w, "missing query param 'user'", http.StatusBadRequest)
		return
	}
	s.Cfg.Lock()
	user, ok := s.Cfg.RegisteredUsers[userID]
	s.Cfg.Unlock()
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

// initiate spotify 3-legged oauth
func (s *SpotifySyncServer) Login(w http.ResponseWriter, req *http.Request) {
	state := generateRandomState()
	http.SetCookie(w, &http.Cookie{
		Name:    "state",
		Value:   state,
		Expires: time.Now().Add(10 * time.Minute),
	})
	s.Cfg.Lock()
	http.Redirect(w, req, s.Cfg.Oauth2Cfg.AuthCodeURL(state), http.StatusSeeOther)
	s.Cfg.Unlock()
}

// spotify oauth2 callback
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

	s.Cfg.Lock()
	token, err := s.Cfg.Oauth2Cfg.Exchange(req.Context(), code)
	s.Cfg.Unlock()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("error exchanging code for token: %s\n", err)
		return
	}

	setTokenCookies(w, token)

	next, err := req.Cookie("nextSyncUser")
	if next == nil || err != nil {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, req, fmt.Sprintf("/sync?user=%s", next.Value), http.StatusSeeOther)

	// Fetch user info to check if we need to register them as a sharer
	user, err := s.SpotifyClient.UserFromToken(req.Context(), token)
	if err != nil {
		log.Printf("ERROR fetching user: %s\n", err)
		return
	}
	s.Cfg.Lock()
	defer s.Cfg.Unlock()
	if !stringSliceContains(s.Cfg.PermittedSharers, user.DisplayName) {
		return
	}
	if _, ok := s.Cfg.RegisteredUsers[user.DisplayName]; !ok {
		s.Cfg.RegisteredUsers[user.DisplayName] = user
		s.Cfg.Save()
	}
}

func (s *SpotifySyncServer) PermitSharer(w http.ResponseWriter, req *http.Request) {
	sharer := req.URL.Query().Get("sharer")
	if sharer == "" {
		http.Error(w, "missing query param 'sharer'", http.StatusBadRequest)
		return
	}
	token, err := tokenFromCookies(req)
	if err != nil {
		http.Error(w, "must be logged in", http.StatusUnauthorized)
		log.Printf("ERROR registering sharer: %s\n", err)
		return
	}
	user, err := s.SpotifyClient.UserFromToken(req.Context(), token)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("ERROR registering sharer: %s\n", err)
		return
	}
	s.Cfg.Lock()
	defer s.Cfg.Unlock()
	if !stringSliceContains(s.Cfg.Admins, user.DisplayName) {
		http.Error(w, "must be an admin", http.StatusUnauthorized)
		fmt.Printf("%v, %v", s.Cfg.Admins, user.DisplayName)
		return
	}
	if stringSliceContains(s.Cfg.PermittedSharers, sharer) {
		fmt.Fprintf(w, "%s already permitted", sharer)
		return
	}
	s.Cfg.PermittedSharers = append(s.Cfg.PermittedSharers, sharer)
	s.Cfg.Save()
	fmt.Fprintf(w, "permitted %s to register as a sharer", sharer)
}

func (s *SpotifySyncServer) Sync(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "./static/sync.html")
}

func (s *SpotifySyncServer) Home(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello world")
}

func (s *SpotifySyncServer) RegisterHandlers() {
	http.HandleFunc("/", s.Home)
	http.HandleFunc("/sync", s.Sync)
	http.HandleFunc("/currentTrack", s.CurrentTrack)
	http.HandleFunc("/login", s.Login)
	http.HandleFunc("/spotifyCallback", s.Callback)
	http.HandleFunc("/permitSharer", s.PermitSharer)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}

func setTokenCookies(w http.ResponseWriter, token *oauth2.Token) {
	http.SetCookie(w, &http.Cookie{
		Name:  "access_token",
		Value: token.AccessToken,
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "refresh_token",
		Value: token.RefreshToken,
	})
	log.Println(token.Expiry.Unix())
	http.SetCookie(w, &http.Cookie{
		Name:  "expiry",
		Value: strconv.Itoa(int(token.Expiry.Unix())),
	})
}

func tokenFromCookies(req *http.Request) (*oauth2.Token, error) {
	accessTokenCookie, err := req.Cookie("access_token")
	if err != nil {
		return nil, fmt.Errorf("no logged in user")
	}
	refreshTokenCookie, err := req.Cookie("refresh_token")
	if err != nil {
		return nil, fmt.Errorf("no logged in user")
	}
	expiryCookie, err := req.Cookie("expiry")
	if err != nil {
		return nil, fmt.Errorf("no logged in user")
	}
	expiryUnixSeconds, err := strconv.Atoi(expiryCookie.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token expiry")
	}
	return &oauth2.Token{
		AccessToken:  accessTokenCookie.Value,
		RefreshToken: refreshTokenCookie.Value,
		Expiry:       time.Unix(int64(expiryUnixSeconds), 0),
	}, nil
}

func generateRandomState() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func stringSliceContains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
