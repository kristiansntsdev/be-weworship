package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html/charset"
)

const (
	pageFetchTimeout = 12 * time.Second
	maxPageBytes     = 3 << 20
	maxPageRedirects = 5
	pageUserAgent    = "Mozilla/5.0 (compatible; WeWorshipBot/1.0; +https://weworship.id)"
)

var (
	// ErrInvalidLink means the link is not something we will fetch at all (bad scheme, port, credentials…).
	ErrInvalidLink = errors.New("link must be a public http(s) URL")
	// ErrBlockedHost means the link resolved to a private, loopback or reserved address.
	ErrBlockedHost = errors.New("link points to a private or reserved address")
	// ErrNotHTML means the link answered with something other than a web page (PDF, image, JSON…).
	ErrNotHTML = errors.New("link is not a web page")
)

// Ranges that IP.IsGlobalUnicast/IsPrivate still let through but must never be reachable from a user-supplied link.
var reservedNets = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8",     // "this network"
		"100.64.0.0/10", // carrier-grade NAT
		"192.0.0.0/24",  // IETF protocol assignments
		"198.18.0.0/15", // benchmarking
		"240.0.0.0/4",   // reserved
		"64:ff9b::/96",  // NAT64 — can embed a private IPv4
		"2002::/16",     // 6to4 — can embed a private IPv4
	} {
		_, n, _ := net.ParseCIDR(cidr)
		out = append(out, n)
	}
	return out
}()

func isPublicIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, n := range reservedNets {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

// ParseLinkURL accepts only absolute http(s) links on the default ports, without embedded credentials.
func ParseLinkURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, ErrInvalidLink
	}
	if err := validateLinkURL(u); err != nil {
		return nil, err
	}
	return u, nil
}

func validateLinkURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidLink
	}
	if u.Hostname() == "" || u.User != nil {
		return ErrInvalidLink
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return ErrInvalidLink
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !isPublicIP(ip) {
		return ErrBlockedHost
	}
	return nil
}

// FetchedPage is a downloaded HTML page, already decoded to UTF-8.
type FetchedPage struct {
	URL  *url.URL // final URL after redirects
	HTML []byte   // truncated at maxPageBytes
}

// PageFetcher downloads user-supplied links safely (no internal addresses, bounded size and time).
type PageFetcher struct {
	client *http.Client
}

func NewPageFetcher() *PageFetcher {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		// Control runs after DNS resolution on every dial, redirects included, so the address checked is the
		// one actually connected to. Checking the hostname up front instead would be open to DNS rebinding.
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip == nil || !isPublicIP(ip) {
				return ErrBlockedHost
			}
			return nil
		},
	}
	transport := &http.Transport{
		// No env proxy: a proxy would make the dial check above inspect the proxy's address, not the target's.
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		ForceAttemptHTTP2:      true,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  8 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
	}
	return &PageFetcher{client: &http.Client{
		Timeout:   pageFetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxPageRedirects {
				return errors.New("too many redirects")
			}
			return validateLinkURL(req.URL)
		},
	}}
}

func (f *PageFetcher) Fetch(ctx context.Context, rawURL string) (*FetchedPage, error) {
	u, err := ParseLinkURL(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrInvalidLink
	}
	req.Header.Set("User-Agent", pageUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")
	req.Header.Set("Accept-Language", "id,en;q=0.8")

	res, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("page returned HTTP %d", res.StatusCode)
	}
	contentType := res.Header.Get("Content-Type")
	if contentType != "" {
		if mt, _, _ := mime.ParseMediaType(contentType); mt != "text/html" && mt != "application/xhtml+xml" {
			return nil, ErrNotHTML
		}
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxPageBytes))
	if err != nil {
		return nil, err
	}
	// Indonesian chord blogs are mostly UTF-8, but older ones still declare windows-1252 / iso-8859-1.
	decoded, err := charset.NewReader(bytes.NewReader(raw), contentType)
	if err != nil {
		return nil, fmt.Errorf("unsupported page encoding: %w", err)
	}
	body, err := io.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	return &FetchedPage{URL: res.Request.URL, HTML: body}, nil
}
