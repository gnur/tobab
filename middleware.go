package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func (app *Tobab) getRBACMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			h := r.Host
			u, err := app.extractUser(r)
			if err != nil {
				//this shouldn't happen
				app.logger.WithError(err).Error("Unable to extract user")
			}
			app.logger.WithFields(logrus.Fields{
				"host": h,
				"user": u,
				"uri":  r.RequestURI,
			}).Debug("checking auth")

			if !hasAccess(u, h, app.config) {
				if err == ErrUnauthenticatedRequest {
					app.logger.Info("should redirect")
					http.Redirect(w, r, app.fqdn, 302)
				} else {
					app.logger.Info("Should return 403")
					http.Error(w, "access denied", http.StatusUnauthorized)
				}

				return
			}

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	}
}
