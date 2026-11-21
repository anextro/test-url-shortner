package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ipSeq atomic.Int64

// uniqueIP returns a fresh X-Forwarded-For value so rate limits don't collide across tests.
func uniqueIP() string {
	n := ipSeq.Add(1)
	return "10." + itoa(int(n>>16&255)) + "." + itoa(int(n>>8&255)) + "." + itoa(int(n&255))
}

func itoa(i int) string { return fmtInt(i) }
func fmtInt(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// newTestServer connects to TEST_DATABASE_URL, resets the table and returns a fresh server.
func newTestServer(t testing.TB) (*Server, *httptest.Server) {
	t.Helper()
	return setupServer(t, true)
}

func newTestServerNoReset(t testing.TB) (*Server, *httptest.Server) {
	t.Helper()
	return setupServer(t, false)
}

func setupServer(t testing.TB, reset bool) (*Server, *httptest.Server) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set (run via `make test`)")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 60
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	st := NewPGStore(pool)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if !reset {
	} else if _, err := pool.Exec(ctx, "TRUNCATE links"); err != nil {
		t.Fatal(err)
	}
	s := NewServer(st)
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return s, ts
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

type resp struct {
	Status   int
	ShortURL string
	ErrCode  string
	Location string
}

func doCreate(t testing.TB, base, ip string, body any) resp {
	t.Helper()
	var buf []byte
	if s, ok := body.(string); ok {
		buf = []byte(s)
	} else {
		buf, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(http.MethodPost, base+"/links", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	return send(t, req)
}

func doGet(t testing.TB, base, path string) resp {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+path, nil)
	return send(t, req)
}

func send(t testing.TB, req *http.Request) resp {
	t.Helper()
	r, err := noRedirect.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out struct {
		ShortURL string `json:"shortUrl"`
		Error    struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	return resp{r.StatusCode, out.ShortURL, out.Error.Code, r.Header.Get("Location")}
}
