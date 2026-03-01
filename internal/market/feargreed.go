package market

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// FearGreedDay holds a single day's Fear & Greed Index value.
type FearGreedDay struct {
	Timestamp time.Time
	Value     int
	Label     string
}

// Historical Fear & Greed cache.
var (
	fgHistMu      sync.Mutex
	fgHistData    []FearGreedDay
	fgHistFetched time.Time
	fgHistDays    int
	fgHistCacheTTL = 1 * time.Hour
)

// FetchHistoricalFearGreed retrieves historical Fear & Greed data from api.alternative.me.
// Results are cached for 1 hour.
func FetchHistoricalFearGreed(days int) ([]FearGreedDay, error) {
	if days <= 0 {
		days = 30
	}

	fgHistMu.Lock()
	defer fgHistMu.Unlock()

	if time.Since(fgHistFetched) < fgHistCacheTTL && fgHistDays >= days && len(fgHistData) > 0 {
		// Return cached data, trimmed to requested days.
		if len(fgHistData) > days {
			return fgHistData[:days], nil
		}
		return fgHistData, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.alternative.me/fng/?limit=%d", days)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fear & greed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fear & greed API returned %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Value          string `json:"value"`
			ValueClass     string `json:"value_classification"`
			Timestamp      string `json:"timestamp"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode fear & greed: %w", err)
	}

	data := make([]FearGreedDay, 0, len(result.Data))
	for _, d := range result.Data {
		v, _ := strconv.Atoi(d.Value)
		ts, _ := strconv.ParseInt(d.Timestamp, 10, 64)
		data = append(data, FearGreedDay{
			Timestamp: time.Unix(ts, 0),
			Value:     v,
			Label:     d.ValueClass,
		})
	}

	fgHistData = data
	fgHistFetched = time.Now()
	fgHistDays = days

	return data, nil
}
