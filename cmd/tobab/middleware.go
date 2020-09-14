package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/asdine/storm"
	"github.com/sirupsen/logrus"
)

func (app *Tobab) getRBACMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			hostname := r.Host
			u, extractUserErr := app.extractUser(r)
			if extractUserErr != nil && extractUserErr != ErrUnauthenticatedRequest {
				//this shouldn't happen unless someone tampered with a cookie manually
				app.logger.WithError(extractUserErr).Error("Unable to extract user")
				//invalid cookie is present, delete it and force re-auth
				c := http.Cookie{
					Name:     "X-Tobab-Token",
					Domain:   app.config.CookieScope,
					SameSite: http.SameSiteLaxMode,
					Secure:   true,
					HttpOnly: true,
					MaxAge:   -1,
					Path:     "/",
				}
				http.SetCookie(w, &c)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			app.logger.WithFields(logrus.Fields{
				"host": hostname,
				"user": u,
				"uri":  r.RequestURI,
			}).Debug("checking auth")

			r.Header.Add("X-Tobab-User", u)

			//configured hostname is always accessible
			if hostname != app.config.Hostname {
				h, err := app.db.GetHost(hostname)
				if err == storm.ErrNotFound {
					http.Error(w, "not found", 404)
					return
				}

				if !h.HasAccess(u) {
					if extractUserErr == ErrUnauthenticatedRequest {
						redirectURL := url.URL{
							Host:   hostname,
							Path:   r.URL.String(),
							Scheme: "https",
						}
						c := http.Cookie{
							Domain:   app.config.CookieScope,
							Secure:   true,
							HttpOnly: true,
							Value:    redirectURL.String(),
							Path:     "/",
							Name:     "X-Tobab-Source",
						}
						http.SetCookie(w, &c)
						http.Redirect(w, r, app.fqdn, 302)
					} else {
						http.Error(w, "access denied", http.StatusUnauthorized)
					}

					return
				}

				//get all cookies, clear them, and then re-add the ones that are not tobab specific
				cookies := r.Cookies()
				r.Header.Del("Cookie")
				for _, c := range cookies {
					if !strings.HasPrefix(c.Name, "X-Tobab") {
						r.AddCookie(c)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (app *Tobab) adminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			u := r.Header.Get("X-Tobab-User")
			if u == "" || !allowAdmin(u, app.config.AdminGlobs) {
				app.logger.WithFields(logrus.Fields{
					"user":  u,
					"globs": app.config.AdminGlobs,
				}).Debug("denying request")
				http.Error(w, "access denied", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
