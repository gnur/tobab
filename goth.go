package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func (app *Tobab) setupgoth(r *mux.Router) {

	goth.UseProviders(
		google.New(app.config.GoogleKey, app.config.GoogleSecret, app.fqdn+"/auth/google/callback"),
	)

	r.HandleFunc("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		gothic.BeginAuthHandler(w, r)
	})

	r.HandleFunc("/auth/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		claims := make(map[string]string)

		claims["avatar"] = user.AvatarURL
		claims["name"] = user.Name
		claims["userid"] = user.UserID

		token, err := app.newToken(user.Email, claims)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		c := http.Cookie{
			Name:     "X-Tobab-Token",
			Domain:   app.config.CookieScope,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
			HttpOnly: true,
			Expires:  time.Now().Add(app.maxAge),
			Value:    token,
			Path:     "/",
		}

		http.SetCookie(w, &c)

		cr, err := r.Cookie("X-Tobab-Source")
		if err != nil {
			http.Redirect(w, r, "/", 302)
		} else {
			http.Redirect(w, r, cr.Value, 302)
		}

	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user, err := app.extractUser(r)
		providerIndex := &ProviderIndex{Providers: []string{"google"}, ProvidersMap: map[string]string{"google": "Google"}}
		if err == nil {
			providerIndex.User = user
		} else {
			app.logger.WithError(err).Error("unable to get user from request")
		}
		err = app.templates.ExecuteTemplate(w, "index.html", providerIndex)
		if err != nil {
			app.logger.WithError(err).Error("failed executing template")
		}
	})
}

type ProviderIndex struct {
	Providers    []string
	ProvidersMap map[string]string
	User         string
}
