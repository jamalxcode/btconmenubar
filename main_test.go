package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFormatPrice(t *testing.T) {
	cases := []struct {
		p    float64
		cur  string
		want string
	}{
		{91234.56, "usd", "$91,235"},
		{1234567, "eur", "€1,234,567"},
		{999.5, "gbp", "£999.50"},
		{14000000, "jpy", "¥14,000,000"},
		{340000, "aed", "AED 340,000"},
		{50000, "xyz", "XYZ 50,000"},
	}
	for _, c := range cases {
		if got := formatPrice(c.p, c.cur); got != c.want {
			t.Errorf("formatPrice(%v, %s) = %q, want %q", c.p, c.cur, got, c.want)
		}
	}
}

func TestMenuBarTitle(t *testing.T) {
	q := Quote{Price: 91000, Currency: "usd", Change24h: -1.234, HasChange: true}
	if got := menuBarTitle(q, true); got != "₿ $91,000 ▼ 1.23%" {
		t.Errorf("got %q", got)
	}
	if got := menuBarTitle(q, false); got != "₿ $91,000" {
		t.Errorf("got %q", got)
	}
	q.HasChange = false
	if got := menuBarTitle(q, true); got != "₿ $91,000" {
		t.Errorf("no change known: got %q", got)
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.10.0", "v1.9.9", true},
		{"v2.0.0", "1.99.99", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "dev", true},
		{"v1.2.0-beta", "v1.1.0", true},
	}
	for _, c := range cases {
		if got := isNewer(c.a, c.b); got != c.want {
			t.Errorf("isNewer(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestParseHeat(t *testing.T) {
	now := time.Now()
	raw := `{"v":1,"generated":` + jsonInt(now.Add(-5*time.Minute).UnixMilli()) + `,"items":[
	  {"id":"ethereum","label":"ETH","price":3000,"chg":1,"unit":"USD"},
	  {"id":"bitcoin","label":"BTC","price":91234.5,"chg":-2.5,"unit":"USD"}]}`
	var p heatPulse
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	q, err := parseHeat(p, now)
	if err != nil {
		t.Fatal(err)
	}
	if q.Price != 91234.5 || !q.HasChange || q.Change24h != -2.5 || q.Source != "heat.sala.company" {
		t.Errorf("unexpected quote %+v", q)
	}

	// stale: heat's pipeline stopped
	if _, err := parseHeat(p, now.Add(time.Hour)); err == nil {
		t.Error("stale pulse.json was accepted")
	}
	// null change
	p.Items[1].Chg = nil
	if q, _ := parseHeat(p, now); q.HasChange {
		t.Error("null chg should mean no change")
	}
	// no bitcoin
	p.Items = p.Items[:1]
	if _, err := parseHeat(p, now); err == nil {
		t.Error("missing bitcoin was accepted")
	}
}

func TestParseCoinGecko(t *testing.T) {
	m := map[string]map[string]float64{"bitcoin": {"eur": 84000, "eur_24h_change": 0.5, "last_updated_at": 1763400000}}
	q, err := parseCoinGecko(m, "eur")
	if err != nil {
		t.Fatal(err)
	}
	if q.Price != 84000 || q.Change24h != 0.5 || q.At.Unix() != 1763400000 || q.Source != "CoinGecko" {
		t.Errorf("unexpected quote %+v", q)
	}
	if _, err := parseCoinGecko(m, "usd"); err == nil {
		t.Error("missing currency was accepted")
	}
}

func TestFetchHeatOverHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"generated":` + jsonInt(time.Now().UnixMilli()) + `,"items":[{"id":"bitcoin","price":90000,"chg":1.5}]}`))
	}))
	defer srv.Close()
	q, err := fetchHeat(context.Background(), srv.URL, time.Now())
	if err != nil || q.Price != 90000 {
		t.Fatalf("got %+v, %v", q, err)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer down.Close()
	if _, err := fetchHeat(context.Background(), down.URL, time.Now()); err == nil {
		t.Error("HTTP 503 was accepted")
	}
}

func jsonInt(n int64) string { b, _ := json.Marshal(n); return string(b) }
