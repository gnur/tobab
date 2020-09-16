package tobab

import (
	"github.com/BurntSushi/toml"
	"github.com/asaskevich/govalidator"
	matcher "github.com/ryanuber/go-glob"
)

type Config struct {
	Hostname     string `valid:"dns"`
	CookieScope  string `valid:"required"`
	Secret       string `valid:"required"`
	Salt         string `valid:"required"`
	CertDir      string `valid:"required"`
	Email        string `valid:"email"`
	Staging      bool
	GoogleKey    string `valid:"required"`
	GoogleSecret string `valid:"required"`
	Loglevel     string
	DatabasePath string `valid:"required"`
	AdminGlobs   []Glob `valid:"required"`
}

type Host struct {
	Hostname string `storm:"id"`
	Backend  string
	Type     string
	Public   bool
	Globs    []Glob
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
