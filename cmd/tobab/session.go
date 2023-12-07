package main

import (
	"time"

	"github.com/gnur/tobab"
	"github.com/lithammer/shortuuid"
	"github.com/looplab/fsm"
)

func setupFSM(base string) *fsm.FSM {
	fsm := fsm.NewFSM(
		"null",
		fsm.Events{
			//from null
			{Name: "startRegistration", Src: []string{"null"}, Dst: "registration"},
			{Name: "startLogin", Src: []string{"null"}, Dst: "login"},

			//from registration
			{Name: "finishRegistration", Src: []string{"registration"}, Dst: "null"},

			//from login
			{Name: "loginSuccess", Src: []string{"login"}, Dst: "authenticated"},
			{Name: "loginFail", Src: []string{"login"}, Dst: "null"},

			//from authenticated
			{Name: "logout", Src: []string{"authenticated"}, Dst: "null"},
			{Name: "addRegistration", Src: []string{"authenticated"}, Dst: "authRegistration"},

			//from authRegistration
			{Name: "finsihAuthRegistration", Src: []string{"authRegistration"}, Dst: "authenticated"},
		},
		fsm.Callbacks{},
	)
	fsm.SetState(base)

	return fsm
}

func (app *Tobab) getSession(id string) *tobab.Session {
	var s *tobab.Session
	newSession := false

	if id == "" {
		newSession = true
	} else {
		dbSess, err := app.db.GetSession(id)
		if err != nil {
			app.logger.Debug("Creating new session because of error getting sesssion", "error", err)
			newSession = true
		} else if dbSess.Expires.Before(time.Now()) {
			app.logger.Debug("Creating new session because of expired session", "expires", dbSess.Expires)
			newSession = true
		} else {
			app.logger.Debug("Using existing session")
			s = dbSess
		}
	}

	if newSession {
		app.logger.Debug("Creating new session", "id", id)
		s = &tobab.Session{
			ID:      shortuuid.New(),
			Created: time.Now(),
			State:   "null",
		}
	}
	s.LastSeen = time.Now()
	s.Expires = time.Now().Add(app.defaultAge)
	s.FSM = setupFSM(s.State)
	if s.Vals == nil {
		s.Vals = make(map[string]string)
	}

	err := app.db.SetSession(*s)
	if err != nil {
		app.logger.Error("could not save session, this breaks everything, crashing hard", "error", err)
		return nil
	}

	return s
}
