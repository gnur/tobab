package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

		u := c.Request.Header.Get("X-Tobab-User")
		if u == "" || !allowAdmin(u, app.config.AdminGlobs) {
			app.logger.WithFields(logrus.Fields{
				"user":  u,
				"globs": app.config.AdminGlobs,
			}).Debug("denying request")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

	}
}
