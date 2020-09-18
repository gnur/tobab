package tobab

import "testing"

func TestHost_Validate(t *testing.T) {
	cookiescope := "example.com"
	type fields struct {
		Hostname string
		Backend  string
		Type     string
		Public   bool
		Globs    []Glob
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "valid",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "https://localhost:1234",
				Type:     "http",
				Public:   true,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid backend scheme",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "htps://localhost:1234",
				Type:     "http",
				Public:   true,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "missing backend scheme",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "localhost:1234",
				Type:     "http",
				Public:   true,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "invalid type",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "http://localhost:1234",
				Type:     "tcp",
				Public:   true,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "missing hostname",
			fields: fields{
				Backend: "http://localhost:1234",
				Type:    "tcp",
				Public:  true,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "unreachable",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "http://localhost:1234",
				Type:     "http",
				Public:   false,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "reachable",
			fields: fields{
				Hostname: "test.example.com",
				Backend:  "http://localhost:1234",
				Type:     "http",
				Public:   false,
				Globs:    []Glob{"*"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unreachable because of domain",
			fields: fields{
				Hostname: "test.example.co.uk",
				Backend:  "http://localhost:1234",
				Type:     "http",
				Public:   false,
				Globs:    []Glob{"*"},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				Hostname: tt.fields.Hostname,
				Backend:  tt.fields.Backend,
				Type:     tt.fields.Type,
				Public:   tt.fields.Public,
				Globs:    tt.fields.Globs,
			}
			got, err := h.Validate(cookiescope)
			if (err != nil) != tt.wantErr {
				t.Errorf("Host.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Host.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
