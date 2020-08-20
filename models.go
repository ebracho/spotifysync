package spotifysync

import (
	"golang.org/x/oauth2"
)

type User struct {
	ID    string `json:"ID"`
	Token *oauth2.Token
}

type Config struct {
	RegisteredUsers map[string]User `json:"registeredUsers"`
	Oauth2Cfg       oauth2.Config
	ListenAddress   string
}
