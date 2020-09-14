package tobab

type Config struct {
	Hostname     string
	CookieScope  string
	Secret       string
	Salt         string
	CertDir      string
	Hosts        map[string]Host
	Email        string
	Staging      bool
	Globs        []Glob
	GoogleKey    string
	GoogleSecret string
	Loglevel     string
	AdminGlobs   []string
}

type Host struct {
	Hostname     string
	Backend      string
	Type         string
	AllowedGlobs []string
	Public       bool
}

type Glob struct {
	Name    string
	Matcher string
}
