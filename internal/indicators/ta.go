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

// ── Research features (normalized to [-1, +1]) ─────────────────────────

// TrendFeature computes EMA12/EMA26 ratio normalized to [-1, 1].
// Positive = bullish trend, negative = bearish.
func TrendFeature(ema12, ema26 float64) float64 {
	if ema26 == 0 {
		return 0
	}
	raw := (ema12/ema26 - 1) * 20
	return clip(raw, -1, 1)
}

// MomentumFeature computes risk-adjusted rate-of-change normalized to [-1, 1].
// ROC_20 / (rolling_vol * sqrt(20)).
func MomentumFeature(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}
	n := len(prices)
	current := prices[n-1]
	past := prices[n-1-period]
	if past == 0 {
		return 0
	}
	roc := current/past - 1

	// Rolling volatility of returns over the period.
	returns := make([]float64, 0, period)
	for i := n - period; i < n; i++ {
		if prices[i-1] != 0 {
			returns = append(returns, prices[i]/prices[i-1]-1)
		}
	}
	vol := sampleStdDev(returns)
	if vol == 0 {
		return 0
	}
	raw := roc / (vol * math.Sqrt(float64(period)))
	return clip(raw, -1, 1)
}

// VolRegimeFeature computes short/long volatility ratio normalized to [-1, 1].
// Positive when short vol < long vol (volatility contracting = favorable).
func VolRegimeFeature(prices []float64, shortPeriod, longPeriod int) float64 {
	if len(prices) < longPeriod+1 {
		return 0
	}
	n := len(prices)
	shortReturns := dailyReturns(prices[n-shortPeriod-1:])
	longReturns := dailyReturns(prices[n-longPeriod-1:])
	volShort := sampleStdDev(shortReturns)
	volLong := sampleStdDev(longReturns)
	if volLong == 0 {
		return 0
	}
	raw := 1 - volShort/volLong
	return clip(raw, -1, 1)
}

// DrawdownFeature computes distance to N-period high normalized to [-1, 1].
// Always <= 0 when below the high. Scaled so -10% DD maps to -1.
func DrawdownFeature(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	n := len(prices)
	current := prices[n-1]
	high := prices[n-period]
	for i := n - period; i < n; i++ {
		if prices[i] > high {
			high = prices[i]
		}
	}
	if high == 0 {
		return 0
	}
	raw := (current/high - 1) * 10
	return clip(raw, -1, 1)
}

// DirVolumeFeature computes volume-weighted price direction normalized to [-1, 1].
// Positive when volume flows into up-moves, negative when into down-moves.
func DirVolumeFeature(prices []float64, volumes []float64, period int) float64 {
	if len(prices) < period+1 || len(volumes) < period+1 {
		return 0
	}
	n := len(prices)
	sumWeighted := 0.0
	sumVol := 0.0
	for i := n - period; i < n; i++ {
		if prices[i-1] == 0 {
			continue
		}
		ret := prices[i]/prices[i-1] - 1
		sign := 0.0
		if ret > 0 {
			sign = 1.0
		} else if ret < 0 {
			sign = -1.0
		}
		sumWeighted += sign * volumes[i]
		sumVol += volumes[i]
	}
	if sumVol == 0 {
		return 0
	}
	raw := sumWeighted / sumVol
	return clip(raw, -1, 1)
}

// clip constrains v to [lo, hi].
func clip(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// sampleStdDev computes the sample standard deviation.
func sampleStdDev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	variance := 0.0
	for _, v := range vals {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(vals) - 1)
	return math.Sqrt(variance)
}

// dailyReturns computes simple returns from a price series.
func dailyReturns(prices []float64) []float64 {
	ret := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			ret = append(ret, prices[i]/prices[i-1]-1)
		}
	}
	return ret
}

// ── Incremental (streaming) indicators for O(1) per-candle updates ───

// StreamEMA maintains EMA state for O(1) updates.
type StreamEMA struct {
	period int
	k      float64
	value  float64
	count  int
	sum    float64 // used during warmup (first `period` values for seed SMA)
}

// NewStreamEMA creates a new streaming EMA.
func NewStreamEMA(period int) *StreamEMA {
	return &StreamEMA{
		period: period,
		k:      2.0 / float64(period+1),
	}
}

// Update feeds a new price and returns the current EMA value.
func (e *StreamEMA) Update(price float64) float64 {
	e.count++
	if e.count <= e.period {
		e.sum += price
		if e.count == e.period {
			e.value = e.sum / float64(e.period)
		}
		return e.value
	}
	e.value = price*e.k + e.value*(1-e.k)
	return e.value
}

// Value returns the current EMA value.
func (e *StreamEMA) Value() float64 { return e.value }

// Ready returns true when enough data has been fed.
func (e *StreamEMA) Ready() bool { return e.count >= e.period }

// StreamRSI maintains RSI state for O(1) updates.
type StreamRSI struct {
	period  int
	avgGain float64
	avgLoss float64
	prev    float64
	count   int
}

