package main

import (
	"math/rand"
	"strings"
	"testing"
)

const codeAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789-_"

// Property: every string of 6-10 chars from the alphabet is valid; any other length or char is invalid.
func TestCodeProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	gen := func(n int, alpha string) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = alpha[rng.Intn(len(alpha))]
		}
		return string(b)
	}
	for i := 0; i < 5000; i++ {
		n := 6 + rng.Intn(5)
		c := gen(n, codeAlphabet)
		if !validCode(c) {
			t.Fatalf("valid code rejected: %q", c)
		}
		if !validCode(strings.ToLower(strings.ToUpper(c))) {
			t.Fatalf("case fold broke %q", c)
		}
		bad := []string{gen(rng.Intn(6), codeAlphabet), gen(11+rng.Intn(20), codeAlphabet)}
		pos := rng.Intn(n)
		bc := []byte(c)
		bc[pos] = "!@#$%^&*()+=. /\\?~`'\"<>,;:[]{}|"[rng.Intn(31)]
		bad = append(bad, string(bc))
		for _, b := range bad {
			if validCode(b) {
				t.Fatalf("invalid code accepted: %q", b)
			}
		}
	}
	for w := range blocklist {
		if !isReserved(strings.ToLower(strings.ToUpper(w))) {
			t.Fatalf("blocklist %q", w)
		}
	}
}

func FuzzValidCode(f *testing.F) {
	for _, s := range []string{"abc123", "abc12", "abcdefghijk", "abc!23", "ABC123", "héllo12"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c := strings.ToLower(s)
		want := len(c) >= 6 && len(c) <= 10 && strings.Trim(c, codeAlphabet) == ""
		if validCode(c) != want {
			t.Fatalf("validCode(%q)=%v want %v", c, !want, want)
		}
	})
}

func TestURLProperties(t *testing.T) {
	good := []string{"https://example.com", "http://example.com/a?b=c#d", "HTTPS://EXAMPLE.COM", "https://a.b:8443/x", "http://192.168.0.1/"}
	bad := []string{"", "ftp://x.com", "javascript:alert(1)", "mailto:a@b.c", "https://", "https:///path", "example.com", "//example.com", "http:example.com", "data:text/html,hi", "https://exa mple.com", "http://[::1"}
	for _, u := range good {
		if _, ok := validURL(u); !ok {
			t.Errorf("rejected %q", u)
		}
	}
	for _, u := range bad {
		if _, ok := validURL(u); ok {
			t.Errorf("accepted %q", u)
		}
	}
	prefix := "https://example.com/"
	if _, ok := validURL(prefix + strings.Repeat("a", 2048-len(prefix))); !ok {
		t.Error("2048 chars rejected")
	}
	if _, ok := validURL(prefix + strings.Repeat("a", 2049-len(prefix))); ok {
		t.Error("2049 chars accepted")
	}
	for _, u := range []string{"http://t/abc123", "https://T/x", "http://t:80/", "http://localhost:8080/x", "http://LOCALHOST:8080"} {
		p, ok := validURL(u)
		if !ok || !isSelfReference(p) {
			t.Errorf("self reference not detected: %q", u)
		}
	}
	for _, u := range []string{"http://localhost:3000", "http://t.co/x", "http://tt/"} {
		p, _ := validURL(u)
		if isSelfReference(p) {
			t.Errorf("false self reference: %q", u)
		}
	}
}

func FuzzValidURL(f *testing.F) {
	for _, s := range []string{"https://example.com", "ftp://x.com", "javascript:alert(1)", "http://t/abc123"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		u, ok := validURL(s)
		if !ok {
			return
		}
		if len(s) > maxURLLen {
			t.Fatalf("too long accepted")
		}
		sc := strings.ToLower(u.Scheme)
		if sc != "http" && sc != "https" || u.Hostname() == "" {
			t.Fatalf("accepted %q (scheme %q host %q)", s, u.Scheme, u.Host)
		}
	})
}
