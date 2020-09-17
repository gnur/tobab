package tobab

type Database interface {
	//hosts
	AddHost(Host) error
	GetHost(string) (*Host, error)
	GetHosts() ([]Host, error)
	DeleteHost(string) error
}
