package main

import (
	"html/template"
	"log/slog"
	"os"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/storm"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var version = "manual build"

type Tobab struct {
	fqdn       string
	config     tobab.Config
	logger     *slog.Logger
	maxAge     time.Duration
	defaultAge time.Duration
	templates  *template.Template
	confLoc    string
	db         tobab.Database
	webauthn   *webauthn.WebAuthn
}

func main() {

	logger := slog.With("", "")

	confLoc := os.Getenv("TOBAB_TOML")
	if confLoc == "" {
		confLoc = "/etc/tobab/tobab.toml"
	}

	cfg, err := tobab.LoadConf(confLoc)
	if err != nil {
		logger.Error("Failed loading config", "error", err)
		return
	}

	if version == "" {
		version = "unknown"
	}

	db, err := storm.New(cfg.DatabasePath)
	if err != nil {
		logger.Error("unable to init database", "error", err, "location", cfg.DatabasePath)
	}
	defer db.Close()

	fqdn := "https://" + cfg.Hostname
	if cfg.Dev {
		fqdn = "http://localhost:8080"
	}

	wconfig := &webauthn.Config{
		RPDisplayName:         cfg.Displayname,
		RPID:                  cfg.CookieScope,
		RPOrigins:             []string{fqdn},
		AttestationPreference: protocol.PreferNoAttestation,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		logger.Error("Unable to initialize webauthn", "error", err)
	}

	app := Tobab{
		config:   cfg,
		logger:   logger.With("version", version),
		maxAge:   720 * time.Hour,
		fqdn:     fqdn,
		confLoc:  confLoc,
		db:       db,
		webauthn: w,
	}

	//check if admin is created already, otherwise set it to false
	hasAdmin, err := app.db.KVGetBool(ADMIN_REGISTERED_KEY)
	if err != nil || !hasAdmin {
		logger.Warn("Setting flag so first user to register will be admin")
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
		logger.Error("unable to load templates", "error", err)
		return
	}

	go app.cleanSessionsLoop()

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
		r.SetFuncMap(templateFunctions)
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
		app.logger.Error("Failed to start web server")
	}
}

func (app *Tobab) getHosts() []string {
	var hosts []string
	err := app.db.KVGet("hosts", &hosts)
	if err != nil {
		app.logger.Error("Failed to get hosts", "error", err)
	}
	return hosts
}

func (app *Tobab) addHost(h string) {
	var hosts []string

	err := app.db.KVGet("hosts", &hosts)
	if err != nil {
		app.logger.Error("Failed to get hosts", "error", err)
	}

	if tobab.Contains(hosts, h) {
		return
	}

	hosts = append(hosts, h)
	err = app.db.KVSet("hosts", hosts)
	if err != nil {
		app.logger.Error("Failed to set hosts", "error", err)
	}
}

func (app *Tobab) cleanSessionsLoop() {
	time.Sleep(2 * time.Second)
	for {
		app.logger.Info("cleaning old sessions")
		app.db.CleanupOldSessions()
		time.Sleep(time.Hour)
	}
}
