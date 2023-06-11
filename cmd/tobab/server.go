package main

import (
	"html/template"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/storm"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sirupsen/logrus"
)

var version = "manual build"

type Tobab struct {
	fqdn       string
	config     tobab.Config
	logger     *logrus.Entry
	maxAge     time.Duration
	defaultAge time.Duration
	templates  *template.Template
	confLoc    string
	db         tobab.Database
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
	app.startServer()

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

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(app.getSessionMiddleware())
	app.setTobabRoutes(r)

	err := r.Run()
	if err != nil {
		app.logger.WithError(err).Error("Failed to start web server")
	}
}
