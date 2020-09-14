package storm

import (
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
	return nil
}

func (db *stormDB) AddGlob(g tobab.Glob) error {
	return db.db.Save(&g)
}

func (db *stormDB) GetGlob(n string) (*tobab.Glob, error) {
	var g tobab.Glob
	err := db.db.One("Name", n, &g)
	return &g, err
}

func (db *stormDB) GetGlobs() ([]tobab.Glob, error) {
	var globs []tobab.Glob
	err := db.db.All(&globs)
	return globs, err
}
func (db *stormDB) DeleteGlob(string) error {
	return nil
}

func (db *stormDB) Close() {
	db.db.Close()
}
