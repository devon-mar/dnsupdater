package updater

import (
	"fmt"
	"time"

	"github.com/bodgit/tsig"
	"github.com/bodgit/tsig/gss"
	"github.com/miekg/dns"
)

type dnsExchanger interface {
	Exchange(*dns.Msg, string) (*dns.Msg, time.Duration, error)
}

type gssNegotiator interface {
	NegotiateContextWithCredentials(string, string, string, string) (string, time.Time, error)
	NegotiateContext(string) (string, time.Time, error)
	DeleteContext(string) error
	Close() error
}

type RFC2136Updater struct {
	servers []string
	dns     dnsExchanger
	gss     gssNegotiator

	// For GSS
	username string
	password string
	domain   string
}

// Servers must have len > 0.
func NewRFC2136(servers []string) *RFC2136Updater {
	return &RFC2136Updater{
		servers: servers,
		dns:     &dns.Client{Net: "tcp"},
	}
}

func (u *RFC2136Updater) Close() error {
	if u.gss != nil {
		return u.gss.Close()
	}
	return nil
}

func (u *RFC2136Updater) WithGSS() error {
	gssClient, err := gss.NewClient(u.dns.(*dns.Client))
	if err != nil {
		return err
	}
	u.gss = gssClient
	u.dns.(*dns.Client).TsigProvider = gssClient
	return err
}

func (u *RFC2136Updater) WithCredentials(username, password, domain string) {
	u.username = username
	u.password = password
	u.domain = domain
}

func (u *RFC2136Updater) getTKEY(host string) (string, func(), error) {
	if u.gss == nil {
		return "", nil, nil
	}
	var key string
	var err error
	if u.username != "" && u.password != "" && u.domain != "" {
		key, _, err = u.gss.NegotiateContextWithCredentials(host, u.domain, u.username, u.password)
	} else {
		key, _, err = u.gss.NegotiateContext(host)
	}
	if err != nil {
		return "", nil, err
	}
	// An error is returned from DeleteContext only if the key is not found.
	// Therefore, we can safely ignore it.
	return key, func() { _ = u.gss.DeleteContext(key) }, nil
}

func (u *RFC2136Updater) Insert(zone string, records []dns.RR) error {
	var err error
	for _, srv := range u.servers {
		err = u.insert(srv, zone, records)
		if err == nil {
			break
		}
	}
	return err
}

func (u *RFC2136Updater) insert(server string, zone string, records []dns.RR) error {
	tkey, cleanup, err := u.getTKEY(server)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	msg := new(dns.Msg)
	msg.SetUpdate(dns.Fqdn(zone))
	msg.RecursionDesired = false
	msg.Insert(records)

	if tkey != "" {
		msg.SetTsig(tkey, tsig.GSS, 300, time.Now().Unix())
	}

	r, _, err := u.dns.Exchange(msg, server)
	if err != nil {
		return err
	}
	if r.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("got rcode %s", dns.RcodeToString[r.Rcode])
	}
	return nil
}
