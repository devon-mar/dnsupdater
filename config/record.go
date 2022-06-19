package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/miekg/dns"
)

type Record struct {
	FQDN  string
	Host  []netip.Addr `yaml:"host"`
	TXT   [][]string   `yaml:"txt"`
	MX    []MXRecord   `yaml:"mx"`
	SRV   []SRVRecord  `yaml:"srv"`
	CNAME string       `yaml:"cname"`
	TTL   uint32       `yaml:"ttl"`
}

type MXRecord struct {
	Preference uint16 `yaml:"preference"`
	MX         string `yaml:"mx"`
}

type SRVRecord struct {
	Priority uint16 `yaml:"priority"`
	Weight   uint16 `yaml:"weight"`
	Port     uint16 `yaml:"port"`
	Target   string `yaml:"target"`
}

func (r *Record) Validate() error {
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
	if len(r.MX) > 0 {
		typeCount++
		if err := r.validateMX(); err != nil {
			return err
		}
	}
	if len(r.SRV) > 0 {
		typeCount++
		if err := r.validateSRV(); err != nil {
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

func (r *Record) validateMX() error {
	for _, mx := range r.MX {
		if err := mx.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (mx *MXRecord) validate() error {
	if mx.MX == "" {
		return errors.New("MX record must have a MX")
	}
	return nil
}

func (r *Record) validateSRV() error {
	for _, srv := range r.SRV {
		if err := srv.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *SRVRecord) validate() error {
	if s.Target == "" {
		return errors.New("SRV record must have a target")
	}
	return nil
}

func (r *Record) header(rrtype uint16) dns.RR_Header {
	return dns.RR_Header{
		Name:   r.FQDN,
		Rrtype: rrtype,
		Class:  dns.ClassINET,
		Ttl:    r.TTL,
	}
}

func (r *Record) host() []dns.RR {
	ret := make([]dns.RR, 0, len(r.Host))
	for _, ip := range r.Host {
		if ip.Is4() {
			ret = append(ret,
				&dns.A{
					Hdr: r.header(dns.TypeA),
					A:   net.IP(ip.AsSlice()).To4(),
				},
			)
		} else {
			ret = append(ret,
				&dns.AAAA{
					Hdr:  r.header(dns.TypeAAAA),
					AAAA: net.IP(ip.AsSlice()).To16(),
				},
			)
		}
	}
	return ret
}

func (r *Record) txt() []dns.RR {
	ret := make([]dns.RR, 0, len(r.TXT))
	for _, txt := range r.TXT {
		ret = append(ret,
			&dns.TXT{
				Hdr: r.header(dns.TypeTXT),
				Txt: txt,
			},
		)
	}
	return ret
}

func (r *Record) cname() *dns.CNAME {
	if r.CNAME == "" {
		return nil
	}
	return &dns.CNAME{
		Hdr:    r.header(dns.TypeCNAME),
		Target: dns.Fqdn(r.CNAME),
	}
}

func (r *Record) mx() []dns.RR {
	ret := make([]dns.RR, 0, len(r.MX))
	for _, mx := range r.MX {
		ret = append(ret,
			&dns.MX{
				Hdr:        r.header(dns.TypeMX),
				Preference: mx.Preference,
				Mx:         dns.Fqdn(mx.MX),
			},
		)
	}
	return ret
}

func (r *Record) srv() []dns.RR {
	ret := make([]dns.RR, 0, len(r.SRV))
	for _, srv := range r.SRV {
		ret = append(ret,
			&dns.SRV{
				Hdr:      r.header(dns.TypeSRV),
				Priority: srv.Priority,
				Weight:   srv.Weight,
				Port:     srv.Port,
				Target:   dns.Fqdn(srv.Target),
			},
		)
	}
	return ret
}

func (r *Record) Records() []dns.RR {
	ret := []dns.RR{}

	ret = append(ret, r.host()...)
	ret = append(ret, r.txt()...)
	ret = append(ret, r.mx()...)
	ret = append(ret, r.srv()...)
	if cname := r.cname(); cname != nil {
		ret = append(ret, cname)
	}
	return ret
}
