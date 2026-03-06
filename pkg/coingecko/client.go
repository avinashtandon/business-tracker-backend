// Package coingecko is a zero-auth CoinGecko price proxy with
// an in-memory TTL cache. No API key required (free public tier).
package coingecko

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	apiURL         = "https://api.coingecko.com/api/v3/simple/price"
	cacheTTL       = 60 * time.Second
	requestTimeout = 8 * time.Second
)

// symbolToID maps common uppercase ticker symbols to CoinGecko coin IDs.
// Extend this map as needed — no API key required.
var symbolToID = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"SOL":   "solana",
	"BNB":   "binancecoin",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOGE":  "dogecoin",
	"MATIC": "matic-network",
	"POL":   "matic-network",
	"DOT":   "polkadot",
	"LTC":   "litecoin",
	"AVAX":  "avalanche-2",
	"LINK":  "chainlink",
	"UNI":   "uniswap",
	"ATOM":  "cosmos",
	"FIL":   "filecoin",
	"TRX":   "tron",
	"NEAR":  "near",
	"ALGO":  "algorand",
	"HBAR":  "hedera-hashgraph",
	"TON":   "the-open-network",
	"SUI":   "sui",
	"APT":   "aptos",
	"OP":    "optimism",
	"ARB":   "arbitrum",
	"INJ":   "injective-protocol",
	"SHIB":  "shiba-inu",
	"PEPE":  "pepe",
	"WIF":   "dogwifcoin",
	"JUP":   "jupiter-exchange-solana",
	"BONK":  "bonk",
}

// PriceResult is the data returned to callers.
type PriceResult struct {
	Prices   map[string]float64 `json:"prices"`    // symbol → INR price
	CachedAt time.Time          `json:"cached_at"` // when prices were fetched
	Stale    bool               `json:"stale"`     // true if serving old cached data
}

// cacheEntry holds the last successful response.
type cacheEntry struct {
	prices   map[string]float64
	cachedAt time.Time
}

// Cache is the global singleton — allocate once, reuse everywhere.
var Cache = &priceCache{}

type priceCache struct {
	mu    sync.RWMutex
	entry *cacheEntry
}

// GetPrices returns INR prices for the requested symbols.
// If CoinGecko is unreachable it serves the last cached values with stale=true.
// Returns an error only when CoinGecko fails AND there is no cached data at all.
func (c *priceCache) GetPrices(symbols []string) (PriceResult, error) {
	// Normalise and resolve to CoinGecko IDs
	upper := make([]string, 0, len(symbols))
	idToSymbol := make(map[string]string) // geckoID → original symbol
	for _, s := range symbols {
		s = strings.TrimSpace(strings.ToUpper(s))
		if s == "" {
			continue
		}
		upper = append(upper, s)
		if id, ok := symbolToID[s]; ok {
			idToSymbol[id] = s
		}
	}

	// Check cache freshness
	c.mu.RLock()
	cached := c.entry
	c.mu.RUnlock()

	if cached != nil && time.Since(cached.cachedAt) < cacheTTL {
		return c.buildResult(upper, cached, false), nil
	}

	// Fetch from CoinGecko
	geckoIDs := make([]string, 0, len(idToSymbol))
	for id := range idToSymbol {
		geckoIDs = append(geckoIDs, id)
	}

	fetched, err := fetchFromCoinGecko(geckoIDs, idToSymbol)
	if err != nil {
		// CoinGecko unreachable — serve stale data if we have it
		if cached != nil {
			return c.buildResult(upper, cached, true), nil
		}
		return PriceResult{}, fmt.Errorf("PRICE_FETCH_FAILED: %w", err)
	}

	// Merge new prices into the full cache (keep symbols we didn't request)
	c.mu.Lock()
	if c.entry == nil {
		c.entry = &cacheEntry{prices: make(map[string]float64)}
	}
	for sym, price := range fetched {
		c.entry.prices[sym] = price
	}
	c.entry.cachedAt = time.Now()
	newEntry := c.entry
	c.mu.Unlock()

	return c.buildResult(upper, newEntry, false), nil
}

// buildResult filters cached data to only the requested symbols.
func (c *priceCache) buildResult(symbols []string, entry *cacheEntry, stale bool) PriceResult {
	prices := make(map[string]float64, len(symbols))
	for _, sym := range symbols {
		if p, ok := entry.prices[sym]; ok {
			prices[sym] = p
		}
		// unknown symbols are absent from the map — frontend handles gracefully
	}
	return PriceResult{
		Prices:   prices,
		CachedAt: entry.cachedAt,
		Stale:    stale,
	}
}

// fetchFromCoinGecko performs a single HTTP call and returns symbol → INR prices.
func fetchFromCoinGecko(geckoIDs []string, idToSymbol map[string]string) (map[string]float64, error) {
	if len(geckoIDs) == 0 {
		return map[string]float64{}, nil
	}

	client := &http.Client{Timeout: requestTimeout}
	url := fmt.Sprintf("%s?ids=%s&vs_currencies=inr", apiURL, strings.Join(geckoIDs, ","))

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko returned HTTP %d", resp.StatusCode)
	}

	// Response: {"bitcoin":{"inr":6800000}, "ethereum":{"inr":210000}}
	var raw map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	result := make(map[string]float64, len(raw))
	for geckoID, currencies := range raw {
		if sym, ok := idToSymbol[geckoID]; ok {
			if inr, ok := currencies["inr"]; ok {
				result[sym] = inr
			}
		}
	}
	return result, nil
}
