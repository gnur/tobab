package main

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/go-webauthn/webauthn/protocol"
	_ "github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/lithammer/shortuuid"
	"github.com/sirupsen/logrus"
)

const ADMIN_REGISTERED_KEY = "admin_registered"

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

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		if sess.State == "registration" {
			sess.FSM.Event(c, "finishRegistration")
			sess.State = sess.FSM.Current()
		}

		if sess.FSM.Current() != "null" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		u, err := app.db.GetUserByName(regStart.Name)
		if err == nil {
			pklog.WithField("username", u.Name).Warning("user that already exists in db is trying to register")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"msg": "user already exists",
			})
			return
		}

		uid := shortuuid.New()
		u = &tobab.User{
			ID:       []byte(uid),
			Name:     regStart.Name,
			Created:  time.Now(),
			LastSeen: time.Now(),
		}

		err = app.db.SetUser(*u)
		if err != nil {
			pklog.WithError(err).Error("failed to save new user in registration start")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		options, session, err := app.webauthn.BeginRegistration(u, webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired))
		pklog.WithFields(logrus.Fields{
			"options": options,
			"session": session,
		}).Debug("Started webauthn registration")
		if err != nil {
			pklog.WithError(err).Error("failed to start webauthn registration")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.Data = session
		sess.UserID = u.ID

		err = sess.FSM.Event(c, "startRegistration")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})
	pk.POST("/register/finish", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		defer func() {
			sess.Data = &webauthn.SessionData{}
			app.db.SetSession(*sess)
		}()

		if sess.FSM.Current() != "registration" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		resp, err := protocol.ParseCredentialCreationResponseBody(c.Request.Body)
		if err != nil {
			pklog.WithError(err).Error("failed to parse credential body")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

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

		hasAdmin, err := app.db.KVGetBool(ADMIN_REGISTERED_KEY)
		if err == nil && !hasAdmin {
			user.Admin = true
			app.db.KVSet(ADMIN_REGISTERED_KEY, true)
		}

		user.Creds = append(user.Creds, *credential)
		user.RegistrationFinished = true
		err = app.db.SetUser(*user)
		if err != nil {
			pklog.WithError(err).Error("failed to store credential with user")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = sess.FSM.Event(c, "finishRegistration")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatus(http.StatusOK)
	})

	pk.POST("/login/anystart", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		if sess.State == "login" {
			sess.FSM.Event(c, "loginFail")
			sess.State = sess.FSM.Current()
		}

		if sess.State != "null" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		options, session, err := app.webauthn.BeginDiscoverableLogin()

		if err != nil {
			pklog.WithError(err).Error("failed to start webauthn login")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.Data = session

		err = sess.FSM.Event(c, "startLogin")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})
	pk.POST("/login/anyfinish", func(c *gin.Context) {

		var loginStart RegistrationStart

		err := c.BindJSON(&loginStart)
		if err != nil {
			pklog.WithError(err).Warning("failed to parse body")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		defer func() {
			sess.Data = &webauthn.SessionData{}
			app.db.SetSession(*sess)
		}()

		if sess.FSM.Current() != "login" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		resp, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
		if err != nil {
			pklog.WithError(err).Error("failed to parse credential body")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess := sess.Data

		user, err := app.db.GetUserByName(loginStart.Name)
		if err != nil {
			pklog.WithError(err).Error("failed to retrieve user from session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		credential, err := app.webauthn.ValidateLogin(user, *webSess, resp)
		if err != nil {
			pklog.WithError(err).Error("failed to validate login")
			c.AbortWithStatus(403)
			return
		}

		err = sess.FSM.Event(c, "loginSuccess")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.UserID = user.WebAuthnID()

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		pklog.WithField("cred", credential.ID).Debug("success logging in!")

		c.AbortWithStatus(http.StatusOK)

	})

	pk.POST("/login/start", func(c *gin.Context) {
		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		if sess.State == "login" {
			sess.FSM.Event(c, "loginFail")
			sess.State = sess.FSM.Current()
		}

		if sess.State != "null" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		var loginStart RegistrationStart

		err := c.BindJSON(&loginStart)
		if err != nil {
			pklog.WithError(err).Warning("failed to parse body")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		u, err := app.db.GetUserByName(loginStart.Name)
		if err != nil {
			pklog.WithField("username", u.Name).Warning("unknown username provided")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"msg": "unknown user",
			})
			return
		}

		options, session, err := app.webauthn.BeginLogin(u)
		if err != nil {
			pklog.WithError(err).Error("failed to start webauthn login")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.Data = session
		sess.UserID = u.ID

		err = sess.FSM.Event(c, "startLogin")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})
	pk.POST("/login/finish", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.WithField("session_id", sess.ID).Debug("using session")

		if sess.FSM.Current() != "login" {
			pklog.WithField("state", sess.FSM.Current()).Warning("invalid source state for this request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		resp, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
		if err != nil {
			pklog.WithError(err).Error("failed to parse credential body")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess := sess.Data

		user, err := app.db.GetUser(resp.Response.UserHandle)
		if err != nil {
			pklog.WithError(err).Error("failed to retrieve user from session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess.UserID = user.WebAuthnID()

		credential, err := app.webauthn.ValidateLogin(user, *webSess, resp)
		if err != nil {
			pklog.WithError(err).Error("failed to validate login")
			c.AbortWithStatus(403)
			return
		}

		err = sess.FSM.Event(c, "loginSuccess")
		if err != nil {
			pklog.WithError(err).Error("failed to transition state")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.UserID = user.WebAuthnID()

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.WithError(err).Error("failed to save session")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		pklog.WithField("cred", credential.ID).Debug("success logging in!")

		c.AbortWithStatus(http.StatusOK)

	})

	api := r.Group("/v1/api/")
	api.Use(app.adminMiddleware())

	r.GET("/register.html", func(c *gin.Context) {

		var user *tobab.User
		var err error

		sess := app.getSession(c.GetString("SESSION_ID"))

		if sess.State == "authenticated" {
			user, err = app.db.GetUser(sess.UserID)
			if err != nil {
				pklog.WithError(err).Error("failed to retrieve user from session")
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.Redirect(307, "/")
		}

		if sess.State != "null" {
			if sess.State == "login" {
				err = sess.FSM.Event(c, "loginFail")
				if err != nil {
					pklog.WithError(err).Error("failed to transition state")
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				err = app.db.SetSession(*sess)
				if err != nil {
					pklog.WithError(err).Error("failed to save session")
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
			}
		}

		name := "unknown"
		if user != nil {
			name = user.Name
		}

		c.HTML(200, "register.html", indexVars{
			State:    sess.State,
			Username: name,
		})
	})

	r.GET("/clean-sessions", func(c *gin.Context) {

		app.db.CleanupOldSessions()

		c.AbortWithStatus(202)
	})

	r.GET("/register", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		sess.Expires = time.Now().Add(-2 * app.maxAge)
		app.db.SetSession(*sess)

		c.SetCookie(COOKIE_NAME, "", -1, "/", app.config.CookieScope, true, true)
		c.Redirect(307, "/register.html")
	})

	r.GET("/signout", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		sess.Expires = time.Now().Add(-2 * app.maxAge)
		app.db.SetSession(*sess)

		c.SetCookie(COOKIE_NAME, "", -1, "/", app.config.CookieScope, true, true)
		c.Redirect(307, "/")
	})

	r.GET("/", func(c *gin.Context) {

		var user *tobab.User
		var err error
		sess := app.getSession(c.GetString("SESSION_ID"))

		if sess.State == "authenticated" {
			user, err = app.db.GetUser(sess.UserID)
			if err != nil {
				pklog.WithError(err).Error("failed to retrieve user from session")
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

		}
		name := "unknown"
		if user != nil {
			name = user.Name
		}

		c.HTML(200, "index.html", indexVars{
			State:    sess.State,
			Username: name,
		})
	})

	r.StaticFS("/static", app.mustFS())
}

type indexVars struct {
	State string

	Username string
}

func (app *Tobab) mustFS() http.FileSystem {
	if app.config.Dev {
		return http.Dir("cmd/tobab/static")
	}
	sub, _ := fs.Sub(staticFS, "static")
	return http.FS(sub)
}
