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
