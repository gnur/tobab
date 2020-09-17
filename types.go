package tobab

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/asaskevich/govalidator"
	"github.com/logrusorgru/aurora"
	matcher "github.com/ryanuber/go-glob"
)

type Config struct {
	Hostname        string `valid:"dns"`
	DefaultTokenAge string
	MaxTokenAge     string
	CookieScope     string `valid:"required"`
	Secret          string `valid:"required"`
	Salt            string `valid:"required"`
	CertDir         string `valid:"required"`
	Email           string `valid:"email"`
	Staging         bool
	GoogleKey       string `valid:"required"`
	GoogleSecret    string `valid:"required"`
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

func (h *Host) Print() {
	fmt.Printf(`
> %s
Backend: %s
Type: %s
Public: %t
Globs: %s
`, aurora.Magenta(aurora.Bold(h.Hostname)), h.Backend, h.Type, h.Public, h.Globs)
}

func (h *Host) Validate() (bool, error) {
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
	return govalidator.ValidateStruct(c)
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
