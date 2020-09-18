package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gnur/tobab"
	"github.com/gnur/tobab/clirpc"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func (app *Tobab) setTobabRoutes(r *mux.Router) {

	api := r.PathPrefix("/v1/api/").Subrouter()
	api.Use(app.adminMiddleware())

	//GET hosts
	api.HandleFunc("/hosts", func(w http.ResponseWriter, r *http.Request) {
		hosts, err := app.db.GetHosts()
		if err != nil {
			app.logger.WithError(err).Error("Failed getting hosts from db")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		js, err := json.Marshal(hosts)
		if err != nil {
			app.logger.WithError(err).Error("Failed marshalling hosts from config into JSON")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(js)
		if err != nil {
			app.logger.WithError(err).Error("failed writing JSON response")
		}
	}).Methods("GET")

	//ADD host
	api.HandleFunc("/host", func(w http.ResponseWriter, r *http.Request) {
		var h tobab.Host
		err := json.NewDecoder(r.Body).Decode(&h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to unmarshal host from body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if ok, err := h.Validate(app.config.CookieScope); !ok {
			http.Error(w, fmt.Sprintf("invalid backend: %e", err), http.StatusBadRequest)
			return
		}
		err = app.db.AddHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to add host to database")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "ok", 202)
		go app.restartServer()

	}).Methods("POST")

	//DELETE host
	api.HandleFunc("/host/{hostname}", func(w http.ResponseWriter, r *http.Request) {
		h, ok := mux.Vars(r)["hostname"]
		if !ok || h == "" {
			app.logger.Error("No hostname provided")
			http.Error(w, "No hostname provided", http.StatusBadRequest)
			return
		}
		err := app.db.DeleteHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to delete host from database")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		http.Error(w, "ok", 202)
		go app.restartServer()

	}).Methods("DELETE")

	//setup user facing auth
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

		token, err := app.newToken(user.Email, app.fqdn, app.defaultAge)
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

func (app *Tobab) CreateToken(in *clirpc.CreateTokenIn, out *clirpc.CreateTokenOut) error {
	token, err := app.newToken(in.Email, "tobab:cli", in.TTL)
	out.Token = token
	return err
}

func (app *Tobab) ValidateToken(in *clirpc.ValidateTokenIn, out *clirpc.ValidateTokenOut) error {
	token, err := app.decryptToken(in.Token)
	out.Token = *token
	return err
}
