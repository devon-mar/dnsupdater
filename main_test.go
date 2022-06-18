package main

import (
	"fmt"
	"net/netip"
	"sort"
	"testing"

	"github.com/devon-mar/dnsupdater/config"

	"github.com/miekg/dns"
)

const testZone = "example.com"

type testUpdater struct {
	insertions map[string][][]dns.RR
	allRecords []dns.RR
}

func (u *testUpdater) init() {
	if u.insertions == nil {
		u.insertions = make(map[string][][]dns.RR)
	}
}

// Close implements updater.Updater
func (*testUpdater) Close() error {
	return nil
}

// Insert implements updater.Updater
func (u *testUpdater) Insert(z string, rrSet []dns.RR) error {
	u.init()
	u.insertions[z] = append(u.insertions[z], rrSet)

	u.allRecords = append(u.allRecords, rrSet...)

	return nil
}

// WithCredentials implements updater.Updater
func (u *testUpdater) WithCredentials(string, string, string) {
	u.init()
}

// WithGSS implements updater.Updater
func (u *testUpdater) WithGSS() error {
	u.init()
	return nil
}

func (u *testUpdater) assert(t *testing.T, want map[string][][]dns.RR) {
	t.Helper()
	if have, want := len(u.insertions), len(want); have != want {
		t.Errorf("expected %d zones but got %d", want, have)
	}
	for zone, insertions := range want {
		have, ok := u.insertions[zone]
		if !ok {
			t.Errorf("expected insertion for zone %q", zone)
			continue
		}
		if haveLen, wantLen := len(have), len(insertions); haveLen != wantLen {
			t.Errorf("expected %d insertions for zone %q, but got %d", wantLen, zone, haveLen)
			continue
		}
		for idx, insertion := range insertions {
			assertRRSet(t, have[idx], insertion)
		}
	}
}

func assertRRSet(t *testing.T, h []dns.RR, w []dns.RR) {
	t.Helper()
	if haveLen, wantLen := len(h), len(w); haveLen != wantLen {
		t.Errorf("got %d records, want %d records", haveLen, wantLen)
		return
	}

	var have []dns.RR
	copy(have, h)
	var want []dns.RR
	copy(want, w)

	sort.Slice(want, func(i, j int) bool { return want[i].String() < want[j].String() })
	sort.Slice(have, func(i, j int) bool { return have[i].String() < have[j].String() })

	for idx, rr := range want {
		if rr.String() != have[idx].String() {
			t.Errorf("idx=%d: expected %q, got %q", idx, rr, have[idx])
		}
	}
}

func mustParseIPs(ips ...string) []netip.Addr {
	ret := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		ret = append(ret, netip.MustParseAddr(ip))
	}
	return ret
}

func testA(name string, ip string) dns.RR {
	r := &config.Record{Name: name, A: []netip.Addr{netip.MustParseAddr(ip)}}
	return r.Records(testZone)[0]
}

func testCNAME(name string, target string) dns.RR {
	r := &config.Record{Name: name, CNAME: target}
	return r.Records(testZone)[0]
}

func TestInsert(t *testing.T) {
	tests := map[string]struct {
		zones map[string]*config.Zone
		want  map[string][][]dns.RR
	}{
		"simple": {
			zones: map[string]*config.Zone{
				"example.com": {
					Records: map[string]*config.Record{
						"www": {
							Name: "www",
							A:    mustParseIPs("192.0.2.1"),
						},
					},
				},
			},
			want: map[string][][]dns.RR{
				"example.com.": {
					{testA("www", "192.0.2.1")},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			u := &testUpdater{}
			insert(u, tc.zones)
			u.assert(t, tc.want)
		})
	}
}

func TestInsertBatch(t *testing.T) {
	tests := map[string]struct {
		zones map[string]*config.Zone
		size  int
		want  map[string][][]dns.RR
	}{
		"single record": {
			size: 1,
			zones: map[string]*config.Zone{
				"example.com": {
					Records: map[string]*config.Record{
						"www": {
							Name: "www",
							A:    mustParseIPs("192.0.2.1"),
						},
					},
				},
			},
			want: map[string][][]dns.RR{
				"example.com.": {
					{testA("www", "192.0.2.1")},
				},
			},
		},
		"size=1": {
			size: 1,
			zones: map[string]*config.Zone{
				"example.com": {
					Records: map[string]*config.Record{
						"www": {
							Name: "www",
							A:    mustParseIPs("192.0.2.1", "192.0.2.2"),
						},
						"www2": {
							Name:  "www2",
							CNAME: "www.example.com",
						},
					},
				},
			},
			want: map[string][][]dns.RR{
				"example.com.": {
					{testA("www", "192.0.2.1")},
					{testA("www", "192.0.2.2")},
					{testCNAME("www2", "www.example.com.")},
				},
			},
		},
		"size=2": {
			size: 2,
			zones: map[string]*config.Zone{
				"example.com": {
					Records: map[string]*config.Record{
						"www": {
							Name: "www",
							A:    mustParseIPs("192.0.2.1", "192.0.2.2"),
						},
						"www2": {
							Name:  "www2",
							CNAME: "www.example.com",
						},
					},
				},
			},
			want: map[string][][]dns.RR{
				"example.com.": {
					{testA("www", "192.0.2.1"), testA("www", "192.0.2.2")},
					{testCNAME("www2", "www.example.com.")},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			u := &testUpdater{}
			insertBatch(u, tc.zones, tc.size)
			u.assert(t, tc.want)
		})
	}
}

func TestInsertBatch2(t *testing.T) {
	zones := map[string]*config.Zone{
		"example.com": {
			Records: map[string]*config.Record{
				// yes, this is not valid.
				"a": {
					Name: "a",
				},
				"b": {
					Name: "b",
					A:    mustParseIPs("192.0.2.1"),
				},
				"c": {
					Name:  "c",
					CNAME: "www.example.com",
				},
				"d": {
					Name: "d",
					A:    mustParseIPs("192.0.2.1", "192.0.2.2"),
				},
				"e": {
					Name: "e",
					A:    mustParseIPs("192.0.2.1", "192.0.2.2", "192.0.2.3"),
				},
				// yes, this is not valid.
				"f": {
					Name: "f",
				},
				"g": {
					Name: "g",
					A:    mustParseIPs("192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4"),
				},
			},
		},
	}

	wantRecords := []dns.RR{
		testA("b", "192.0.2.1"),
		testCNAME("c", "www.example.com."),
		testA("d", "192.0.2.1"),
		testA("d", "192.0.2.2"),
		testA("e", "192.0.2.1"),
		testA("e", "192.0.2.2"),
		testA("e", "192.0.2.3"),
		testA("g", "192.0.2.1"),
		testA("g", "192.0.2.2"),
		testA("g", "192.0.2.3"),
		testA("g", "192.0.2.4"),
	}

	// There are 11 records
	for i := 1; i <= 12; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := &testUpdater{}
			insertBatch(u, zones, i)

			assertRRSet(t, u.allRecords, wantRecords)
		})
	}
}
