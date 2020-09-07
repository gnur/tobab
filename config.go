package main

type Config struct {
	CertDir string
	Hosts   []Host
	Email   string
	Staging bool
}

type Host struct {
	Host    string
	Backend string
	Type    string
}
