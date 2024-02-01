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
)

const ADMIN_REGISTERED_KEY = "admin_registered"

type RegistrationStart struct {
	Name string
}

func (app *Tobab) setTobabRoutes(r *gin.Engine) {

	pk := r.Group("/passkey/")
	pklog := app.logger.With("method", "passkey")

	pk.POST("/register/start", func(c *gin.Context) {
		var regStart RegistrationStart

		err := c.BindJSON(&regStart)
		if err != nil {
			pklog.Warn("failed to parse body", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.Debug("using session", "session_id", sess.ID)

		if sess.State == "registration" {
			sess.FSM.Event(c, "finishRegistration")
			sess.State = sess.FSM.Current()
		}

		if sess.FSM.Current() != "null" {
			pklog.Warn("invalid source state for this request", "state", sess.FSM.Current())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		u, err := app.db.GetUserByName(regStart.Name)
		if err == nil {
			pklog.Warn("user that already exists in db is trying to register", "username", u.Name)
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
			pklog.Error("failed to save new user in registration start", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		authSelect := protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationPreferred,
		}
		conveyancePref := protocol.PreferNoAttestation

		options, session, err := app.webauthn.BeginRegistration(u, webauthn.WithAuthenticatorSelection(authSelect), webauthn.WithConveyancePreference(conveyancePref))

		pklog.With(
			"options", options,
			"session", session,
		).Debug("Started webauthn registration")
		if err != nil {
			pklog.Error("failed to start webauthn registration", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.Data = session
		sess.UserID = u.ID

		err = sess.FSM.Event(c, "startRegistration")
		if err != nil {
			pklog.Error("failed to transition state", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.Error("failed to save session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})
	pk.POST("/register/finish", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.Debug("using session", "session_id", sess.ID)

		defer func() {
			sess.Data = &webauthn.SessionData{}
			app.db.SetSession(*sess)
		}()

		if sess.FSM.Current() != "registration" {
			pklog.Warn("invalid source state for this request", "state", sess.FSM.Current())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		resp, err := protocol.ParseCredentialCreationResponseBody(c.Request.Body)
		if err != nil {
			pklog.Error("failed to parse credential body", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess := sess.Data

		user, err := app.db.GetUser(sess.UserID)
		if err != nil {
			pklog.Error("failed to retrieve user from session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		credential, err := app.webauthn.CreateCredential(user, *webSess, resp)
		if err != nil {
			pklog.Error("failed to create credential", "error", err)
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
			pklog.Error("failed to store credential with user", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = sess.FSM.Event(c, "finishRegistration")
		if err != nil {
			pklog.Error("failed to transition state", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.Error("failed to save session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatus(http.StatusOK)
	})

	pk.POST("/login/anystart", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.Debug("using session", "session_id", sess.ID)

		if sess.State == "login" {
			sess.FSM.Event(c, "loginFail")
			sess.State = sess.FSM.Current()
		}

		if sess.State != "null" {
			pklog.Warn("invalid source state for this request", "state", sess.FSM.Current())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		options, session, err := app.webauthn.BeginDiscoverableLogin()

		if err != nil {
			pklog.Error("failed to start webauthn login", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.Data = session

		err = sess.FSM.Event(c, "startLogin")
		if err != nil {
			pklog.Error("failed to transition state", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.Error("failed to save session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, options)

	})

	pk.POST("/login/finish", func(c *gin.Context) {

		sess := app.getSession(c.GetString("SESSION_ID"))
		pklog.Debug("using session", "session_id", sess.ID)

		if sess.FSM.Current() != "login" {
			pklog.Warn("invalid source state for this request", "state", sess.FSM.Current())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		resp, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
		if err != nil {
			pklog.Error("failed to parse credential body", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess := sess.Data

		user, err := app.db.GetUser(resp.Response.UserHandle)
		if err != nil {
			pklog.Error("failed to retrieve user from session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		webSess.UserID = user.WebAuthnID()

		credential, err := app.webauthn.ValidateLogin(user, *webSess, resp)
		if err != nil {
			pklog.Error("failed to validate login", "error", err)
			c.AbortWithStatus(403)
			return
		}

		err = sess.FSM.Event(c, "loginSuccess")
		if err != nil {
			pklog.Error("failed to transition state", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess.UserID = user.WebAuthnID()

		err = app.db.SetSession(*sess)
		if err != nil {
			pklog.Error("failed to save session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		pklog.Debug("success logging in!", "cred", credential.ID)

		res := gin.H{}

		if url, ok := sess.Vals["redirect_url"]; ok {
			delete(sess.Vals, "redirect_url")
			app.db.SetSession(*sess)
			pklog.Info("redirecting to url")
			res = gin.H{
				"redirect_url": url,
			}
		}

		c.AbortWithStatusJSON(http.StatusOK, res)

	})

	r.GET("/register.html", func(c *gin.Context) {

		var user *tobab.User
		var err error

		sess := app.getSession(c.GetString("SESSION_ID"))

		if sess.State == "authenticated" {
			user, err = app.db.GetUser(sess.UserID)
			if err != nil {
				pklog.Error("failed to retrieve user from session", "error", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.Redirect(307, "/")
		}

		if sess.State != "null" {
			if sess.State == "login" {
				err = sess.FSM.Event(c, "loginFail")
				if err != nil {
					pklog.Error("failed to transition state", "error", err)
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				err = app.db.SetSession(*sess)
				if err != nil {
					pklog.Error("failed to save session", "error", err)
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
			app.logger.Warn("invalid username provided", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		hosts := app.getHosts()
		if !tobab.Contains(hosts, hostName) {
			app.logger.Warn("invalid hostname provided")
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
			app.logger.Warn("Failed to update user", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(200, gin.H{})
	})

	admin.POST("/toggleAdmin", func(c *gin.Context) {
		userName := c.Query("user")

		u, err := app.db.GetUserByName(userName)
		if err != nil {
			app.logger.Warn("invalid username provided", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		u.Admin = !u.Admin

		err = app.db.SetUser(*u)
		if err != nil {
			app.logger.Warn("Failed to update user", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(200, gin.H{})
	})

	admin.GET("/index.html", func(c *gin.Context) {

		users, err := app.db.GetUsers()
		if err != nil {
			app.logger.Error("failed to retrieve users from database", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		sess := app.getSession(c.GetString("SESSION_ID"))
		hosts := app.getHosts()

		user, err := app.db.GetUser(sess.UserID)
		if err != nil {
			pklog.Error("failed to retrieve user from session", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.HTML(200, "admin.html", adminVars{
			Users: users,
			Hosts: hosts,
			User:  *user,
		})
	})

	r.GET("/", func(c *gin.Context) {

		var user *tobab.User
		var err error
		sess := app.getSession(c.GetString("SESSION_ID"))

		if sess.State == "authenticated" {
			user, err = app.db.GetUser(sess.UserID)
			if err != nil {
				pklog.Error("failed to retrieve user from session", "error", err)
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

	ll := app.logger.With("service", "verify")
	sess := app.getSession(c.GetString("SESSION_ID"))

	host := c.GetHeader("X-Forwarded-Host")
	proto := c.GetHeader("X-Forwarded-Proto")
	uri := c.GetHeader("X-Forwarded-Uri")
	u := "unknown"

	ll = ll.With(
		"host", host,
		"proto", proto,
		"uri", uri,
		"user", u,
	)

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
			ll.Error("failed to save session", "error", err)
		}
		ll.With("redirect_url", redirect_url.String()).Info("redirecting to login")

		c.Header("HX-Redirect", app.fqdn)
		c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
		return
	}

	user, err = app.db.GetUser(sess.UserID)
	if err != nil && err != storm.ErrNotFound {
		ll.Error("failed to retrieve user from session", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if user == nil {
		c.Header("HX-Redirect", app.fqdn)
		c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
		return
	}

	ll = ll.With(
		"user", user.Name,
	)

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

	ll.Warn("Return 307 to unknown user")
	c.Header("HX-Redirect", app.fqdn)
	c.Redirect(http.StatusTemporaryRedirect, app.fqdn)
}
