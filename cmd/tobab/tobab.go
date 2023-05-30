package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/clirpc"
)

func (app *Tobab) setTobabRoutes(r *gin.Engine) {

	api := r.Group("/v1/api/")
	api.Use(app.adminMiddleware())

	//GET hosts
	api.GET("/hosts", func(c *gin.Context) {
		hosts, err := app.db.GetHosts()
		if err != nil {
			app.logger.WithError(err).Error("Failed getting hosts from db")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatusJSON(200, hosts)
	})

	//ADD host
	api.POST("/host", func(c *gin.Context) {
		var h tobab.Host
		err := c.BindJSON(&h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to unmarshal host from body")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if ok, err := h.Validate(app.config.CookieScope); !ok {
			app.logger.WithError(err).Error("Invalid hostconfig provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err = app.db.AddHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to add host to database")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatusJSON(202, h)
		go app.restartServer()

	})

	//DELETE host
	api.DELETE("/host/:hostname", func(c *gin.Context) {
		h := c.Param("hostname")
		if h == "" {
			app.logger.Error("No hostname provided")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err := app.db.DeleteHost(h)
		if err != nil {
			app.logger.WithError(err).Error("Failed to delete host from database")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.AbortWithStatus(202)
		go app.restartServer()

	})

	r.GET("/", func(c *gin.Context) {
		providerIndex := &ProviderIndex{Providers: []string{"google"}, ProvidersMap: map[string]string{"google": "Google"}}

		user, err := app.extractUser(c.Request)
		if err == nil {
			providerIndex.User = user
		} else {
			app.logger.WithError(err).Error("unable to get user from request")
		}

		c.HTML(200, "index.html", providerIndex)
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
