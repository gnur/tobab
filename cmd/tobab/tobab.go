package main

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/clirpc"
	"github.com/go-webauthn/webauthn/protocol"
	_ "github.com/go-webauthn/webauthn/protocol"
	"github.com/lithammer/shortuuid"
	"github.com/sirupsen/logrus"
)

type RegistrationStart struct {
	Name string
}

func (app *Tobab) setTobabRoutes(r *gin.Engine) {

	pk := r.Group("/passkey/")
	pklog := app.logger.WithField("method", "passkey")

	pk.POST("/register/start", func(c *gin.Context) {
		var regStart RegistrationStart

		err := c.BindJSON(&regStart)
		if err != nil {
			pklog.WithError(err).Warning("failed to parse body")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		user, ok := c.Get("USER")
		if ok {
			//this is kind of unsafe, but if it is found I will assume it is of the correct type as well
			u := user.(*tobab.User)
			if u.RegistrationFinished {
				pklog.WithField("username", u.Name).Warning("user that alread exists in session is trying to register")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}

		u, err := app.db.GetUserByName(regStart.Name)
		if err == nil && u.RegistrationFinished {
			pklog.WithField("username", u.Name).Warning("user that alread exists in db is trying to register")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		uid := shortuuid.New()
		u = &tobab.User{
			ID:      []byte(uid),
			Name:    regStart.Name,
			Created: time.Now(),
		}

		err = app.db.SetUser(*u)
		if err != nil {
			pklog.WithError(err).Error("failed to save new user in registration start")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		options, session, err := app.webauthn.BeginRegistration(u)
		pklog.WithFields(logrus.Fields{
			"options": options,
			"session": session,
		}).Debug("Started webauthn registration")
		if err != nil {
			pklog.WithError(err).Error("failed to start webauthn registration")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		rawSession, ok := c.Get("SESSION")
		if !ok {
			pklog.WithError(err).Error("failed to get session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess := rawSession.(*tobab.Session)
		sess.Data = session
		sess.UserID = u.ID

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})
	pk.POST("/register/finish", func(c *gin.Context) {
		resp, err := protocol.ParseCredentialCreationResponseBody(c.Request.Body)
		if err != nil {
			pklog.WithError(err).Error("failed to parse credential body")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		rawSession, ok := c.Get("SESSION")
		if !ok {
			pklog.WithError(err).Error("failed to get session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess := rawSession.(*tobab.Session)
		webSess := sess.Data

		user, err := app.db.GetUser(sess.UserID)
		if err != nil {
			pklog.WithError(err).Error("failed to retrieve user from session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		credential, err := app.webauthn.CreateCredential(user, *webSess, resp)
		if err != nil {
			pklog.WithError(err).Error("failed to create credential")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user.Creds = append(user.Creds, *credential)
		err = app.db.SetUser(*user)
		if err != nil {
			pklog.WithError(err).Error("failed to store credential with user")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatus(http.StatusOK)
	})

	pk.POST("/login/start", func(c *gin.Context) {})
	pk.POST("/login/finish", func(c *gin.Context) {})

	api := r.Group("/v1/api/")
	api.Use(app.adminMiddleware())

	//GET hosts
	api.GET("/hosts", func(c *gin.Context) {
		hosts, err := app.db.GetHosts()
		if err != nil {
			app.logger.WithError(err).Error("Failed getting hosts from db")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatusJSON(200, hosts)
	})

	//ADD host
	api.POST("/host", func(c *gin.Context) {
		var h tobab.Host
		err := c.BindJSON(&h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to unmarshal host from body")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if ok, err := h.Validate(app.config.CookieScope); !ok {
			app.logger.WithError(err).Error("Invalid hostconfig provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err = app.db.AddHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to add host to database")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatusJSON(202, h)
		go app.restartServer()

	})

	//DELETE host
	api.DELETE("/host/:hostname", func(c *gin.Context) {
		h := c.Param("hostname")
		if h == "" {
			app.logger.Error("No hostname provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err := app.db.DeleteHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to delete host from database")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatus(202)
		go app.restartServer()

	})

	r.GET("/", func(c *gin.Context) {
		providerIndex := &ProviderIndex{Providers: []string{"google"}, ProvidersMap: map[string]string{"google": "Google"}}

		user, err := app.extractUser(c.Request)
		if err == nil {
			providerIndex.User = user
		} else {
			app.logger.WithError(err).Error("unable to get user from request")
		}

		c.HTML(200, "index.html", providerIndex)
	})

	r.StaticFS("/static", mustFS())
}

func mustFS() http.FileSystem {
	sub, _ := fs.Sub(staticFS, "static")
	return http.FS(sub)
}

type ProviderIndex struct {
	Providers    []string
	ProvidersMap map[string]string
	User         string
}

func (app *Tobab) GetHosts(in *clirpc.Empty, out *clirpc.GetHostsOut) error {
	h, err := app.db.GetHosts()
	out.Hosts = h
	return err
}

func (app *Tobab) AddHost(in *clirpc.AddHostIn, out *clirpc.Empty) error {
	ok, err := in.Host.Validate(app.config.CookieScope)
	if !ok {
		return err
	}
	err = app.db.AddHost(in.Host)
	if err == nil {
		go app.restartServer()
	}
	return err
}

func (app *Tobab) DeleteHost(in *clirpc.DeleteHostIn, out *clirpc.Empty) error {
	err := app.db.DeleteHost(in.Hostname)
	if err == nil {
		go app.restartServer()
	}
	return err
}
