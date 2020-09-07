package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

var salt = []byte("kcqbBue2Sr7U5yrpEaZFpGVdR6z4jfUeSECy6zuYDXktgxhFCxMtEkV9")

type Tobab struct {
	fqdn   string
	key    []byte
	config Config
	logger *logrus.Entry
	maxAge time.Duration
}

func main() {
	cfg := Config{
		Hostname:    "login.tobab.erwin.land",
		CookieScope: ".erwin.land",
		Secret:      "asldfkjaslkdf",
		CertDir:     "/home/erwin/certs",
		Hosts: map[string]Host{
			"tobab.erwin.land": {
				Backend: "https://postman-echo.com",
				Type:    "http",
			},
			"echo.tobab.erwin.land": {
				Backend: "http://httpbin.org",
				Type:    "http",
			},
			"ip.tobab.erwin.land": {
				Backend:      "https://ifconfig.co",
				Type:         "http",
				AllowedGlobs: []string{"admin"},
			},
		},
		Globs: map[string]string{
			"admin": "*@gnur.nl",
		},
		Email:   "test@dekeijzer.xyz",
		Staging: true,
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = cfg.Email

	if cfg.Staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}

	key := argon2.IDKey([]byte(cfg.Secret), salt, 4, 4*1024, 2, 32)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	app := Tobab{
		key:    key,
		config: cfg,
		logger: logger.WithField("source", "tobab"),
		maxAge: 720 * time.Hour,
		fqdn:   "https://" + cfg.Hostname,
	}

	r := mux.NewRouter()
	hosts := []string{app.config.Hostname}

	for h, conf := range cfg.Hosts {
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
	r.Use(handlers.CompressHandler)
	r.Use(app.getRBACMiddleware())

	//log.Fatal(autotls.RunWithManager(r, &m))
	err := certmagic.HTTPS(hosts, r)
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
