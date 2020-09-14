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

	t, err := app.decryptToken(c.Value)
	if err != nil {
		return "", err
	}

	return t.Subject, nil
}

func (app *Tobab) decryptToken(t string) (*paseto.JSONToken, error) {
	// Decrypt data
	var token paseto.JSONToken
	var footer string
	err := v2.Decrypt(t, app.key, &token, &footer)
	if err != nil {
		return nil, ErrInvalidToken
	}
	err = token.Validate()
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (app *Tobab) newToken(u string, claims map[string]string) (string, error) {
	now := time.Now()
	exp := now.Add(app.maxAge)
	nbt := now

	jsonToken := paseto.JSONToken{
		Issuer:     app.fqdn,
		Subject:    u,
		IssuedAt:   now,
		Expiration: exp,
		NotBefore:  nbt,
	}

	for k, v := range claims {
		jsonToken.Set(k, v)
	}

	token, err := v2.Encrypt(app.key, jsonToken, footer)
	if err != nil {
		return "", err
	}

	return token, nil
}
