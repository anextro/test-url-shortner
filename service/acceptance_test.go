package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestAcceptance runs spec acceptance tests 1-18 in order against real Postgres.
func TestAcceptance(t *testing.T) {
	_, ts := newTestServer(t)
	base := ts.URL
	link := func(url, code string) map[string]string { return map[string]string{"url": url, "code": code} }
	expect := func(t *testing.T, got resp, status int, errCode string) {
		t.Helper()
		if got.Status != status || got.ErrCode != errCode {
			t.Fatalf("got %d %q, want %d %q", got.Status, got.ErrCode, status, errCode)
		}
	}

	t.Run("01_create_new", func(t *testing.T) {
		r := doCreate(t, base, uniqueIP(), link("https://example.com", "abc123"))
		expect(t, r, 201, "")
		if r.ShortURL != "t/abc123" {
			t.Fatalf("shortUrl %q", r.ShortURL)
		}
	})
	t.Run("02_same_again", func(t *testing.T) {
		r := doCreate(t, base, uniqueIP(), link("https://example.com", "abc123"))
		expect(t, r, 200, "")
		if r.ShortURL != "t/abc123" {
			t.Fatalf("shortUrl %q", r.ShortURL)
		}
	})
	t.Run("03_second_code_same_url", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "mylink1")), 201, "")
	})
	t.Run("04_code_taken", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://other.com", "abc123")), 409, "CODE_TAKEN")
	})
	t.Run("05_code_taken_case_insensitive", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://other.com", "ABC123")), 409, "CODE_TAKEN")
	})
	t.Run("06_code_too_short", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "abc12")), 400, "INVALID_CODE")
	})
	t.Run("07_code_too_long", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "abcdefghijk")), 400, "INVALID_CODE")
	})
	t.Run("08_code_bad_char", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "abc!23")), 400, "INVALID_CODE")
	})
	t.Run("09_reserved_code", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "health")), 400, "RESERVED_CODE")
	})
	t.Run("10_ftp_url", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("ftp://x.com", "goodcode1")), 400, "INVALID_URL")
	})
	t.Run("11_javascript_url", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("javascript:alert(1)", "goodcode1")), 400, "INVALID_URL")
	})
	t.Run("12_url_too_long", func(t *testing.T) {
		long := "https://example.com/" + strings.Repeat("a", 2049-len("https://example.com/"))
		expect(t, doCreate(t, base, uniqueIP(), link(long, "goodcode1")), 400, "INVALID_URL")
	})
	t.Run("13_self_reference", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("http://t/abc123", "goodcode1")), 400, "SELF_REFERENCE")
	})
	t.Run("14_bad_url_and_bad_code", func(t *testing.T) {
		expect(t, doCreate(t, base, uniqueIP(), link("ftp://x.com", "abc!")), 400, "INVALID_URL")
	})
	t.Run("15_rate_limited", func(t *testing.T) {
		ip := uniqueIP()
		for i := 0; i < 10; i++ {
			r := doCreate(t, base, ip, link("https://example.com", "abc123"))
			if r.Status == http.StatusTooManyRequests {
				t.Fatalf("request %d rate limited early", i+1)
			}
		}
		expect(t, doCreate(t, base, ip, link("https://example.com", "abc123")), 429, "RATE_LIMITED")
		// a different IP is unaffected
		expect(t, doCreate(t, base, uniqueIP(), link("https://example.com", "abc123")), 200, "")
	})
	t.Run("16_redirect", func(t *testing.T) {
		r := doGet(t, base, "/abc123")
		if r.Status != 301 || r.Location != "https://example.com" {
			t.Fatalf("got %d %q", r.Status, r.Location)
		}
	})
	t.Run("17_redirect_uppercase", func(t *testing.T) {
		r := doGet(t, base, "/ABC123")
		if r.Status != 301 || r.Location != "https://example.com" {
			t.Fatalf("got %d %q", r.Status, r.Location)
		}
	})
	t.Run("18_not_found", func(t *testing.T) {
		expect(t, doGet(t, base, "/zzzzzz"), 404, "NOT_FOUND")
	})
}

// TestExtras covers spec rules not in the numbered table.
func TestExtras(t *testing.T) {
	_, ts := newTestServer(t)
	base := ts.URL
	t.Run("malformed_json", func(t *testing.T) {
		r := doCreate(t, base, uniqueIP(), "{not json")
		if r.Status != 400 || r.ErrCode != "INVALID_URL" {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("self_reference_localhost_8080", func(t *testing.T) {
		r := doCreate(t, base, uniqueIP(), map[string]string{"url": "http://localhost:8080/x", "code": "goodcode1"})
		if r.Status != 400 || r.ErrCode != "SELF_REFERENCE" {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("redirect_served_from_db_on_cache_miss", func(t *testing.T) {
		// Create via one server, read via a second server with a cold cache.
		if r := doCreate(t, base, uniqueIP(), map[string]string{"url": "https://cold.example", "code": "coldcode"}); r.Status != 201 {
			t.Fatalf("create %+v", r)
		}
		_, ts2 := newTestServerNoReset(t)
		if r := doGet(t, ts2.URL, "/coldcode"); r.Status != 301 || r.Location != "https://cold.example" {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("cors_preflight", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, base+"/links", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != 204 || r.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Fatalf("got %d %q", r.StatusCode, r.Header.Get("Access-Control-Allow-Origin"))
		}
	})
	t.Run("cors_other_origin_not_allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, base+"/links", nil)
		req.Header.Set("Origin", "http://evil.example")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("unexpected CORS header")
		}
	})
}
