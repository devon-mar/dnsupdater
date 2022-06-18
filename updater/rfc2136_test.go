package updater

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/miekg/dns"
)

const (
	testZone = "example.com"

	testNS1 = "ns1.example.com"
	testNS2 = "ns2.example.com"

	ns1ServFailName = "ns1fail.example.com."
	allFailName     = "ns2fail.example.com."
	exchangeErrName = "exchfail.examlpe.com."

	testTKEY = "tkey"
)

type testGSS struct {
	returnError bool

	credentials bool

	deletedContext string
}

// DeleteContext implements gssNegotiator
func (g *testGSS) DeleteContext(c string) error {
	g.deletedContext = c
	return nil
}

// Close implements updater.Updater
func (*testGSS) Close() error {
	return nil
}

// NegotiateContext implements gssNegotiator
func (g *testGSS) NegotiateContext(string) (string, time.Time, error) {
	if g.credentials {
		return "", time.Time{}, errors.New("expected credentials to be used")
	}

	if g.returnError {
		return "", time.Time{}, errors.New("returnError is true")
	}

	return "tkey", time.Time{}, nil
}

// NegotiateContextWithCredentials implements gssNegotiator
func (g *testGSS) NegotiateContextWithCredentials(string, string, string, string) (string, time.Time, error) {
	if !g.credentials {
		return "", time.Time{}, errors.New("expected no credentials to be used")
	}

	if g.returnError {
		return "", time.Time{}, errors.New("returnError is true")
	}

	return "tkey", time.Time{}, nil
}

func (g *testGSS) assert(t *testing.T) {
	t.Helper()
	if g.deletedContext != testTKEY && !g.returnError {
		t.Errorf("expected DeleteContext to be called with %q but got %q", testTKEY, g.deletedContext)
	}
}

type testDNS struct {
	exchanges map[string][]*dns.Msg
	want      map[string]int
	wantTSIG  bool
}

// Exchange implements dnsExchanger
func (d *testDNS) Exchange(msg *dns.Msg, server string) (*dns.Msg, time.Duration, error) {
	d.init()
	d.exchanges[server] = append(d.exchanges[server], msg)

	rcode := dns.RcodeSuccess

	name := msg.Ns[0].Header().Name

	if (server == testNS1 && name == ns1ServFailName) || name == allFailName {
		rcode = dns.RcodeServerFailure
	}
	if hasTSIG := (len(msg.Extra) != 0 && msg.Extra[0].Header().Rrtype == dns.TypeTSIG); d.wantTSIG && !hasTSIG {
		rcode = dns.RcodeNotAuth
	} else if !d.wantTSIG && hasTSIG {
		rcode = dns.RcodeBadSig
	}

	if name == exchangeErrName {
		return nil, time.Millisecond, errors.New("got exchange fail name")
	}

	return &dns.Msg{MsgHdr: dns.MsgHdr{Rcode: rcode}}, time.Millisecond, nil
}

func (d *testDNS) init() {
	if d.exchanges == nil {
		d.exchanges = map[string][]*dns.Msg{}
	}
}

func (d *testDNS) assert(t *testing.T) {
	t.Helper()
	if have, want := len(d.exchanges), len(d.want); have != want {
		t.Errorf("expected %d servers to be used but got %d: %+v", want, have, want)
	}
	for server, want := range d.want {
		if have := len(d.exchanges[server]); have != want {
			t.Errorf("got %d updates to %s, want %d", have, server, want)
		}
	}
}

func TestInsert(t *testing.T) {
	records := []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: "test"}}}

	tests := map[string]struct {
		wantError bool
		gss       *testGSS
		dns       *testDNS
		toInsert  []dns.RR

		username string
		password string
		domain   string
	}{
		"no gss": {
			dns:      &testDNS{want: map[string]int{testNS1: 1}},
			toInsert: records,
		},
		"gss": {
			gss:      &testGSS{credentials: false},
			dns:      &testDNS{want: map[string]int{testNS1: 1}, wantTSIG: true},
			toInsert: records,
		},
		"gss error": {
			gss:       &testGSS{returnError: true},
			dns:       &testDNS{want: map[string]int{}, wantTSIG: true},
			wantError: true,
		},
		"gss with cred": {
			gss:      &testGSS{credentials: true},
			dns:      &testDNS{want: map[string]int{testNS1: 1}, wantTSIG: true},
			username: "a", password: "a", domain: "a",
			toInsert: records,
		},
		"ns1 error": {
			dns:      &testDNS{want: map[string]int{testNS1: 1, testNS2: 1}},
			toInsert: []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: ns1ServFailName}}},
		},
		"all ns error": {
			dns:       &testDNS{want: map[string]int{testNS1: 1, testNS2: 1}},
			toInsert:  []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: allFailName}}},
			wantError: true,
		},
		"exchange error": {
			dns:       &testDNS{want: map[string]int{testNS1: 1, testNS2: 1}},
			toInsert:  []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: exchangeErrName}}},
			wantError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			u := &RFC2136Updater{
				servers:  []string{testNS1, testNS2},
				dns:      tc.dns,
				username: tc.username, password: tc.password, domain: tc.domain,
			}
			if tc.gss != nil {
				u.gss = tc.gss
			}
			err := u.Insert(testZone, tc.toInsert)
			if err == nil && tc.wantError {
				t.Errorf("expected an error")
			} else if err != nil && !tc.wantError {
				t.Errorf("expected no error but got: %v", err)
			}

			tc.dns.assert(t)
			if tc.gss != nil {
				tc.gss.assert(t)
			}
		})
	}
}

func TestNewRFC2136(t *testing.T) {
	servers := []string{testNS1, testNS2}
	u := NewRFC2136(servers)
	defer u.Close()
	if u.dns == nil {
		t.Errorf("expected dns to be non-nil")
	}
	if !reflect.DeepEqual(u.servers, servers) {
		t.Errorf("got servers %#v, want %#v", u.servers, servers)
	}

	if err := u.WithGSS(); err != nil {
		t.Errorf("got unexpected error: %v", err)
	}
	if u.gss == nil {
		t.Errorf("expected gss to be non-nil, but got %+v", u.gss)
	}

	username := "username"
	password := "username"
	domain := "domain"
	u.WithCredentials(username, password, domain)
	if u.username != username {
		t.Errorf("got username %q, want %q", u.username, username)
	}
	if u.password != password {
		t.Errorf("got password %q, want %q", u.password, password)
	}
	if u.domain != domain {
		t.Errorf("got domain %q, want %q", u.domain, domain)
	}
}
