// OpenFGA spike harness: synthetic tuple load generator + Check/ListObjects benchmark.
// Usage:
//
//	go run . setup                        # create store + write authorization model
//	go run . load <scale>                 # load synthetic tuples (scale=1 -> v1 envelope, 10 -> 10x)
//	go run . bench <scale> <requests> <concurrency>
//
// Env: FGA_API_URL (default http://localhost:8080), FGA_STORE_ID (set by setup)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var apiURL = env("FGA_API_URL", "http://localhost:8080")

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type tupleKey struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}

func post(path string, body any) (map[string]any, int, string) {
	b, _ := json.Marshal(body)
	resp, err := http.Post(apiURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	json.Unmarshal(data, &out)
	return out, resp.StatusCode, string(data)
}

func mustPost(path string, body any) map[string]any {
	out, code, raw := post(path, body)
	if code >= 300 {
		panic(fmt.Sprintf("POST %s -> %d: %s", path, code, raw))
	}
	return out
}

func storeID() string {
	id := os.Getenv("FGA_STORE_ID")
	if id == "" {
		panic("FGA_STORE_ID not set (run setup first)")
	}
	return id
}

func setup() {
	res := mustPost("/stores", map[string]any{"name": "m1spike"})
	id := res["id"].(string)
	model, err := os.ReadFile("model.json")
	if err != nil {
		panic(err)
	}
	var m map[string]any
	json.Unmarshal(model, &m)
	mustPost("/stores/"+id+"/authorization-models", m)
	fmt.Println(id)
}

// topology at scale=1 (v1 envelope, plan §12.1/4: 100 clusters, 5k resource instances)
func generate(scale int) []tupleKey {
	orgs := 20
	teamsPerOrg := 4
	users := 500
	tenantsPerOrg := 2
	clustersTotal := 100 * scale
	accountsPerTenant := 1
	itemsPerAccount := 25
	instancesTotal := 5000 * scale

	var tuples []tupleKey
	add := func(u, r, o string) { tuples = append(tuples, tupleKey{u, r, o}) }

	var tenants []string
	var clusters []string
	var items []string
	for o := 0; o < orgs; o++ {
		org := fmt.Sprintf("organization:org-%d", o)
		adminTeam := fmt.Sprintf("team:org-%d-admins", o)
		add(adminTeam+"#member", "admin", org)
		platTeam := fmt.Sprintf("team:org-%d-platform", o)
		add(platTeam+"#member", "viewer", org)
		for t := 0; t < teamsPerOrg; t++ {
			team := fmt.Sprintf("team:org-%d-team-%d", o, t)
			add(team+"#member", "viewer", org)
			for u := 0; u < users/(orgs*teamsPerOrg); u++ {
				add(fmt.Sprintf("user:u-%d", (o*teamsPerOrg+t)*users/(orgs*teamsPerOrg)+u), "member", team)
			}
		}
		add(fmt.Sprintf("user:u-admin-%d", o), "member", adminTeam)
		for tn := 0; tn < tenantsPerOrg; tn++ {
			tenant := fmt.Sprintf("tenant:org-%d-t-%d", o, tn)
			add(org, "parent", tenant)
			add(fmt.Sprintf("team:org-%d-team-%d#member", o, tn%teamsPerOrg), "admin", tenant)
			tenants = append(tenants, tenant)
			add(tenant, "parent", fmt.Sprintf("tenant_zone:zone-%d-%d", o, tn))
		}
	}
	for c := 0; c < clustersTotal; c++ {
		tenant := tenants[c%len(tenants)]
		cluster := fmt.Sprintf("cluster:c-%d", c)
		add(tenant, "parent", cluster)
		clusters = append(clusters, cluster)
		for a := 0; a < accountsPerTenant; a++ {
			acct := fmt.Sprintf("cloud_account:acct-%d-%d", c, a)
			add(cluster, "parent", acct)
			for i := 0; i < itemsPerAccount; i++ {
				item := fmt.Sprintf("catalog_item:ci-%d-%d-%d", c, a, i)
				add(acct, "parent", item)
				items = append(items, item)
			}
		}
	}
	ownerTeams := []string{}
	for o := 0; o < orgs; o++ {
		ownerTeams = append(ownerTeams, fmt.Sprintf("team:org-%d-team-0#member", o))
	}
	for r := 0; r < instancesTotal; r++ {
		item := items[r%len(items)]
		ri := fmt.Sprintf("resource_instance:ri-%d", r)
		add(item, "parent", ri)
		add(ownerTeams[r%len(ownerTeams)], "editor", ri)
	}
	return tuples
}

func load(scale, startIdx int) {
	tuples := generate(scale)
	id := storeID()
	const chunk = 100
	start := time.Now()
	for i := startIdx; i < len(tuples); i += chunk {
		end := i + chunk
		if end > len(tuples) {
			end = len(tuples)
		}
		var code int
		var raw string
		for attempt := 0; ; attempt++ {
			_, code, raw = post("/stores/"+id+"/write", map[string]any{
				"writes": map[string]any{"tuple_keys": tuples[i:end]},
			})
			if code < 300 {
				break
			}
			if strings.Contains(raw, "already exists") {
				break // idempotent resume: chunk partially written before
			}
			if strings.Contains(raw, "deadline_exceeded") && attempt < 5 {
				time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
				continue
			}
			panic(fmt.Sprintf("write -> %d: %s", code, raw))
		}
		if (i/chunk)%50 == 0 {
			fmt.Printf("wrote %d/%d tuples\n", end, len(tuples))
		}
	}
	fmt.Printf("loaded %d tuples in %s\n", len(tuples), time.Since(start).Round(time.Millisecond))
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	idx := int(p * float64(len(sorted)-1))
	return sorted[idx]
}

func bench(scale, requests, concurrency int) {
	id := storeID()
	instancesTotal := 5000 * scale
	usersTotal := 500 + 20

	run := func(name string, fn func(u string, ri int) error) {
		lat := make([]time.Duration, requests)
		var errs int64
		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)
		start := time.Now()
		for i := 0; i < requests; i++ {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()
				rng := rand.New(rand.NewSource(42 + int64(i)))
				u := fmt.Sprintf("user:u-%d", rng.Intn(usersTotal))
				ri := rng.Intn(instancesTotal)
				t0 := time.Now()
				if err := fn(u, ri); err != nil {
					atomic.AddInt64(&errs, 1)
				}
				lat[i] = time.Since(t0)
			}(i)
		}
		wg.Wait()
		wall := time.Since(start)
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		fmt.Printf("%s: n=%d conc=%d errors=%d wall=%s rps=%.0f p50=%s p95=%s p99=%s max=%s\n",
			name, requests, concurrency, errs, wall.Round(time.Millisecond),
			float64(requests)/wall.Seconds(),
			percentile(lat, .50).Round(time.Microsecond),
			percentile(lat, .95).Round(time.Microsecond),
			percentile(lat, .99).Round(time.Microsecond),
			lat[len(lat)-1].Round(time.Microsecond))
	}

	check := func(u, rel, obj string) error {
		_, code, raw := post("/stores/"+id+"/check", map[string]any{
			"tuple_key": map[string]string{"user": u, "relation": rel, "object": obj},
		})
		if code >= 300 {
			return fmt.Errorf("%d: %s", code, raw)
		}
		return nil
	}
	listObjects := func(u, typ string) error {
		_, code, raw := post("/stores/"+id+"/list-objects", map[string]any{
			"type": typ, "relation": "viewer", "user": u,
		})
		if code >= 300 {
			return fmt.Errorf("%d: %s", code, raw)
		}
		return nil
	}

	run("Check(editor,resource_instance)", func(u string, ri int) error {
		return check(u, "editor", fmt.Sprintf("resource_instance:ri-%d", ri))
	})
	run("Check(viewer,resource_instance)", func(u string, ri int) error {
		return check(u, "viewer", fmt.Sprintf("resource_instance:ri-%d", ri))
	})
	run("ListObjects(viewer,resource_instance)", func(u string, _ int) error {
		return listObjects(u, "resource_instance")
	})
	run("ListObjects(viewer,cluster)", func(u string, _ int) error {
		return listObjects(u, "cluster")
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: setup | load <scale> [startIdx] | bench <scale> <requests> <concurrency>")
		os.Exit(1)
	}
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	switch os.Args[1] {
	case "setup":
		setup()
	case "load":
		start := 0
		if len(os.Args) > 3 {
			start = atoi(os.Args[3])
		}
		load(atoi(os.Args[2]), start)
	case "bench":
		bench(atoi(os.Args[2]), atoi(os.Args[3]), atoi(os.Args[4]))
	}
}
