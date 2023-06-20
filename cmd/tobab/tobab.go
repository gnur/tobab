package main

import (
	"io/fs"
	"net/http"
	"net/url"
	"time"

	"github.com/asdine/storm"
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

		if url, ok := sess.Vals["redirect_url"]; ok {
			delete(sess.Vals, "redirect_url")
			app.db.SetSession(*sess)
			pklog.Info("redirecting to url")
			c.Redirect(http.StatusFound, url)
			return
		}

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

		c.HTML(200, "register.html", tplVars{
			State:    sess.State,
			Username: name,
		})
	})

	r.GET("/verify", app.verifyForwardAuth)

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

	admin := r.Group("/admin")
	admin.Use(app.adminMiddleware())

	admin.POST("/toggleAccess", func(c *gin.Context) {
		userName := c.Query("user")
		hostName := c.Query("host")

		u, err := app.db.GetUserByName(userName)
		if err != nil {
			app.logger.WithError(err).Warning("invalid username provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		hosts := app.getHosts()
		if !tobab.Contains(hosts, hostName) {
			app.logger.Warning("invalid hostname provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		found := false

		for i, h := range u.AccessibleHosts {
			if h == hostName {
				u.AccessibleHosts = append(u.AccessibleHosts[:i], u.AccessibleHosts[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			u.AccessibleHosts = append(u.AccessibleHosts, hostName)
		}
		err = app.db.SetUser(*u)
		if err != nil {
			app.logger.WithError(err).Warning("Failed to update user")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(200, gin.H{})
	})

	admin.GET("/index.html", func(c *gin.Context) {

		users, err := app.db.GetUsers()
		if err != nil {
			app.logger.WithError(err).Error("failed to retrieve users from database")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		hosts := app.getHosts()

		c.HTML(200, "admin.html", adminVars{
			Users: users,
			Hosts: hosts,
		})
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

		c.HTML(200, "index.html", tplVars{
			State:    sess.State,
			User:     user,
			Username: name,
		})
	})

	r.StaticFS("/static", app.mustFS())
}

type adminVars struct {
	State string
	User  tobab.User

	Users []tobab.User
	Hosts []string
}

type tplVars struct {
	State string
	User  *tobab.User

	Username string
}

func (app *Tobab) mustFS() http.FileSystem {
	if app.config.Dev {
		return http.Dir("cmd/tobab/static")
	}
	sub, _ := fs.Sub(staticFS, "static")
	return http.FS(sub)
}

func (app *Tobab) verifyForwardAuth(c *gin.Context) {
	var user *tobab.User
	var err error

	ll := app.logger.WithField("service", "verify")
	sess := app.getSession(c.GetString("SESSION_ID"))

	host := c.GetHeader("X-Forwarded-Host")
	proto := c.GetHeader("X-Forwarded-Proto")
	uri := c.GetHeader("X-Forwarded-Uri")
	u := "unknown"

	ll = ll.WithFields(logrus.Fields{
		"host":  host,
		"proto": proto,
		"uri":   uri,
		"user":  u,
	})

	app.addHost(host)

	if sess.State != "authenticated" {
		redirect_url, err := url.ParseRequestURI(uri)
		if err != nil {
			redirect_url = &url.URL{}
		}
		redirect_url.Host = host
		redirect_url.Scheme = proto

		sess.Vals["redirect_url"] = redirect_url.String()
		err = app.db.SetSession(*sess)
		if err != nil {
			ll.WithError(err).Error("failed to save session")
		}
		ll.WithField("redirect_url", redirect_url.String()).Info("redirecting to login")

		c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
		return
	}

	user, err = app.db.GetUser(sess.UserID)
	if err != nil && err != storm.ErrNotFound {
		ll.WithError(err).Error("failed to retrieve user from session")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if user == nil {
		c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
		return
	}

	ll = ll.WithFields(logrus.Fields{
		"user": user.Name,
	})

	if user.Admin {
		ll.Info("Return 200 to admin")
		c.AbortWithStatus(200)
		return
	}

	if user.CanAccess(host) {
		ll.Info("Return 200 to user")
		c.AbortWithStatus(200)
		return
	}

	ll.Warning("Return 307 to unknown user")
	c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
}
