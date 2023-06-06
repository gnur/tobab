package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/asdine/storm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const COOKIE_NAME = "X-Tobab-Session-ID"

func (app *Tobab) getProxyRouter() gin.HandlerFunc {
	return func(c *gin.Context) {

		hostname := c.Request.Host

		if prox, ok := app.proxies[hostname]; ok {
			prox.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		}
	}
}

func (app *Tobab) getSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		//Ignore error, empty string will result in error when retrieving session
		sessID, _ := c.Cookie(COOKIE_NAME)
		session := app.getSession(sessID)

		c.SetCookie(COOKIE_NAME, session.ID, int(app.defaultAge.Seconds()), "/", app.config.CookieScope, true, true)
		c.Set("SESSION_ID", session.ID)
	}
}

func (app *Tobab) getRBACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		hostname := c.Request.Host
		u, extractUserErr := app.extractUser(c.Request)
		if extractUserErr != nil && extractUserErr != ErrUnauthenticatedRequest {
			app.logger.WithError(extractUserErr).Error("Unable to extract user")
			//invalid cookie is present, delete it and force re-auth
			c.SetCookie("X-Tobab-Token", "", -1, "/", app.config.CookieScope, true, true)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		app.logger.WithFields(logrus.Fields{
			"host": hostname,
			"user": u,
			"uri":  c.Request.RequestURI,
		}).Debug("checking auth")

		c.Request.Header.Add("X-Tobab-User", u)

		//configured hostname is always accessible
		if hostname != app.config.Hostname {
			h, err := app.db.GetHost(hostname)
			if err == storm.ErrNotFound {
				c.AbortWithStatus(404)
				return
			}

			if !h.HasAccess(u) {
				if extractUserErr == ErrUnauthenticatedRequest {
					redirectURL := url.URL{
						Host:   hostname,
						Path:   c.Request.URL.String(),
						Scheme: "https",
					}

					c.SetCookie("X-Tobab-Source", redirectURL.String(), 10, "/", app.config.CookieScope, true, true)
					c.Redirect(302, app.fqdn)
				} else {
					c.AbortWithStatus(http.StatusUnauthorized)
				}

				return
			}

			//get all cookies, clear them, and then re-add the ones that are not tobab specific
			cookies := c.Request.Cookies()
			c.Request.Header.Del("Cookie")
			for _, cook := range cookies {
				if !strings.HasPrefix(cook.Name, "X-Tobab") {
					http.SetCookie(c.Writer, cook)
				}
			}
		}
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
