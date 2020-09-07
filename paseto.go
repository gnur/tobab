package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/o1egl/paseto"
)

var ErrUnauthenticatedRequest = errors.New("No user information in request")
var ErrInvalidToken = errors.New("Unable to parse token")

var v2 = paseto.NewV2()
var footer = "tobab"

func (app *Tobab) extractUser(r *http.Request) (string, error) {

	c, err := r.Cookie("X-Tobab-Token")
	if err != nil {
		return "", ErrUnauthenticatedRequest
	}

	// Decrypt data
	var newJsonToken paseto.JSONToken
	var newFooter string
	err = v2.Decrypt(c.Value, app.key, &newJsonToken, &newFooter)
	if err != nil {
		app.logger.WithError(err).Warning("Unable to parse cookie token")
		return "", ErrInvalidToken
	}

	user := newJsonToken.Subject

	return user, nil
}

func (app *Tobab) newToken(u string) (string, error) {
	now := time.Now()
	exp := now.Add(app.maxAge)
	nbt := now

	jsonToken := paseto.JSONToken{
		Issuer:     app.config.Hostname,
		Subject:    u,
		IssuedAt:   now,
		Expiration: exp,
		NotBefore:  nbt,
	}
	// Add custom claim    to the token
	jsonToken.Set("data", "this is a signed message")
	token, err := v2.Encrypt(app.key, jsonToken, footer)
	if err != nil {
		return "", err
	}

	return token, nil
}
