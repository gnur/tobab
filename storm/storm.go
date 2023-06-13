package storm

import (
	"fmt"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/gnur/tobab"
)

type stormDB struct {
	db *storm.DB
}

func New(path string) (*stormDB, error) {
	db, err := storm.Open(path)
	if err != nil {
		return nil, err
	}

	database := stormDB{
		db: db,
	}

	return &database, nil
}

func (db *stormDB) KVSet(k string, v any) error {
	return db.db.Set("tobab", k, v)
}
func (db *stormDB) KVGetString(k string) (string, error) {
	var s string
	err := db.db.Get("tobab", k, &s)
	return s, err
}
func (db *stormDB) KVGetBool(k string) (bool, error) {
	var b bool
	err := db.db.Get("tobab", k, &b)
	return b, err
}
func (db *stormDB) KVGet(k string, v any) error {
	return db.db.Get("tobab", k, &v)
}

func (db *stormDB) Close() {
	db.db.Close()
}

func (db *stormDB) GetUsers() ([]tobab.User, error) {
	var users []tobab.User
	err := db.db.All(&users)
	return users, err
}

func (db *stormDB) GetUser(id []byte) (*tobab.User, error) {
	var u tobab.User
	err := db.db.One("ID", id, &u)
	if err == nil {
		u.LastSeen = time.Now()
		db.SetUser(u)
	}
	return &u, err
}

func (db *stormDB) GetUserByName(id string) (*tobab.User, error) {
	var u tobab.User
	err := db.db.One("Name", id, &u)
	return &u, err
}

func (db *stormDB) SetUser(u tobab.User) error {
	return db.db.Save(&u)
}

func (db *stormDB) GetSession(id string) (*tobab.Session, error) {
	var s tobab.Session
	err := db.db.One("ID", id, &s)
	return &s, err
}

func (db *stormDB) SetSession(s tobab.Session) error {
	s.State = s.FSM.Current()
	return db.db.Save(&s)
}

func (db *stormDB) CleanupOldSessions() {
	var sess []tobab.Session
	q := db.db.Select(q.Lte("Expires", time.Now()))
	q.Find(&sess)
	for _, s := range sess {
		fmt.Println(s.Expires)
		db.db.DeleteStruct(&s)
	}
}
