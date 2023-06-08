package tobab

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/asaskevich/govalidator"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/logrusorgru/aurora"
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
	CertDir         string `valid:"required"`
	Email           string `valid:"email"`
	Staging         bool
	Loglevel        string
	DatabasePath    string `valid:"required"`
	AdminGlobs      []Glob `valid:"required"`
}

type Host struct {
	Hostname string `storm:"id" valid:"dns"`
	Backend  string `valid:"required"`
	Type     string `valid:"required"`
	Public   bool
	Globs    []Glob
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

func (h *Host) Print() {
	fmt.Printf(`
> %s
Backend: %s
Type: %s
Public: %t
Globs: %s
`, aurora.Magenta(aurora.Bold(h.Hostname)), h.Backend, h.Type, h.Public, h.Globs)
}

func (h *Host) Validate(cookiescope string) (bool, error) {
	ok, err := govalidator.ValidateStruct(h)
	if !ok {
		return ok, err
	}
	if h.Type != "http" {
		return false, errors.New("host type must be http")
	}
	u, err := url.ParseRequestURI(h.Backend)
	if err != nil {
		return false, fmt.Errorf("%s failed to parse as a url: %w", h.Backend, err)
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		return false, fmt.Errorf("%s has invalid or missing scheme", h.Backend)
	}
	if !strings.HasSuffix(h.Hostname, cookiescope) && !h.Public {
		return false, fmt.Errorf("'%s' won't be accessible because the cookiescope ('%s') does not match this domain", h.Hostname, cookiescope)
	}
	if !h.Public && len(h.Globs) == 0 {
		return false, fmt.Errorf("%s will not be accessible by anybody", h.Hostname)
	}

	return ok, err
}

type Glob string

func (g Glob) Match(s string) bool {
	return matcher.Glob(string(g), s)
}

func (h Host) HasAccess(user string) bool {

	if h.Public {
		return true
	} else if user == "" {
		return false
	}

	for _, g := range h.Globs {
		if g.Match(user) {
			return true
		}
	}

	return false
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
