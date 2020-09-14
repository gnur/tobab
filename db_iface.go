package tobab

type Database interface {
	//hosts
	AddHost(Host) error
	GetHost(string) (*Host, error)
	GetHosts() ([]Host, error)
	DeleteHost(string) error

	//globs
	AddGlob(Glob) error
	GetGlob(string) (*Glob, error)
	GetGlobs() ([]Glob, error)
	DeleteGlob(string) error
}
