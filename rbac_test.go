package main

import "testing"

func Test_hasAccess(t *testing.T) {
	c := Config{
		Hostname: "login.example.com",
		Hosts: map[string]Host{
			"noone.example.com": {
				Public:       false,
				AllowedGlobs: []string{},
			},
			"everyone.example.com": {
				Public:       true,
				AllowedGlobs: []string{},
			},
			"admin.example.com": {
				Public:       false,
				AllowedGlobs: []string{"admin"},
			},
			"allsignedin.example.com": {
				Public:       false,
				AllowedGlobs: []string{"everyone"},
			},
			"mail.example.com": {
				Public:       false,
				AllowedGlobs: []string{"mail"},
			},
		},
		Globs: map[string]string{
			"everyone": "*",
			"admin":    "*@admin.example.com",
			"mail":     "*@mail.example.com",
		},
	}
	tests := []struct {
		name string
		user string
		host string
		want bool
	}{
		{
			name: "unknown host",
			user: "erwin@example.com",
			host: "invalidhost.example.com",
			want: false,
		},
		{
			name: "unknown host",
			user: "",
			host: "invalidhost.example.com",
			want: false,
		},
		{
			name: "no one",
			user: "erwin@example.com",
			host: "noone.example.com",
			want: false,
		},
		{
			name: "no one 2",
			user: "literally anything",
			host: "noone.example.com",
			want: false,
		},
		{
			name: "everyone",
			user: "erwin@example.com",
			host: "everyone.example.com",
			want: true,
		},
		{
			name: "signed in without user",
			user: "",
			host: "allsignedin.example.com",
			want: false,
		},
		{
			name: "signed in without user",
			user: "erwin@example.com",
			host: "allsignedin.example.com",
			want: true,
		},
		{
			name: "valid admin",
			user: "erwin@admin.example.com",
			host: "admin.example.com",
			want: true,
		},
		{
			name: "invalid admin",
			user: "erwin@example.com",
			host: "admin.example.com",
			want: false,
		},
		{
			name: "valid mail",
			user: "erwin@mail.example.com",
			host: "mail.example.com",
			want: true,
		},
		{
			name: "invalid mail",
			user: "erwin@example.com",
			host: "mail.example.com",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAccess(tt.user, tt.host, c); got != tt.want {
				t.Errorf("hasAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
