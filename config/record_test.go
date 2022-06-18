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
			r: &Record{Name: "a", A: []netip.Addr{netip.MustParseAddr("192.0.2.1")}, TTL: 300},
			want: []dns.RR{
				&dns.A{
					Hdr: dns.RR_Header{Name: "a." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 1).To4(),
				},
			},
		},
		"A multiple": {
			r: &Record{
				Name: "a",
				A:    []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2")},
				TTL:  300,
			},
			want: []dns.RR{
				&dns.A{
					Hdr: dns.RR_Header{Name: "a." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 1).To4(),
				},
				&dns.A{
					Hdr: dns.RR_Header{Name: "a." + testZone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
					A:   net.IPv4(192, 0, 2, 2).To4(),
				},
			},
		},
		"AAAA": {
			r: &Record{Name: "aaaa", AAAA: []netip.Addr{netip.MustParseAddr("2001:db8::1")}, TTL: 300},
			want: []dns.RR{
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "aaaa." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::1"),
				},
			},
		},
		"AAAA multiple": {
			r: &Record{
				Name: "aaaa",
				AAAA: []netip.Addr{netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("2001:db8::2")},
				TTL:  300,
			},
			want: []dns.RR{
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "aaaa." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::1"),
				},
				&dns.AAAA{
					Hdr:  dns.RR_Header{Name: "aaaa." + testZone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
					AAAA: net.ParseIP("2001:db8::2"),
				},
			},
		},
		"CNAME": {
			r: &Record{Name: "cname", CNAME: "abc.example.com."},
			want: []dns.RR{
				&dns.CNAME{
					Hdr:    dns.RR_Header{Name: "cname." + testZone, Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
					Target: "abc.example.com.",
				},
			},
		},
		"TXT": {
			r: &Record{Name: "txt", TXT: [][]string{{"123"}}},
			want: []dns.RR{
				&dns.TXT{
					Hdr: dns.RR_Header{Name: "txt." + testZone, Rrtype: dns.TypeTXT, Class: dns.ClassINET},
					Txt: []string{"123"},
				},
			},
		},
		"TXT multiple": {
			r: &Record{Name: "txt", TXT: [][]string{{"123", "456"}, {"abc", "def"}}},
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if have := tc.r.Records(testZone); !reflect.DeepEqual(have, tc.want) {
				t.Errorf("got %+v, want %+v", have, tc.want)
			}
		})
	}
}

func TestRecordValidate(t *testing.T) {
	tests := map[string]struct {
		r           *Record
		wantInvalid bool
	}{
		"all except cname valid": {
			r: &Record{
				Name: "test",
				A:    []netip.Addr{netip.MustParseAddr("192.0.2.1")},
				AAAA: []netip.Addr{netip.MustParseAddr("2001:db8::1")},
				TXT:  [][]string{{"abc"}},
			},
		},
		"invalid A": {
			r: &Record{
				Name: "test",
				A:    []netip.Addr{netip.MustParseAddr("2001:db8::1")},
			},
			wantInvalid: true,
		},
		"invalid AAAA": {
			r: &Record{
				Name: "test",
				AAAA: []netip.Addr{netip.MustParseAddr("192.0.2.1")},
			},
			wantInvalid: true,
		},
		"empty txt": {
			r: &Record{
				Name: "test",
				TXT: [][]string{
					{"abc", "def"},
					{},
					{"ghi", "jkl"},
				},
			},
			wantInvalid: true,
		},
		"cname": {
			r: &Record{
				Name:  "test",
				CNAME: "test2.example.com",
			},
		},
		"CNAME and A": {
			r: &Record{
				Name:  "test",
				CNAME: "test2.example.com",
				A:     []netip.Addr{netip.MustParseAddr("192.0.2.1")},
			},
			wantInvalid: true,
		},
		"CNAME and AAAA": {
			r: &Record{
				Name:  "test",
				CNAME: "test2.example.com",
				AAAA:  []netip.Addr{netip.MustParseAddr("2001:db8::1")},
			},
			wantInvalid: true,
		},
		"no name": {
			r: &Record{
				CNAME: "test2.example.com",
			},
			wantInvalid: true,
		},
		"no records": {
			r: &Record{
				Name: "test",
			},
			wantInvalid: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err := tc.r.Validate(); err != nil && !tc.wantInvalid {
				t.Errorf("expected to be valid but got error: %v", err)
			} else if err == nil && tc.wantInvalid {
				t.Errorf("expected to be invalid")
			}
		})
	}
}
