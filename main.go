package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/caddyserver/certmagic"
	"github.com/gnur/tobab/muxlogger"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

var salt = []byte("kcqbBue2Sr7U5yrpEaZFpGVdR6z4jfUeSECy6zuYDXktgxhFCxMtEkV9")

type Tobab struct {
	fqdn      string
	key       []byte
	config    Config
	logger    *logrus.Entry
	maxAge    time.Duration
	templates *template.Template
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	confLoc := os.Getenv("TOBAB_CONFIG")
	if confLoc == "" {
		confLoc = "./tobab.toml"
	}

	var cfg Config
	_, err := toml.DecodeFile(confLoc, &cfg)
	if err != nil {
		logger.WithError(err).Fatal("Could not read config")
	}

	if lvl, err := logrus.ParseLevel(cfg.Loglevel); err == nil {
		logger.SetLevel(lvl)
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = cfg.Email

	if cfg.Staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}

	//only use provided salt if it makes any sense at all
	//otherwise use the default salt, shouldn't be a problem
	if len(cfg.Salt) > 2 {
		salt = []byte(cfg.Salt)
	}

	secret := []byte(cfg.Secret)
	//only use provided secret if is it provided
	if len(secret) < 1 {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			logger.WithError(err).Fatal("unable to generate secure secret, please provide it in config")
		}
		secret = b
	}

	//set secret that goth uses
	os.Setenv("SESSION_SECRET", string(secret))

	//transform provided salt and secret into a 32 byte key that can be used by paseto
	key := argon2.IDKey(secret, salt, 4, 4*1024, 2, 32)

	app := Tobab{
		key:    key,
		config: cfg,
		logger: logger.WithField("source", "tobab"),
		maxAge: 720 * time.Hour,
		fqdn:   "https://" + cfg.Hostname,
	}

	app.templates, err = loadTemplates()
	if err != nil {
		logger.WithError(err).Fatal("unable to load templates")
	}

	r := mux.NewRouter()
	hosts := []string{app.config.Hostname}

	for h, conf := range cfg.Hosts {
		if conf.Type != "http" {
			app.logger.WithField("type", conf.Type).Fatal("Unsupported type, currently only http is supported")
		}

		proxy, err := generateProxy(h, conf.Backend)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.Host(h).PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})
		hosts = append(hosts, h)
	}

	if _, ok := cfg.Hosts[app.config.Hostname]; !ok {
		cfg.Hosts[app.config.Hostname] = Host{
			Public: true,
		}
	}

	tobabRoutes := r.Host(app.config.Hostname).Subrouter()
	app.setupgoth(tobabRoutes)

	r.Use(muxlogger.NewLogger(app.logger).Middleware)
	r.Use(handlers.CompressHandler)
	r.Use(app.getRBACMiddleware())

	err = certmagic.HTTPS(hosts, r)
	if err != nil {
		fmt.Println(err)
	}
}

func generateProxy(host, backend string) (http.Handler, error) {
	url, err := url.Parse(backend)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{Director: func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", url.Hostname())
		req.Header.Add("X-Origin-Host", host)
		req.Host = url.Host

		url.Path = req.URL.Path
		req.URL = url

	}, Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}}

	return proxy, nil
}
