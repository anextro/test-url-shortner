package main

import (
	"fmt"
	"sync"
	"testing"
)

// 50 goroutines create the same code with different URLs at once: exactly one 201, rest 409.
func TestConcurrentCreateSameCode(t *testing.T) {
	_, ts := newTestServer(t)
	const n = 50
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]resp, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = doCreate(t, ts.URL, uniqueIP(), map[string]string{"url": fmt.Sprintf("https://site%d.example", i), "code": "racecode"})
		}(i)
	}
	close(start)
	wg.Wait()
	counts := map[int]int{}
	winner := -1
	for i, r := range results {
		counts[r.Status]++
		if r.Status == 201 {
			winner = i
		}
	}
	if counts[201] != 1 || counts[409] != n-1 {
		t.Fatalf("status counts %v, want 1x201 and %dx409", counts, n-1)
	}
	if r := doGet(t, ts.URL, "/racecode"); r.Location != fmt.Sprintf("https://site%d.example", winner) {
		t.Fatalf("redirect %q does not match winner %d", r.Location, winner)
	}
	t.Logf("status counts: %v", counts)
}
