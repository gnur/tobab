package main

type Config struct {
	Hostname     string
	CookieScope  string
	Secret       string
	CertDir      string
	Hosts        map[string]Host
	Email        string
	Staging      bool
	Globs        map[string]string
	GoogleKey    string
	GoogleSecret string
	Loglevel     string
}

type Host struct {
	Backend      string
	Type         string
	AllowedGlobs []string
	Public       bool
}
