// Command loadtest is a small HTTP load generator for the performance tests of the thesis.
//
//	go run ./cmd/loadtest -url http://localhost:8080 -email admin@school.local -password ... -c 20 -d 15s
//
// It logs in once, then hammers a set of read endpoints concurrently and prints
// throughput and latency percentiles per endpoint.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type result struct {
	d  time.Duration
	ok bool
}

func main() {
	base := flag.String("url", "http://localhost:8080", "base URL of the server")
	email := flag.String("email", "", "login e-mail (needed for authenticated endpoints)")
	password := flag.String("password", "", "login password")
	conc := flag.Int("c", 10, "concurrent workers")
	dur := flag.Duration("d", 10*time.Second, "test duration per endpoint")
	paths := flag.String("paths", "/health,/,/api/news,/api/classes,/api/subjects", "comma separated endpoints")
	flag.Parse()

	client := &http.Client{Timeout: 15 * time.Second}
	token := ""
	if *email != "" {
		t, err := login(client, *base, *email, *password)
		if err != nil {
			fmt.Fprintln(os.Stderr, "login failed:", err)
			os.Exit(1)
		}
		token = t
	}

	fmt.Printf("%-22s %8s %8s %8s %8s %8s %8s %6s\n", "endpoint", "req", "req/s", "p50", "p95", "p99", "max", "err%")
	failed := false
	for _, p := range strings.Split(*paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		res := run(client, *base+p, token, *conc, *dur)
		s := summarize(res, *dur)
		fmt.Printf("%-22s %8d %8.0f %8s %8s %8s %8s %5.1f%%\n", p, s.n, s.rps, s.p50, s.p95, s.p99, s.max, s.errPct)
		if s.errPct > 1 {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func login(c *http.Client, base, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := c.Post(base+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Data.Token == "" {
		return "", fmt.Errorf("status %d, no token", resp.StatusCode)
	}
	return out.Data.Token, nil
}

func run(c *http.Client, url, token string, workers int, d time.Duration) []result {
	var mu sync.Mutex
	var all []result
	var wg sync.WaitGroup
	deadline := time.Now().Add(d)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local []result
			for time.Now().Before(deadline) {
				req, _ := http.NewRequest("GET", url, nil)
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				start := time.Now()
				resp, err := c.Do(req)
				ok := err == nil
				if err == nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					ok = resp.StatusCode < 400
				}
				local = append(local, result{time.Since(start), ok})
			}
			mu.Lock()
			all = append(all, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return all
}

type summary struct {
	n                  int
	rps, errPct        float64
	p50, p95, p99, max time.Duration
}

func summarize(res []result, d time.Duration) summary {
	s := summary{n: len(res)}
	if len(res) == 0 {
		s.errPct = 100
		return s
	}
	ds := make([]time.Duration, 0, len(res))
	bad := 0
	for _, r := range res {
		ds = append(ds, r.d)
		if !r.ok {
			bad++
		}
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	q := func(p float64) time.Duration { return ds[min(len(ds)-1, int(float64(len(ds))*p))] }
	s.rps = float64(len(res)) / d.Seconds()
	s.errPct = float64(bad) * 100 / float64(len(res))
	s.p50, s.p95, s.p99, s.max = q(0.50).Round(time.Microsecond), q(0.95).Round(time.Microsecond), q(0.99).Round(time.Microsecond), ds[len(ds)-1].Round(time.Microsecond)
	return s
}
