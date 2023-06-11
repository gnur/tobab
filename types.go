package tobab

import (
	"fmt"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/asaskevich/govalidator"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/looplab/fsm"
	matcher "github.com/ryanuber/go-glob"
)

type Config struct {
	Hostname        string `valid:"dns"`
	Dev             bool
	Displayname     string `valid:"required"`
	DefaultTokenAge string
	MaxTokenAge     string
	CookieScope     string `valid:"required"`
	Secret          string `valid:"required"`
	Salt            string `valid:"required"`
	Loglevel        string
	DatabasePath    string `valid:"required"`
	AdminGlobs      []Glob `valid:"required"`
}

type Host struct {
	Hostname string `storm:"id" valid:"dns"`
	Public   bool
}

type User struct {
	ID                   []byte `storm:"id"`
	Name                 string `storm:"unique"`
	RegistrationFinished bool
	Created              time.Time
	LastSeen             time.Time
	Admin                bool
	Creds                []webauthn.Credential
}

func (user *User) WebAuthnID() []byte {
	return user.ID
}

func (user *User) WebAuthnName() string {
	return user.Name
}

func (user *User) WebAuthnDisplayName() string {
	return user.Name
}

func (user *User) WebAuthnIcon() string {
	return "https://pics.com/avatar.png"
}

func (user *User) WebAuthnCredentials() []webauthn.Credential {
	return user.Creds
}

type Session struct {
	ID       string `storm:"id"`
	UserID   []byte
	Created  time.Time
	LastSeen time.Time
	Expires  time.Time `storm:"index"`
	Vals     map[string]string
	Data     *webauthn.SessionData
	FSM      *fsm.FSM
	State    string
}

type Glob string

func (g Glob) Match(s string) bool {
	return matcher.Glob(string(g), s)
}

func (c *Config) Validate() (bool, error) {
	ok, err := govalidator.ValidateStruct(c)
	if !ok {
		return ok, err
	}

	if !strings.HasSuffix(c.Hostname, c.CookieScope) {
		return false, fmt.Errorf("Hostname: '%s' should be in the same domain as the cookiescope: '%s'", c.Hostname, c.CookieScope)
	}

	return ok, err
}

func LoadConf(path string) (Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return cfg, err
	}

	ok, err := cfg.Validate()
	if !ok {
		return cfg, err
	}
	return cfg, err

}
