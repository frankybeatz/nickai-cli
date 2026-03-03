package indicators

import (
	"math"
	"testing"
)

func TestRSI(t *testing.T) {
	// Generate a simple uptrend: 100, 101, 102, ..., 115 (16 data points, period 14).
	prices := make([]float64, 16)
	for i := range prices {
		prices[i] = 100 + float64(i)
	}

	rsi := RSI(prices, 14)
	// All gains, no losses => RSI should be 100.
	if rsi != 100 {
		t.Errorf("RSI of uptrend = %.2f, want 100", rsi)
	}
}

func TestRSIDowntrend(t *testing.T) {
	// All losses.
	prices := make([]float64, 16)
	for i := range prices {
		prices[i] = 115 - float64(i)
	}

	rsi := RSI(prices, 14)
	// All losses => RSI should be 0.
	if math.Abs(rsi) > 0.01 {
		t.Errorf("RSI of downtrend = %.2f, want 0", rsi)
	}
}

func TestRSIInsufficientData(t *testing.T) {
	rsi := RSI([]float64{100, 101}, 14)
	if rsi != 50 {
		t.Errorf("RSI with insufficient data = %.2f, want 50 (neutral)", rsi)
	}
}

func TestSMA(t *testing.T) {
	prices := []float64{10, 20, 30, 40, 50}
	sma := SMA(prices, 3)
	// Average of last 3: (30+40+50)/3 = 40.
	expected := 40.0
	if math.Abs(sma-expected) > 0.01 {
		t.Errorf("SMA(3) = %.2f, want %.2f", sma, expected)
	}
}

func TestSMAInsufficientData(t *testing.T) {
	sma := SMA([]float64{10, 20}, 5)
	if sma != 0 {
		t.Errorf("SMA with insufficient data = %.2f, want 0", sma)
	}
}

func TestEMA(t *testing.T) {
	prices := []float64{22, 24, 23, 25, 26, 28, 27, 29, 30, 28}
	ema := EMA(prices, 5)
	// EMA should be > 0 and in the range of prices.
	if ema <= 20 || ema >= 35 {
		t.Errorf("EMA(5) = %.2f, out of expected range", ema)
	}
}

func TestEMAInsufficientData(t *testing.T) {
	ema := EMA([]float64{10, 20}, 5)
	if ema != 0 {
		t.Errorf("EMA with insufficient data = %.2f, want 0", ema)
	}
}

func TestMACDCalc(t *testing.T) {
	// Need at least 26 data points.
	prices := make([]float64, 40)
	for i := range prices {
		prices[i] = 100 + float64(i)*0.5 // gentle uptrend
	}

	macd, signal, histogram := MACDCalc(prices)

	// In an uptrend, MACD should be positive (EMA12 > EMA26).
	if macd <= 0 {
		t.Errorf("MACD in uptrend = %.4f, expected positive", macd)
	}

	// Histogram = MACD - Signal.
	expectedHist := macd - signal
	if math.Abs(histogram-expectedHist) > 0.001 {
		t.Errorf("Histogram = %.4f, want %.4f (macd-signal)", histogram, expectedHist)
	}
}

func TestMACDCalcInsufficientData(t *testing.T) {
	macd, signal, histogram := MACDCalc([]float64{100, 101, 102})
	if macd != 0 || signal != 0 || histogram != 0 {
		t.Error("MACDCalc with insufficient data should return zeros")
	}
}

func TestBollingerBands(t *testing.T) {
	// Constant prices => stddev = 0 => bands collapse to middle.
	constant := make([]float64, 20)
	for i := range constant {
		constant[i] = 100
	}
	upper, middle, lower := BollingerBands(constant, 20)
	if middle != 100 {
		t.Errorf("middle = %.2f, want 100", middle)
	}
	if upper != 100 {
		t.Errorf("upper = %.2f, want 100 (zero stddev)", upper)
	}
	if lower != 100 {
		t.Errorf("lower = %.2f, want 100 (zero stddev)", lower)
	}

	// Varying prices => bands should be wider.
	varying := []float64{90, 95, 100, 105, 110, 90, 95, 100, 105, 110,
		90, 95, 100, 105, 110, 90, 95, 100, 105, 110}
	upper, middle, lower = BollingerBands(varying, 20)
	if upper <= middle {
		t.Error("upper band should be above middle")
	}
	if lower >= middle {
		t.Error("lower band should be below middle")
	}
	// Bands should be symmetric around middle.
	if math.Abs((upper-middle)-(middle-lower)) > 0.01 {
		t.Error("bands should be symmetric around middle")
	}
}

func TestBollingerBandsInsufficientData(t *testing.T) {
	upper, middle, lower := BollingerBands([]float64{100}, 20)
	if upper != 0 || middle != 0 || lower != 0 {
		t.Error("BollingerBands with insufficient data should return zeros")
	}
}

