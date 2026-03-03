package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBuildStreamURL(t *testing.T) {
	ws := &BinanceWS{
		symbols: []string{"BTC", "ETH", "SOL"},
		stopCh:  make(chan struct{}),
	}

	u := ws.buildStreamURL()

	if !strings.Contains(u, "stream.binance.com:9443") {
		t.Errorf("URL missing host: %s", u)
	}
	if !strings.Contains(u, "btcusdt@miniTicker") {
		t.Errorf("URL missing btcusdt stream: %s", u)
	}
	if !strings.Contains(u, "ethusdt@miniTicker") {
		t.Errorf("URL missing ethusdt stream: %s", u)
	}
	if !strings.Contains(u, "solusdt@miniTicker") {
		t.Errorf("URL missing solusdt stream: %s", u)
	}
}

func TestBuildStreamURLWithSuffix(t *testing.T) {
	ws := &BinanceWS{
		symbols: []string{"BTCUSDT", "ethusdt"},
		stopCh:  make(chan struct{}),
	}

	u := ws.buildStreamURL()

	// Should not double-append "usdt".
	if strings.Contains(u, "btcusdtusdt") {
		t.Errorf("URL has doubled usdt suffix: %s", u)
	}
}

func TestWebSocketPriceCallback(t *testing.T) {
	// Create a test websocket server that sends mini ticker events.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	var mu sync.Mutex
	received := make(map[string]float64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Send a combined stream message.
		msg := binanceCombinedStream{
			Stream: "btcusdt@miniTicker",
			Data: binanceMiniTicker{
				Event:  "24hrMiniTicker",
				Symbol: "BTCUSDT",
				Close:  "97500.50",
			},
		}
		data, _ := json.Marshal(msg)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}

		// Send a second message for ETH.
		msg2 := binanceCombinedStream{
			Stream: "ethusdt@miniTicker",
			Data: binanceMiniTicker{
				Event:  "24hrMiniTicker",
				Symbol: "ETHUSDT",
				Close:  "3456.78",
			},
		}
		data2, _ := json.Marshal(msg2)
		if err := conn.WriteMessage(websocket.TextMessage, data2); err != nil {
			return
		}

		// Keep connection open briefly.
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	// Convert http URL to ws URL.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Directly test the connection using a custom BinanceWS.
	ws := &BinanceWS{
		symbols: []string{"BTC", "ETH"},
		onPrice: func(symbol string, price float64) {
			mu.Lock()
			received[symbol] = price
			mu.Unlock()
		},
		stopCh: make(chan struct{}),
	}

	// Connect directly to test server.
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	ws.mu.Lock()
	ws.conn = conn
	ws.mu.Unlock()

	// Run readLoop in background.
	done := make(chan struct{})
	go func() {
		ws.readLoop()
		close(done)
	}()

	// Wait for messages to be processed.
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not complete in time")
	}

	mu.Lock()
	defer mu.Unlock()

	if received["BTCUSDT"] != 97500.50 {
		t.Errorf("BTCUSDT price = %v, want 97500.50", received["BTCUSDT"])
	}
	if received["ETHUSDT"] != 3456.78 {
		t.Errorf("ETHUSDT price = %v, want 3456.78", received["ETHUSDT"])
	}
}

func TestStopWebSocket(t *testing.T) {
	ws := &BinanceWS{
		symbols: []string{"BTC"},
		onPrice: func(string, float64) {},
		stopCh:  make(chan struct{}),
	}

	// StopWebSocket should not panic even without a connection.
	ws.StopWebSocket()

	if !ws.stopped {
		t.Error("expected stopped to be true after StopWebSocket")
	}

	// Double stop should be safe.
	ws.StopWebSocket()
}
