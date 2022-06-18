package updater

import "github.com/miekg/dns"

type Updater interface {
	Insert(string, []dns.RR) error
	Close() error
}
