package config

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

const (
	defaultTTL = 3600

	envServers  = "DNS_SERVERS"
	envUsername = "GSS_USERNAME"
	envPassword = "GSS_PASSWORD"
	envDomain   = "GSS_DOMAIN"
)

type Config struct {
	Servers []string         `yaml:"servers"`
	Zones   map[string]*Zone `yaml:"zones"`
	GSS     *GSSConfig       `yaml:"gss"`
}

type Zone struct {
	Records map[string]*Record `yaml:"records"`
	TTL     uint32             `yaml:"ttl"`
}

// zoneName should be a FQDN.
func (z *Zone) init(zoneName string) {
	if z.TTL == 0 {
		z.TTL = defaultTTL
	}
	for name, r := range z.Records {
		if name == "@" {
			r.FQDN = zoneName
		} else {
			r.FQDN = name + "." + zoneName
		}
		if r.TTL == 0 {
			r.TTL = z.TTL
		}
	}
}

func (z *Zone) Validate() error {
	if len(z.Records) == 0 {
		return errors.New("zone has no records")
	}
	for _, r := range z.Records {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Read the config from the file, overriding with env variables, and filling defaults.
func ReadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)

	c := &Config{}
	// https: //github.com/go-yaml/yaml/issues/639#issuecomment-666935833
	if err := d.Decode(c); err != nil && err != io.EOF {
		return nil, err
	}

	c.loadEnv()
	c.init()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) init() {
	for name, z := range c.Zones {
		z.init(dns.Fqdn(name))
	}
}

// Load config from env variables.
func (c *Config) loadEnv() {
	if servers := os.Getenv(envServers); servers != "" {
		serverSlice := strings.Split(servers, "\n")
		for _, s := range serverSlice {
			trimmed := strings.TrimSpace(s)
			if trimmed == "" {
				continue
			}
			c.Servers = append(c.Servers, trimmed)
		}
	}

	// gss.Validate() will check that the rest are not empty.
	if username := os.Getenv(envUsername); username != "" {
		c.GSS = &GSSConfig{
			Username: username,
			Password: os.Getenv(envPassword),
			Domain:   os.Getenv(envDomain),
		}
	}
}

func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return errors.New("servers must not be empty")
	}
	if len(c.Zones) == 0 {
		return errors.New("zones cannot be empty")
	}
	for _, z := range c.Zones {
		if err := z.Validate(); err != nil {
			return err
		}
	}
	if c.GSS != nil {
		if err := c.GSS.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type GSSConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Domain   string `yaml:"domain"`
}

func (c *GSSConfig) Validate() error {
	if c.Username == "" && c.Password == "" && c.Domain == "" {
		return nil
	}
	if c.Username == "" {
		return errors.New("GSS username must not be empty")
	}
	if c.Password == "" {
		return errors.New("GSS password must not be empty")
	}
	if c.Domain == "" {
		return errors.New("GSS domain must not be empty")
	}
	return nil
}
