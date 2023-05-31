package main

import (
	"errors"
	"net/http"
)

var ErrUnauthenticatedRequest = errors.New("No user information in request")

func (app *Tobab) extractUser(r *http.Request) (string, error) {

	c, err := r.Cookie("X-Tobab-Token")
	if err != nil {
		return "", ErrUnauthenticatedRequest
	}

	sess, err := app.db.GetSession(c.Value)
	if err != nil {
		return "", ErrUnauthenticatedRequest
	}

	user, err := app.db.GetUser(sess.UserID)
	if err != nil {
		return "", ErrUnauthenticatedRequest
	}

	return user.Name, nil
}
