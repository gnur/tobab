package main

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/httputil"
	"net/rpc"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/storm"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

var version = "manual build"

type Tobab struct {
	fqdn       string
	key        []byte
	config     tobab.Config
	logger     *logrus.Entry
	maxAge     time.Duration
	defaultAge time.Duration
	templates  *template.Template
	confLoc    string
	db         tobab.Database
	proxies    map[string]http.Handler
	webauthn   *webauthn.WebAuthn
}

func run(confLoc string) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	cfg, err := tobab.LoadConf(confLoc)
	if err != nil {
		logger.WithError(err).Fatal("Failed loading config")
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
	salt := []byte(cfg.Salt)
	secret := []byte(cfg.Secret)

	//transform provided salt and secret into a 32 byte key that can be used by paseto
	key := argon2.IDKey(secret, salt, 4, 4*1024, 2, 32)

	if version == "" {
		version = "unknown"
	}

	db, err := storm.New(cfg.DatabasePath)
	if err != nil {
		logger.WithError(err).WithField("location", cfg.DatabasePath).Fatal("Unable to initialize database")
	}
	defer db.Close()

	wconfig := &webauthn.Config{
		RPDisplayName: cfg.Displayname,
		RPID:          cfg.CookieScope,
		RPOrigins:     []string{"https://" + cfg.Hostname},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		logger.WithError(err).Fatal("Unable to initialize webauthn")
	}

	app := Tobab{
		key:      key,
		config:   cfg,
		logger:   logger.WithField("version", version),
		maxAge:   720 * time.Hour,
		fqdn:     "https://" + cfg.Hostname,
		confLoc:  confLoc,
		db:       db,
		webauthn: w,
	}

	//check if admin is created already, otherwise set it to false
	hasAdmin, err := app.db.KVGetBool(ADMIN_REGISTERED_KEY)
	if err != nil || !hasAdmin {
		logger.Warning("Setting flag so first user to register will be admin")
		app.db.KVSet(ADMIN_REGISTERED_KEY, false)
	}

	if age, err := time.ParseDuration(cfg.DefaultTokenAge); err != nil {
		app.defaultAge = 720 * time.Hour
	} else {
		app.defaultAge = age
	}

	if age, err := time.ParseDuration(cfg.MaxTokenAge); err != nil {
		app.maxAge = 24 * 365 * time.Hour
	} else {
		app.maxAge = age
	}

	app.templates, err = loadTemplates()
	if err != nil {
		logger.WithError(err).Fatal("unable to load templates")
	}
	go app.startServer()
	go app.startRPCServer()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c
	app.logger.Info("shutting down")
}

func (app *Tobab) startRPCServer() {
	err := rpc.Register(app)
	if err != nil {
		app.logger.WithError(err).Error("Failed to register rpc")
		return
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		app.logger.WithError(err).Error("Failed to start rpc listener")
		return
	}
	err = http.Serve(l, nil)
	if err != nil {
		app.logger.WithError(err).Error("Failed to start rpc http")
		return
	}

}

func (app *Tobab) restartServer() {
	app.setupProxies()
}

func (app *Tobab) setupProxies() {
	app.logger.Debug("loading hosts")
	hosts, err := app.db.GetHosts()
	if err != nil {
		app.logger.WithError(err).Fatal("unable to load hosts")
	}

	app.proxies = make(map[string]http.Handler)
	for _, conf := range hosts {
		if conf.Type != "http" {
			app.logger.WithField("type", conf.Type).Fatal("Unsupported type, currently only http is supported")
		}

		proxy, err := generateProxy(conf.Hostname, conf.Backend)
		if err != nil {
			app.logger.WithError(err).WithField("host", conf.Hostname).Error("Failed creating proxy")
			continue
		}

		app.logger.WithField("host", conf.Hostname).Debug("starting proxy listener")
		app.proxies[conf.Hostname] = proxy
	}

}

func (app *Tobab) startServer() {
	app.logger.Info("starting server")

	if app.config.Dev {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	if app.config.Dev {
		gin.SetMode(gin.DebugMode)
		r.LoadHTMLGlob("cmd/tobab/templates/*.html")
	} else {
		gin.SetMode(gin.ReleaseMode)
		r.SetHTMLTemplate(app.templates)
	}
	certHosts := []string{app.config.Hostname}

	app.setupProxies()

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(app.getSessionMiddleware())
	r.Use(app.getRBACMiddleware())
	r.Use(app.getProxyRouter())
	app.setTobabRoutes(r)

	certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
		DecisionFunc: func(name string) error {
			if !strings.HasSuffix(name, app.config.CookieScope) {
				return fmt.Errorf("not allowed")
			}
			return nil
		},
	}

	go func() {
		err := certmagic.HTTPS(certHosts, r)

		if err != nil {
			if err != http.ErrServerClosed {
				app.logger.WithError(err).Fatal("Failed starting magic listener")
			}
		}
	}()
}

func generateProxy(host, backend string) (http.Handler, error) {
	url, err := url.Parse(backend)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", url.Hostname())
			req.Header.Add("X-Origin-Host", host)
			req.Host = url.Host
			req.URL.Host = url.Host
			req.URL.Scheme = url.Scheme

		}, Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConns:        100,
			Dial: (&net.Dialer{
				Timeout:   600 * time.Second,
				KeepAlive: 300 * time.Second,
			}).Dial,
		}}

	return proxy, nil
}