// ── Research feature tests ──

func TestTrendFeature(t *testing.T) {
	tests := []struct {
		name     string
		ema12    float64
		ema26    float64
		expected float64
	}{
		{"bullish", 102, 100, 0.4},         // (1.02-1)*20
		{"bearish", 98, 100, -0.4},          // (0.98-1)*20
		{"neutral", 100, 100, 0},
		{"strong bull clipped", 110, 100, 1.0}, // 2.0 → clipped
		{"strong bear clipped", 90, 100, -1.0}, // -2.0 → clipped
		{"zero ema26", 100, 0, 0},
	}
	for _, tt := range tests {
		got := TrendFeature(tt.ema12, tt.ema26)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("%s: TrendFeature(%f, %f) = %f, want %f", tt.name, tt.ema12, tt.ema26, got, tt.expected)
		}
	}
}

func TestMomentumFeature(t *testing.T) {
	// Uptrend → positive momentum.
	prices := make([]float64, 25)
	for i := range prices {
		prices[i] = 100 + float64(i)*0.8
	}
	got := MomentumFeature(prices, 20)
	if got <= 0 {
		t.Errorf("uptrend: MomentumFeature = %f, want > 0", got)
	}
	if got > 1 {
		t.Errorf("uptrend: MomentumFeature = %f, want <= 1", got)
	}

	// Downtrend → negative.
	down := make([]float64, 25)
	for i := range down {
		down[i] = 120 - float64(i)*0.8
	}
	got2 := MomentumFeature(down, 20)
	if got2 >= 0 {
		t.Errorf("downtrend: MomentumFeature = %f, want < 0", got2)
	}

	// Insufficient data → 0.
	if got3 := MomentumFeature([]float64{100, 101}, 20); got3 != 0 {
		t.Errorf("short data: MomentumFeature = %f, want 0", got3)
	}
}

func TestVolRegimeFeature(t *testing.T) {
	// Build data where early portion is volatile and recent portion is calm.
	prices := make([]float64, 70)
	for i := range prices {
		if i < 30 {
			prices[i] = 100 + float64(i%2)*10 - 5
		} else {
			prices[i] = 100 + float64(i%2)*0.5
		}
	}
	got := VolRegimeFeature(prices, 10, 60)
	if got <= 0 {
		t.Errorf("calm regime: VolRegimeFeature = %f, want > 0", got)
	}

	// Insufficient data.
	if got2 := VolRegimeFeature(prices[:10], 10, 60); got2 != 0 {
		t.Errorf("short data: VolRegimeFeature = %f, want 0", got2)
	}
}

func TestDrawdownFeature(t *testing.T) {
	// At the high → 0.
	atHigh := make([]float64, 20)
	for i := range atHigh {
		atHigh[i] = 100
	}
	got := DrawdownFeature(atHigh, 20)
	if math.Abs(got) > 0.001 {
		t.Errorf("at high: DrawdownFeature = %f, want ~0", got)
	}

	// 5% below 20-day high → (0.95-1)*10 = -0.5.
	below := make([]float64, 20)
	for i := range below {
		below[i] = 100
	}
	below[19] = 95
	got2 := DrawdownFeature(below, 20)
	if math.Abs(got2-(-0.5)) > 0.001 {
		t.Errorf("5%% drawdown: DrawdownFeature = %f, want -0.5", got2)
	}

	// Insufficient data.
	if got3 := DrawdownFeature([]float64{100, 95}, 20); got3 != 0 {
		t.Errorf("short data: DrawdownFeature = %f, want 0", got3)
	}
}

func TestDirVolumeFeature(t *testing.T) {
	// All up-moves → +1.
	prices := make([]float64, 21)
	volumes := make([]float64, 21)
	for i := range prices {
		prices[i] = 100 + float64(i)
		volumes[i] = 1000
	}
	got := DirVolumeFeature(prices, volumes, 20)
	if got != 1.0 {
		t.Errorf("all up: DirVolumeFeature = %f, want 1.0", got)
	}

	// All down-moves → -1.
	down := make([]float64, 21)
	for i := range down {
		down[i] = 120 - float64(i)
		volumes[i] = 1000
	}
	got2 := DirVolumeFeature(down, volumes, 20)
	if got2 != -1.0 {
		t.Errorf("all down: DirVolumeFeature = %f, want -1.0", got2)
	}

	// Insufficient data.
	if got3 := DirVolumeFeature(prices[:5], volumes[:5], 20); got3 != 0 {
		t.Errorf("short data: DirVolumeFeature = %f, want 0", got3)
	}
}

