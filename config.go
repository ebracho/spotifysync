package spotifysync

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

type User struct {
	ID    string        `json:"id"` // Spotify user id
	Token *oauth2.Token // Spotify oauth token
}

type Config struct {
	sync.Mutex
	Admins           []string        // Spotify user ids of people with admin priviliges
	PermittedSharers []string        // Spotify user ids of people permitted to register as sharers
	RegisteredUsers  map[string]User `json:"registeredUsers"` // Users who are permitted to share music
	Oauth2Cfg        oauth2.Config
	ListenAddress    string
	Host             string
	Filepath         string `json:"-"`
}

func NewConfigFromFile(fp string) (*Config, error) {
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg := Config{
		Filepath:        fp,
		RegisteredUsers: make(map[string]User),
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	bytes, err := json.MarshalIndent(c, "", "	")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.Filepath, bytes, 0644)
}
