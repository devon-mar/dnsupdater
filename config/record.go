package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/miekg/dns"
)

type Record struct {
	Name  string
	Host  []netip.Addr `yaml:"host"`
	TXT   [][]string   `yaml:"txt"`
	CNAME string       `yaml:"cname"`
	TTL   uint32       `yaml:"ttl"`
}

func (r *Record) Validate() error {
	if r.Name == "" {
		return errors.New("record name is empty")
	}

	var typeCount int
	if len(r.Host) > 0 {
		typeCount++
		if err := r.validateHost(); err != nil {
			return err
		}
	}
	if len(r.TXT) > 0 {
		typeCount++
		if err := r.validateTXT(); err != nil {
			return err
		}
	}
	if r.CNAME != "" {
		typeCount++
	}

	if typeCount == 0 {
		return errors.New("must specify at least one type")
	}
	if r.CNAME != "" && typeCount > 1 {
		return errors.New("cannot have other records with CNAME")
	}
	return nil
}

func (r *Record) validateTXT() error {
	for _, t := range r.TXT {
		if len(t) == 0 {
			return errors.New("TXT must not be empty")
		}
	}
	return nil
}

func (r *Record) validateHost() error {
	for _, ip := range r.Host {
		if !ip.Is4() && !ip.Is6() {
			// This should probably never happen.
			return fmt.Errorf("cannot determine record type for host %q", ip)
		}
	}
	return nil
}

func (r *Record) header(fqdn string, rrtype uint16) dns.RR_Header {
	return dns.RR_Header{
		Name:   fqdn,
		Rrtype: rrtype,
		Class:  dns.ClassINET,
		Ttl:    r.TTL,
	}
}

func (r *Record) host(fqdn string) []dns.RR {
	ret := make([]dns.RR, 0, len(r.Host))
	for _, ip := range r.Host {
		if ip.Is4() {
			ret = append(ret,
				&dns.A{
					Hdr: r.header(fqdn, dns.TypeA),
					A:   net.IP(ip.AsSlice()).To4(),
				},
			)
		} else {
			ret = append(ret,
				&dns.AAAA{
					Hdr:  r.header(fqdn, dns.TypeAAAA),
					AAAA: net.IP(ip.AsSlice()).To16(),
				},
			)
		}
	}
	return ret
}

func (r *Record) txt(fqdn string) []dns.RR {
	ret := make([]dns.RR, 0, len(r.TXT))
	for _, txt := range r.TXT {
		ret = append(ret,
			&dns.TXT{
				Hdr: r.header(fqdn, dns.TypeTXT),
				Txt: txt,
			},
		)
	}
	return ret
}

func (r *Record) cname(fqdn string) *dns.CNAME {
	if r.CNAME == "" {
		return nil
	}
	return &dns.CNAME{
		Hdr:    r.header(fqdn, dns.TypeCNAME),
		Target: dns.Fqdn(r.CNAME),
	}
}

func (r *Record) Records(zone string) []dns.RR {
	ret := []dns.RR{}

	fqdn := dns.Fqdn(r.Name + "." + zone)
	ret = append(ret, r.host(fqdn)...)
	ret = append(ret, r.txt(fqdn)...)
	if cname := r.cname(fqdn); cname != nil {
		ret = append(ret, cname)
	}
	return ret
}
