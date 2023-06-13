package tobab

type Database interface {
	KVSet(string, any) error

	KVGetString(string) (string, error)
	KVGetBool(string) (bool, error)
	KVGet(string, any) error

	GetUsers() ([]User, error)
	GetUser([]byte) (*User, error)
	GetUserByName(string) (*User, error)
	SetUser(User) error

	GetSession(string) (*Session, error)
	CleanupOldSessions()
	SetSession(Session) error
}