// NewStreamRSI creates a new streaming RSI.
func NewStreamRSI(period int) *StreamRSI {
	return &StreamRSI{period: period}
}

// Update feeds a new price and returns the current RSI value.
func (r *StreamRSI) Update(price float64) float64 {
	r.count++
	if r.count == 1 {
		r.prev = price
		return 50
	}

	change := price - r.prev
	r.prev = price

	gain := 0.0
	loss := 0.0
	if change > 0 {
		gain = change
	} else {
		loss = -change
	}

	if r.count <= r.period+1 {
		// Accumulation phase.
		r.avgGain += gain
		r.avgLoss += loss
		if r.count == r.period+1 {
			r.avgGain /= float64(r.period)
			r.avgLoss /= float64(r.period)
		} else {
			return 50
		}
	} else {
		// Wilder's smoothing.
		p := float64(r.period)
		r.avgGain = (r.avgGain*(p-1) + gain) / p
		r.avgLoss = (r.avgLoss*(p-1) + loss) / p
	}

	if r.avgLoss == 0 {
		return 100
	}
	rs := r.avgGain / r.avgLoss
	return 100 - (100 / (1 + rs))
}

// Ready returns true when enough data has been fed.
func (r *StreamRSI) Ready() bool { return r.count > r.period }

// StreamSMA maintains a rolling SMA using a circular buffer for O(1) updates.
type StreamSMA struct {
	period int
	buf    []float64
	idx    int
	sum    float64
	count  int
}

// NewStreamSMA creates a new streaming SMA.
func NewStreamSMA(period int) *StreamSMA {
	return &StreamSMA{
		period: period,
		buf:    make([]float64, period),
	}
}

// Update feeds a new price and returns the current SMA value.
func (s *StreamSMA) Update(price float64) float64 {
	if s.count >= s.period {
		s.sum -= s.buf[s.idx]
	}
	s.buf[s.idx] = price
	s.sum += price
	s.idx = (s.idx + 1) % s.period
	s.count++

	if s.count < s.period {
		return s.sum / float64(s.count)
	}
	return s.sum / float64(s.period)
}

// Value returns the current SMA value.
func (s *StreamSMA) Value() float64 {
	if s.count == 0 {
		return 0
	}
	if s.count < s.period {
		return s.sum / float64(s.count)
	}
	return s.sum / float64(s.period)
}

// Ready returns true when enough data has been fed.
func (s *StreamSMA) Ready() bool { return s.count >= s.period }

// StreamMACD maintains MACD state using streaming EMAs for O(1) updates.
type StreamMACD struct {
	fast   *StreamEMA
	slow   *StreamEMA
	signal *StreamEMA
}

// NewStreamMACD creates a new streaming MACD with given fast/slow/signal periods.
func NewStreamMACD(fastPeriod, slowPeriod, signalPeriod int) *StreamMACD {
	return &StreamMACD{
		fast:   NewStreamEMA(fastPeriod),
		slow:   NewStreamEMA(slowPeriod),
		signal: NewStreamEMA(signalPeriod),
	}
}

// Update feeds a new price and returns (macd, signal, histogram).
func (m *StreamMACD) Update(price float64) (macd, signal, histogram float64) {
	f := m.fast.Update(price)
	s := m.slow.Update(price)
	if !m.slow.Ready() {
		return 0, 0, 0
	}
	macd = f - s
	signal = m.signal.Update(macd)
	if !m.signal.Ready() {
		return macd, 0, macd
	}
	histogram = macd - signal
	return macd, signal, histogram
}

// Ready returns true when the MACD signal line is available.
func (m *StreamMACD) Ready() bool { return m.slow.Ready() && m.signal.Ready() }

// StreamBollinger maintains Bollinger Bands state using a rolling window.
type StreamBollinger struct {
	sma *StreamSMA
	buf []float64
	idx int
	period int
	count int
}

// NewStreamBollinger creates a new streaming Bollinger Bands calculator.
func NewStreamBollinger(period int) *StreamBollinger {
	return &StreamBollinger{
		sma:    NewStreamSMA(period),
		buf:    make([]float64, period),
		period: period,
	}
}

// Update feeds a new price and returns (upper, middle, lower).
func (b *StreamBollinger) Update(price float64) (upper, middle, lower float64) {
	middle = b.sma.Update(price)
	b.buf[b.idx] = price
	b.idx = (b.idx + 1) % b.period
	b.count++

	if b.count < b.period {
		return 0, 0, 0
	}

	// Compute stddev over the buffer.
	sum := 0.0
	for i := 0; i < b.period; i++ {
		diff := b.buf[i] - middle
		sum += diff * diff
	}
	stddev := math.Sqrt(sum / float64(b.period))
	upper = middle + 2*stddev
	lower = middle - 2*stddev
	return upper, middle, lower
}

// Ready returns true when enough data has been fed.
func (b *StreamBollinger) Ready() bool { return b.count >= b.period }

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
