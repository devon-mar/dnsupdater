package config

import (
	"errors"
	"os"
	"strings"

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
	TTL     uint32             `yaml:"default_ttl"`
}

func (z *Zone) init() {
	if z.TTL == 0 {
		z.TTL = defaultTTL
	}
	for name, r := range z.Records {
		r.Name = name
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
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(raw, c); err != nil {
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
	for _, z := range c.Zones {
		z.init()
	}
}

// Load config from env variables.
func (c *Config) loadEnv() {
	if servers := os.Getenv(envServers); servers != "" {
		c.Servers = strings.Split(servers, "\n")
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