func TestClip(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float64
	}{
		{0.5, -1, 1, 0.5},
		{1.5, -1, 1, 1.0},
		{-1.5, -1, 1, -1.0},
		{0, 0, 1, 0},
	}
	for _, tt := range tests {
		got := clip(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clip(%f, %f, %f) = %f, want %f", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}

func TestSampleStdDev(t *testing.T) {
	// [2, 4, 4, 4, 5, 5, 7, 9] → sample stddev ≈ 2.138.
	vals := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	got := sampleStdDev(vals)
	if math.Abs(got-2.138) > 0.01 {
		t.Errorf("sampleStdDev = %f, want ~2.138", got)
	}
	if got2 := sampleStdDev([]float64{5}); got2 != 0 {
		t.Errorf("single value: sampleStdDev = %f, want 0", got2)
	}
	if got3 := sampleStdDev(nil); got3 != 0 {
		t.Errorf("nil: sampleStdDev = %f, want 0", got3)
	}
}

func TestDailyReturns(t *testing.T) {
	prices := []float64{100, 110, 105}
	ret := dailyReturns(prices)
	if len(ret) != 2 {
		t.Fatalf("len = %d, want 2", len(ret))
	}
	if math.Abs(ret[0]-0.10) > 0.001 {
		t.Errorf("ret[0] = %f, want 0.10", ret[0])
	}
	expected := (105.0 - 110.0) / 110.0
	if math.Abs(ret[1]-expected) > 0.001 {
		t.Errorf("ret[1] = %f, want %f", ret[1], expected)
	}
}

// ── Streaming indicator tests ──

func TestStreamEMA(t *testing.T) {
	prices := []float64{22.27, 22.19, 22.08, 22.17, 22.18, 22.13, 22.23, 22.43, 22.24, 22.29,
		22.15, 22.39, 22.38, 22.61, 23.36, 24.05, 23.75, 23.83, 23.95, 23.63}

	stream := NewStreamEMA(10)
	var streamVal float64
	for _, p := range prices {
		streamVal = stream.Update(p)
	}
	batchVal := EMA(prices, 10)
	if math.Abs(streamVal-batchVal) > 0.01 {
		t.Errorf("StreamEMA = %f, batch EMA = %f, diff too large", streamVal, batchVal)
	}
}

func TestStreamRSI(t *testing.T) {
	// Generate a price series with known behavior.
	prices := make([]float64, 50)
	prices[0] = 100
	for i := 1; i < 50; i++ {
		if i%3 == 0 {
			prices[i] = prices[i-1] - 1.5
		} else {
			prices[i] = prices[i-1] + 1.0
		}
	}

	stream := NewStreamRSI(14)
	var streamVal float64
	for _, p := range prices {
		streamVal = stream.Update(p)
	}
	batchVal := RSI(prices, 14)
	if math.Abs(streamVal-batchVal) > 0.5 {
		t.Errorf("StreamRSI = %f, batch RSI = %f, diff too large", streamVal, batchVal)
	}
}

func TestStreamSMA(t *testing.T) {
	prices := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	stream := NewStreamSMA(5)
	var streamVal float64
	for _, p := range prices {
		streamVal = stream.Update(p)
	}
	batchVal := SMA(prices, 5)
	if math.Abs(streamVal-batchVal) > 0.001 {
		t.Errorf("StreamSMA = %f, batch SMA = %f", streamVal, batchVal)
	}
}

func TestStreamMACD(t *testing.T) {
	prices := make([]float64, 60)
	for i := range prices {
		prices[i] = 100 + float64(i)*0.5 + math.Sin(float64(i)/5)*3
	}

	stream := NewStreamMACD(12, 26, 9)
	var sMACD, sSignal, sHist float64
	for _, p := range prices {
		sMACD, sSignal, sHist = stream.Update(p)
	}
	bMACD, bSignal, bHist := MACDCalc(prices)

	if math.Abs(sMACD-bMACD) > 0.1 {
		t.Errorf("StreamMACD macd = %f, batch = %f", sMACD, bMACD)
	}
	if math.Abs(sSignal-bSignal) > 0.1 {
		t.Errorf("StreamMACD signal = %f, batch = %f", sSignal, bSignal)
	}
	if math.Abs(sHist-bHist) > 0.1 {
		t.Errorf("StreamMACD histogram = %f, batch = %f", sHist, bHist)
	}
}

func TestStreamBollinger(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 100 + float64(i%5)*2 - 4
	}

	stream := NewStreamBollinger(20)
	var sU, sM, sL float64
	for _, p := range prices {
		sU, sM, sL = stream.Update(p)
	}
	bU, bM, bL := BollingerBands(prices, 20)

	if math.Abs(sU-bU) > 0.01 {
		t.Errorf("StreamBollinger upper = %f, batch = %f", sU, bU)
	}
	if math.Abs(sM-bM) > 0.01 {
		t.Errorf("StreamBollinger middle = %f, batch = %f", sM, bM)
	}
	if math.Abs(sL-bL) > 0.01 {
		t.Errorf("StreamBollinger lower = %f, batch = %f", sL, bL)
	}
}
