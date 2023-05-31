package storm

import (
	"time"

	"github.com/asdine/storm"
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

func (db *stormDB) AddHost(h tobab.Host) error {
	return db.db.Save(&h)
}

func (db *stormDB) GetHost(hostname string) (*tobab.Host, error) {
	var h tobab.Host
	err := db.db.One("Hostname", hostname, &h)
	return &h, err
}
func (db *stormDB) GetHosts() ([]tobab.Host, error) {
	var hosts []tobab.Host
	err := db.db.All(&hosts)
	return hosts, err
}
func (db *stormDB) DeleteHost(hostname string) error {
	h, err := db.GetHost(hostname)
	if err != nil {
		return err
	}
	return db.db.DeleteStruct(h)
}

func (db *stormDB) Close() {
	db.db.Close()
}

func (db *stormDB) GetUser(id []byte) (*tobab.User, error) {
	var u tobab.User
	err := db.db.One("ID", id, &u)
	return &u, err
}

func (db *stormDB) SetUser(u tobab.User) error {
	return db.db.Save(&u)
}

func (db *stormDB) GetSession(id string) (*tobab.Session, error) {
	var s tobab.Session
	err := db.db.One("ID", id, &s)
	s.LastSeen = time.Now()
	db.SetSession(s)

	return &s, err
}

func (db *stormDB) SetSession(s tobab.Session) error {
	return db.db.Save(&s)
}

func (db *stormDB) CleanupOldSessions() {
	//TODO: create this
}
