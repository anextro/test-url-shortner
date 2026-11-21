package main

import (
	"net/url"
	"strings"
)

const maxURLLen = 2048

var blocklist = func() map[string]struct{} {
	m := map[string]struct{}{}
	for _, w := range strings.Fields("health status metrics static assets favicon sitemap signup signin logout account settings password profile support security verify billing payment") {
		m[w] = struct{}{}
	}
	return m
}()

// validURL reports whether raw is a well-formed absolute http(s) URL of at most 2048 chars.
func validURL(raw string) (*url.URL, bool) {
	if raw == "" || len(raw) > maxURLLen {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	s := strings.ToLower(u.Scheme)
	if (s != "http" && s != "https") || u.Host == "" || u.Opaque != "" || u.Hostname() == "" {
		return nil, false
	}
	return u, true
}

func isSelfReference(u *url.URL) bool {
	return strings.EqualFold(u.Hostname(), "t") || strings.EqualFold(u.Host, "localhost:8080")
}

// validCode checks format on the already-lowercased code.
func validCode(c string) bool {
	if len(c) < 6 || len(c) > 10 {
		return false
	}
	for i := 0; i < len(c); i++ {
		ch := c[i]
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func isReserved(c string) bool {
	_, ok := blocklist[c]
	return ok
}
