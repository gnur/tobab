package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
)

const COOKIE_NAME = "X-Tobab-Session-ID"

func (app *Tobab) getSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		//Ignore error, empty string will result in error when retrieving session
		sessID, _ := c.Cookie(COOKIE_NAME)
		session := app.getSession(sessID)

		c.SetCookie(COOKIE_NAME, session.ID, int(app.defaultAge.Seconds()), "/", app.config.CookieScope, true, true)
		c.Set("SESSION_ID", session.ID)
	}
}

func (app *Tobab) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *tobab.User
		var err error
		sess := app.getSession(c.GetString("SESSION_ID"))

		if sess.State != "authenticated" {
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		user, err = app.db.GetUser(sess.UserID)
		if err != nil {
			app.logger.WithError(err).Error("failed to retrieve user from session")
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		if !user.Admin {
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

	}
}
