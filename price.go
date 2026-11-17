package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Quote is one Bitcoin price reading.
type Quote struct {
	Price     float64   // in Currency
	Change24h float64   // percent; NaN-free, 0 when unknown (see HasChange)
	HasChange bool      // false when the source gave no 24h change
	Currency  string    // lower case, e.g. "usd"
	At        time.Time // when the source made the number
	Source    string    // "heat.sala.company" or "CoinGecko"
}

const (
	heatURL      = "https://heat.sala.company/pulse.json"
	coingeckoURL = "https://api.coingecko.com/api/v3/simple/price"
	// heat rebuilds pulse.json every 10 minutes; older than this means its pipeline is stuck.
	heatMaxAge = 30 * time.Minute
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// fetchQuote reads the stored price from heat.sala.company (USD only), and falls back to
// CoinGecko when heat is unreachable, stale or the currency is not USD.
func fetchQuote(ctx context.Context, currency string) (Quote, error) {
	currency = strings.ToLower(currency)
	var heatErr error
	if currency == "usd" {
		q, err := fetchHeat(ctx, heatURL, time.Now())
		if err == nil {
			return q, nil
		}
		heatErr = err
	}
	q, err := fetchCoinGecko(ctx, coingeckoURL, currency)
	if err != nil && heatErr != nil {
		return Quote{}, fmt.Errorf("heat: %v; coingecko: %v", heatErr, err)
	}
	return q, err
}

func getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "btconmenubar/"+version)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.New("rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// pulse.json, as written by heat's scripts/pulse.mjs.
type heatPulse struct {
	Generated int64 `json:"generated"` // ms since epoch
	Items     []struct {
		ID    string   `json:"id"`
		Price float64  `json:"price"`
		Chg   *float64 `json:"chg"`
		Unit  string   `json:"unit"`
	} `json:"items"`
}

func fetchHeat(ctx context.Context, url string, now time.Time) (Quote, error) {
	var p heatPulse
	if err := getJSON(ctx, url, &p); err != nil {
		return Quote{}, err
	}
	return parseHeat(p, now)
}

func parseHeat(p heatPulse, now time.Time) (Quote, error) {
	at := time.UnixMilli(p.Generated)
	if p.Generated <= 0 || now.Sub(at) > heatMaxAge {
		return Quote{}, fmt.Errorf("stale data (generated %s)", at.Format(time.RFC3339))
	}
	for _, it := range p.Items {
		if it.ID != "bitcoin" || !(it.Price > 0) {
			continue
		}
		q := Quote{Price: it.Price, Currency: "usd", At: at, Source: "heat.sala.company"}
		if it.Chg != nil {
			q.Change24h, q.HasChange = *it.Chg, true
		}
		return q, nil
	}
	return Quote{}, errors.New("no bitcoin price in pulse.json")
}

func fetchCoinGecko(ctx context.Context, base, currency string) (Quote, error) {
	url := base + "?ids=bitcoin&include_24hr_change=true&include_last_updated_at=true&vs_currencies=" + currency
	var m map[string]map[string]float64
	if err := getJSON(ctx, url, &m); err != nil {
		return Quote{}, err
	}
	return parseCoinGecko(m, currency)
}

func parseCoinGecko(m map[string]map[string]float64, currency string) (Quote, error) {
	b := m["bitcoin"]
	price := b[currency]
	if !(price > 0) {
		return Quote{}, fmt.Errorf("no bitcoin price in %s", strings.ToUpper(currency))
	}
	q := Quote{Price: price, Currency: currency, At: time.Now(), Source: "CoinGecko"}
	if ts, ok := b["last_updated_at"]; ok && ts > 0 {
		q.At = time.Unix(int64(ts), 0)
	}
	if c, ok := b[currency+"_24h_change"]; ok {
		q.Change24h, q.HasChange = c, true
	}
	return q, nil
}
