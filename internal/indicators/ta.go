package indicators

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// Analysis holds full technical analysis for a symbol.
type Analysis struct {
	Symbol         string  `json:"symbol"`
	Price          float64 `json:"price"`
	RSI            float64 `json:"rsi"`
	RSISignal      string  `json:"rsi_signal"`
	MACD           float64 `json:"macd"`
	MACDSignal     float64 `json:"macd_signal"`
	MACDHistogram  float64 `json:"macd_histogram"`
	MACDTrend      string  `json:"macd_trend"`
	BollingerUpper float64 `json:"bollinger_upper"`
	BollingerLower float64 `json:"bollinger_lower"`
	BollingerPos   string  `json:"bollinger_position"`
	SMA20          float64 `json:"sma_20"`
	SMA50          float64 `json:"sma_50"`
	Trend          string  `json:"trend"`
	FearGreed      int     `json:"fear_greed_index"`
	FearGreedLabel string  `json:"fear_greed_label"`
	Summary        string  `json:"summary"`
}

// RSI calculates the Relative Strength Index.
func RSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // neutral default
	}

	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// Smooth using Wilder's method for remaining data.
	for i := period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) - change) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// SMA calculates a Simple Moving Average.
func SMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// EMA calculates an Exponential Moving Average.
func EMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	k := 2.0 / float64(period+1)
	ema := SMA(prices[:period], period)
	for i := period; i < len(prices); i++ {
		ema = prices[i]*k + ema*(1-k)
	}
	return ema
}

// MACD calculates MACD line, signal, and histogram using 12/26/9.
func MACDCalc(prices []float64) (macd, signal, histogram float64) {
	if len(prices) < 26 {
		return 0, 0, 0
	}
	ema12 := EMA(prices, 12)
	ema26 := EMA(prices, 26)
	macd = ema12 - ema26

	// Build MACD series for signal line.
	if len(prices) >= 35 { // need enough for 9-period EMA of MACD
		macdSeries := make([]float64, 0, len(prices)-25)
		for i := 26; i <= len(prices); i++ {
			e12 := EMA(prices[:i], 12)
			e26 := EMA(prices[:i], 26)
			macdSeries = append(macdSeries, e12-e26)
		}
		if len(macdSeries) >= 9 {
			signal = EMA(macdSeries, 9)
		}
	}

	histogram = macd - signal
	return macd, signal, histogram
}

// BollingerBands calculates upper, middle, and lower bands.
func BollingerBands(prices []float64, period int) (upper, middle, lower float64) {
	if len(prices) < period {
		return 0, 0, 0
	}
	middle = SMA(prices, period)

	// Standard deviation.
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - middle
		sum += diff * diff
	}
	stddev := math.Sqrt(sum / float64(period))

	upper = middle + 2*stddev
	lower = middle - 2*stddev
	return upper, middle, lower
}

// TrendDirection determines trend from recent price action.
func TrendDirection(prices []float64) string {
	if len(prices) < 20 {
		return "neutral"
	}
	sma20 := SMA(prices, 20)
	current := prices[len(prices)-1]

	pctDiff := (current - sma20) / sma20 * 100
	switch {
	case pctDiff > 1:
		return "bullish"
	case pctDiff < -1:
		return "bearish"
	default:
		return "neutral"
	}
}

// Analyze runs full technical analysis on a symbol.
func Analyze(symbol string, currentPrice float64, priceHistory []float64, fearGreed int, fgLabel string) *Analysis {
	a := &Analysis{
		Symbol:         symbol,
		Price:          currentPrice,
		FearGreed:      fearGreed,
		FearGreedLabel: fgLabel,
	}

	if len(priceHistory) < 2 {
		a.Summary = "Insufficient price data for analysis."
		return a
	}

	// RSI.
	a.RSI = RSI(priceHistory, 14)
	switch {
	case a.RSI >= 70:
		a.RSISignal = "overbought"
	case a.RSI <= 30:
		a.RSISignal = "oversold"
	default:
		a.RSISignal = "neutral"
	}

	// MACD.
	a.MACD, a.MACDSignal, a.MACDHistogram = MACDCalc(priceHistory)
	if a.MACDHistogram > 0 {
		a.MACDTrend = "bullish"
	} else {
		a.MACDTrend = "bearish"
	}

	// Bollinger Bands.
	a.BollingerUpper, _, a.BollingerLower = BollingerBands(priceHistory, 20)
	switch {
	case currentPrice >= a.BollingerUpper:
		a.BollingerPos = "above"
	case currentPrice <= a.BollingerLower:
		a.BollingerPos = "below"
	default:
		a.BollingerPos = "middle"
	}

	// Moving averages.
	a.SMA20 = SMA(priceHistory, 20)
	a.SMA50 = SMA(priceHistory, 50)

	// Trend.
	a.Trend = TrendDirection(priceHistory)

	// Summary.
	bullish, bearish := 0, 0
	if a.RSISignal == "oversold" {
		bullish++
	} else if a.RSISignal == "overbought" {
		bearish++
	}
	if a.MACDTrend == "bullish" {
		bullish++
	} else {
		bearish++
	}
	if a.Trend == "bullish" {
		bullish++
	} else if a.Trend == "bearish" {
		bearish++
	}
	if a.BollingerPos == "below" {
		bullish++
	} else if a.BollingerPos == "above" {
		bearish++
	}

	switch {
	case bullish >= 3:
		a.Summary = fmt.Sprintf("%s shows strong bullish signals (%d/4 indicators positive). RSI %.1f (%s), MACD %s, trend %s.", symbol, bullish, a.RSI, a.RSISignal, a.MACDTrend, a.Trend)
	case bearish >= 3:
		a.Summary = fmt.Sprintf("%s shows strong bearish signals (%d/4 indicators negative). RSI %.1f (%s), MACD %s, trend %s.", symbol, bearish, a.RSI, a.RSISignal, a.MACDTrend, a.Trend)
	default:
		a.Summary = fmt.Sprintf("%s shows mixed signals. RSI %.1f (%s), MACD %s, trend %s. Consider waiting for clearer direction.", symbol, a.RSI, a.RSISignal, a.MACDTrend, a.Trend)
	}

	if fgLabel != "" {
		a.Summary += fmt.Sprintf(" Market sentiment: %s (%d/100).", fgLabel, fearGreed)
	}

	return a
}

// Fear & Greed cache.
var (
	fgMu       sync.Mutex
	fgValue    int
	fgLabel    string
	fgFetched  time.Time
	fgCacheTTL = 5 * time.Minute
)

// FetchFearGreed retrieves the crypto Fear & Greed Index from api.alternative.me.
// Results are cached for 5 minutes.
func FetchFearGreed() (value int, label string, err error) {
	fgMu.Lock()
	defer fgMu.Unlock()

	if time.Since(fgFetched) < fgCacheTTL {
		return fgValue, fgLabel, nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.alternative.me/fng/")
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Value           string `json:"value"`
			ValueClass      string `json:"value_classification"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", err
	}
	if len(result.Data) == 0 {
		return 50, "Neutral", nil
	}

	v := 50
	fmt.Sscanf(result.Data[0].Value, "%d", &v)
	l := result.Data[0].ValueClass

	fgValue = v
	fgLabel = l
	fgFetched = time.Now()

	return v, l, nil
}
