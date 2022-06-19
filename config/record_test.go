package config

import (
	"net"
	"net/netip"
	"reflect"
	"testing"

	"github.com/miekg/dns"
)

const (
	testZone = "example.com."
)

func TestRecords(t *testing.T) {
	tests := map[string]struct {
		r    *Record
		zone string

		want []dns.RR
	}{
		"A": {
			r: &Record{FQDN: "a." + testZone, Host: []netip.Addr{netip.MustParseAddr("192.0.2.1")}, TTL: 300},
			want: []dns.RR{
				&dns.A{
					Hdr: dns.RR_Header{Name: "a." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 1).To4(),
				},
			},
		},
		"host multiple": {
			r: &Record{
				FQDN: "host." + testZone,
				Host: []netip.Addr{
					netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2"),
					netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("2001:db8::2"),
				},
				TTL: 300,
			},
			want: []dns.RR{
				&dns.A{
					Hdr: dns.RR_Header{Name: "host." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 1).To4(),
				},
				&dns.A{
					Hdr: dns.RR_Header{Name: "host." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 2).To4(),
				},
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "host." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::1"),
				},
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "host." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::2"),
				},
			},
		},
		"AAAA": {
			r: &Record{FQDN: "aaaa." + testZone, Host: []netip.Addr{netip.MustParseAddr("2001:db8::1")}, TTL: 300},
			want: []dns.RR{
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "aaaa." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::1"),
				},
			},
		},
		"CNAME": {
			r: &Record{FQDN: "cname." + testZone, CNAME: "abc.example.com."},
			want: []dns.RR{
				&dns.CNAME{
					Hdr:    dns.RR_Header{Name: "cname." + testZone, Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
					Target: "abc.example.com.",
				},
			},
		},
		"MX": {
			r: &Record{FQDN: "mx." + testZone, MX: []MXRecord{{Preference: 0, MX: "mail.example.com"}}},
			want: []dns.RR{
				&dns.MX{
					Hdr:        dns.RR_Header{Name: "mx." + testZone, Rrtype: dns.TypeMX, Class: dns.ClassINET},
					Preference: 0,
					Mx:         "mail.example.com.",
				},
			},
		},
		"MX multiple": {
			r: &Record{FQDN: "mail." + testZone, MX: []MXRecord{{Preference: 0, MX: "mx1.example.com"}, {Preference: 10, MX: "mx2.example.com"}}},
			want: []dns.RR{
				&dns.MX{
					Hdr:        dns.RR_Header{Name: "mail." + testZone, Rrtype: dns.TypeMX, Class: dns.ClassINET},
					Preference: 0,
					Mx:         "mx1.example.com.",
				},
				&dns.MX{
					Hdr:        dns.RR_Header{Name: "mail." + testZone, Rrtype: dns.TypeMX, Class: dns.ClassINET},
					Preference: 10,
					Mx:         "mx2.example.com.",
				},
			},
		},
		"TXT": {
			r: &Record{FQDN: "txt." + testZone, TXT: [][]string{{"123"}}},
			want: []dns.RR{
				&dns.TXT{
					Hdr: dns.RR_Header{Name: "txt." + testZone, Rrtype: dns.TypeTXT, Class: dns.ClassINET},
					Txt: []string{"123"},
				},
			},
		},
		"TXT multiple": {
			r: &Record{FQDN: "txt." + testZone, TXT: [][]string{{"123", "456"}, {"abc", "def"}}},
			want: []dns.RR{
				&dns.TXT{
					Hdr: dns.RR_Header{Name: "txt." + testZone, Rrtype: dns.TypeTXT, Class: dns.ClassINET},
					Txt: []string{"123", "456"},
				},
				&dns.TXT{
					Hdr: dns.RR_Header{Name: "txt." + testZone, Rrtype: dns.TypeTXT, Class: dns.ClassINET},
					Txt: []string{"abc", "def"},
				},
			},
		},
		"SRV": {
			r: &Record{FQDN: "srv." + testZone, SRV: []SRVRecord{{Priority: 10, Weight: 15, Port: 80, Target: "www.example.net"}}},
			want: []dns.RR{
				&dns.SRV{
					Hdr:      dns.RR_Header{Name: "srv." + testZone, Rrtype: dns.TypeSRV, Class: dns.ClassINET},
					Priority: 10,
					Weight:   15,
					Port:     80,
					Target:   "www.example.net.",
				},
			},
		},
		"SRV multiple": {
			r: &Record{FQDN: "srv." + testZone, SRV: []SRVRecord{
				{Priority: 10, Weight: 15, Port: 80, Target: "www.example.net"},
				{Priority: 20, Weight: 15, Port: 80, Target: "www2.example.net"},
			}},
			want: []dns.RR{
				&dns.SRV{
					Hdr:      dns.RR_Header{Name: "srv." + testZone, Rrtype: dns.TypeSRV, Class: dns.ClassINET},
					Priority: 10,
					Weight:   15,
					Port:     80,
					Target:   "www.example.net.",
				},
				&dns.SRV{
					Hdr:      dns.RR_Header{Name: "srv." + testZone, Rrtype: dns.TypeSRV, Class: dns.ClassINET},
					Priority: 20,
					Weight:   15,
					Port:     80,
					Target:   "www2.example.net.",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if have := tc.r.Records(); !reflect.DeepEqual(have, tc.want) {
				t.Errorf("got %+v, want %+v", have, tc.want)
			}
		})
	}
}
