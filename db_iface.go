package tobab

type Database interface {
	//hosts
	AddHost(Host) error
	GetHost(string) (*Host, error)
	GetHosts() ([]Host, error)
	DeleteHost(string) error

	GetUser([]byte) (*User, error)
	SetUser(User) error

	GetSession(string) (*Session, error)
	CleanupOldSessions()
	SetSession(Session) error
}
