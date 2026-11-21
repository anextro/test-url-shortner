package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type loadResult struct {
	lat  []time.Duration
	errs int
}

// runLoad issues qps requests/sec for dur, open-loop, and returns latencies.
func runLoad(t *testing.T, qps int, dur time.Duration, mk func(i int) *http.Request, okStatus int) loadResult {
	client := &http.Client{
		Transport:     &http.Transport{MaxIdleConns: 512, MaxIdleConnsPerHost: 512},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Timeout:       5 * time.Second,
	}
	total := int(float64(qps) * dur.Seconds())
	interval := time.Second / time.Duration(qps)
	var mu sync.Mutex
	var res loadResult
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < total; i++ {
		if d := time.Until(start.Add(time.Duration(i) * interval)); d > 0 {
			time.Sleep(d)
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := mk(i)
			t0 := time.Now()
			r, err := client.Do(req)
			el := time.Since(t0)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.errs++
				return
			}
			r.Body.Close()
			if r.StatusCode != okStatus {
				res.errs++
			}
			res.lat = append(res.lat, el)
		}(i)
	}
	wg.Wait()
	achieved := float64(total) / time.Since(start).Seconds()
	t.Logf("sent %d requests, achieved %.0f QPS", total, achieved)
	return res
}

func pct(l []time.Duration, p float64) time.Duration {
	s := append([]time.Duration(nil), l...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	idx := int(float64(len(s))*p+0.5) - 1
	if idx < 0 {
		idx = 0
	}
	return s[idx]
}

func TestLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("load test skipped in -short mode")
	}
	_, ts := newTestServer(t)
	dur := 10 * time.Second
	if v := os.Getenv("LOAD_SECONDS"); v != "" {
		d, err := time.ParseDuration(v + "s")
		if err == nil {
			dur = d
		}
	}
	const nCodes = 1000
	for i := 0; i < nCodes; i++ {
		if r := doCreate(t, ts.URL, uniqueIP(), map[string]string{"url": fmt.Sprintf("https://example.com/%d", i), "code": fmt.Sprintf("load%05d", i)}); r.Status != 201 {
			t.Fatalf("seed %d: %+v", i, r)
		}
	}

	var redir, create loadResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		redir = runLoad(t, 1000, dur, func(i int) *http.Request {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/load%05d", ts.URL, i%nCodes), nil)
			return req
		}, 301)
	}()
	go func() {
		defer wg.Done()
		create = runLoad(t, 100, dur, func(i int) *http.Request {
			body := fmt.Sprintf(`{"url":"https://example.org/%d","code":"new%06d"}`, i, i)
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/links", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", uniqueIP())
			return req
		}, 201)
	}()
	wg.Wait()

	rp50, rp99 := pct(redir.lat, 0.50), pct(redir.lat, 0.99)
	cp50, cp99 := pct(create.lat, 0.50), pct(create.lat, 0.99)
	fmt.Printf("LOAD redirect: n=%d errors=%d p50=%v p99=%v (target p99 <= 10ms @ 1000 QPS)\n", len(redir.lat), redir.errs, rp50, rp99)
	fmt.Printf("LOAD create:   n=%d errors=%d p50=%v p99=%v (@ 100 QPS)\n", len(create.lat), create.errs, cp50, cp99)
	if redir.errs > 0 || create.errs > 0 {
		t.Errorf("errors: redirect=%d create=%d", redir.errs, create.errs)
	}
	if rp99 > 10*time.Millisecond {
		t.Errorf("redirect p99 %v > 10ms", rp99)
	}
}

