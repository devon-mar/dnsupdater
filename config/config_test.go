package config

import (
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	tests := map[string]struct {
		want    *Config
		wantErr bool
	}{
		"simple": {
			want: &Config{
				Servers: []string{"ns.example.com"},
				Zones: map[string]*Zone{
					"example.com": {
						TTL:     defaultTTL,
						Records: map[string]*Record{"test": {FQDN: "test.example.com.", CNAME: "a", TTL: defaultTTL}},
					},
				},
			},
		},
		"all_types": {
			want: &Config{
				Servers: []string{"ns.example.com"},
				Zones: map[string]*Zone{
					"example.com": {
						TTL: 10,
						Records: map[string]*Record{
							"test": {FQDN: "test.example.com.", CNAME: "a", TTL: 10},
							"test2": {
								FQDN: "test2.example.com.",
								TTL:  10,
								Host: []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("2001:db8::1")},
								TXT:  []string{"abc"},
								MX:   []MXRecord{{MX: "mx1.example.com", Preference: 10}, {MX: "mx2.example.com", Preference: 15}},
								SRV:  []SRVRecord{{Target: "www.example.com", Port: 80, Priority: 1, Weight: 10}},
							},
						},
					},
				},
			},
		},
		"gss_cred": {
			want: &Config{
				Servers: []string{"ns.example.com"},
				Zones: map[string]*Zone{
					"example.com": {
						TTL:     defaultTTL,
						Records: map[string]*Record{"test": {FQDN: "test.example.com.", CNAME: "a", TTL: defaultTTL}},
					},
				},
				GSS: &GSSConfig{
					Username: "username",
					Password: "password",
					Domain:   "domain",
				},
			},
		},
		"gss_no_cred": {
			want: &Config{
				Servers: []string{"ns.example.com"},
				Zones: map[string]*Zone{
					"example.com": {
						TTL:     defaultTTL,
						Records: map[string]*Record{"test": {FQDN: "test.example.com.", CNAME: "a", TTL: defaultTTL}},
					},
				},
				GSS: &GSSConfig{},
			},
		},
		"apex": {
			want: &Config{
				Servers: []string{"ns.example.com"},
				Zones: map[string]*Zone{
					"example.com": {
						TTL:     defaultTTL,
						Records: map[string]*Record{"@": {FQDN: "example.com.", Host: []netip.Addr{netip.MustParseAddr("192.0.2.1")}, TTL: defaultTTL}},
					},
				},
			},
		},
		"gss_no_username": {wantErr: true},
		"gss_no_password": {wantErr: true},
		"gss_no_domain":   {wantErr: true},
		"filenotfound":    {wantErr: true},
		"wrong_type":      {wantErr: true},
		"no_zones":        {wantErr: true},
		"no_servers":      {wantErr: true},
		"invalid_record":  {wantErr: true},
		"zone_no_records": {wantErr: true},
		"extra_key":       {wantErr: true},
		"mx_invalid":      {wantErr: true},
		"srv_invalid":     {wantErr: true},
		"txt_empty_slice": {wantErr: true},
		"cname_and_host":  {wantErr: true},
	}
	for file, tc := range tests {
		t.Run(file, func(t *testing.T) {
			c, err := ReadConfig(filepath.Join("testdata", file+".yml"))
			if err == nil && tc.wantErr {
				t.Error("expected an error")
			} else if err != nil && !tc.wantErr {
				t.Errorf("expected no error but got: %v", err)
			}
			if !reflect.DeepEqual(c, tc.want) {
				t.Errorf("got %#v, want %#v", c, tc.want)
			}
		})
	}
}

func clearEnv() {
	os.Setenv(envServers, "")
	os.Setenv(envUsername, "")
	os.Setenv(envPassword, "")
	os.Setenv(envDomain, "")
}

func TestConfigLoadEnvServers(t *testing.T) {
	clearEnv()

	servers := []string{"ns1.example.com", "ns2.example.com"}
	os.Setenv(envServers, strings.Join(servers, "\n"))

	c := &Config{}
	c.loadEnv()

	want := &Config{Servers: servers}

	if !reflect.DeepEqual(c, want) {
		t.Errorf("expected %#v, got %#v", c, want)
	}
}

func TestConfigLoadEnvGSS(t *testing.T) {
	clearEnv()

	u := "username"
	p := "password"
	d := "domain"
	os.Setenv(envUsername, u)
	os.Setenv(envPassword, p)
	os.Setenv(envDomain, d)

	c := &Config{}
	c.loadEnv()

	if c.GSS == nil {
		t.Fatalf("expected GSS to be non-nil")
	}

	if c.GSS.Username != u {
		t.Errorf("got username %q, want %q", c.GSS.Username, u)
	}
	if c.GSS.Password != p {
		t.Errorf("got password %q, want %q", c.GSS.Password, p)
	}
	if c.GSS.Domain != d {
		t.Errorf("got domain %q, want %q", c.GSS.Domain, d)
	}
}
