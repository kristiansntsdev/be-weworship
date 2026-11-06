package providers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseLinkURL(t *testing.T) {
	ok := []string{"https://www.jrchord.com/above-all", "http://chords.example/a?b=1", "https://x.com:443/a"}
	for _, raw := range ok {
		if _, err := ParseLinkURL(raw); err != nil {
			t.Errorf("ParseLinkURL(%q) = %v, want ok", raw, err)
		}
	}
	bad := []string{
		"", "chords.example/a", "ftp://x.com/a", "file:///etc/passwd", "javascript:alert(1)",
		"https://u:p@x.com/", "http://x.com:6379/", "http://127.0.0.1/", "http://10.0.0.5/", "http://[::1]/",
		"http://169.254.169.254/latest/meta-data/", "http://100.64.0.1/",
	}
	for _, raw := range bad {
		if _, err := ParseLinkURL(raw); err == nil {
			t.Errorf("ParseLinkURL(%q) = nil, want error", raw)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	public := []string{"8.8.8.8", "142.250.4.100", "2606:4700::1111"}
	for _, s := range public {
		if !isPublicIP(net.ParseIP(s)) {
			t.Errorf("isPublicIP(%s) = false, want true", s)
		}
	}
	private := []string{
		"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "0.1.2.3",
		"0.0.0.0", "224.0.0.1", "255.255.255.255", "::1", "fe80::1", "fd00::1", "::ffff:127.0.0.1", "64:ff9b::a00:1",
	}
	for _, s := range private {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("isPublicIP(%s) = true, want false", s)
		}
	}
}

// A hostname that resolves to a loopback address must be refused at dial time, before any request is sent.
func TestPageFetcher_RefusesHostnameResolvingToLoopback(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<pre>secret</pre>"))
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))

	// "localhost" passes ParseLinkURL (it's not an IP literal); only the dial check stands in the way.
	// The port is non-default, so call the client directly to exercise the dialer rather than the URL rule.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:"+port+"/", nil)
	_, err := NewPageFetcher().client.Do(req)

	// errors.Is (not a string match): SongLinkService maps this error by identity through url/net wrappers.
	if !errors.Is(err, ErrBlockedHost) {
		t.Fatalf("err = %v, want %v", err, ErrBlockedHost)
	}
	if hit {
		t.Fatal("request reached the loopback server")
	}
}
