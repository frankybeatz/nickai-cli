package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// binanceMiniTicker is the JSON structure for Binance 24hr mini ticker events.
type binanceMiniTicker struct {
	Event  string `json:"e"` // "24hrMiniTicker"
	Symbol string `json:"s"` // "BTCUSDT"
	Close  string `json:"c"` // current close price as string
}

// binanceCombinedStream wraps individual stream events in the combined stream format.
type binanceCombinedStream struct {
	Stream string            `json:"stream"`
	Data   binanceMiniTicker `json:"data"`
}

// PriceCallback is called when a new price update arrives from the websocket.
type PriceCallback func(symbol string, price float64)

// BinanceWS manages a websocket connection to Binance for live price streaming.
type BinanceWS struct {
	symbols  []string
	onPrice  PriceCallback
	conn     *websocket.Conn
	mu       sync.Mutex
	stopCh   chan struct{}
	stopped  bool
}

// StartWebSocket connects to the Binance combined mini ticker stream for the
// given symbols and calls onPrice whenever a price update arrives.
// It handles automatic reconnection with exponential backoff.
// Symbols should be base symbols like "BTC", "ETH", "SOL" (without USDT suffix).
func StartWebSocket(symbols []string, onPrice PriceCallback) (*BinanceWS, error) {
	ws := &BinanceWS{
		symbols: symbols,
		onPrice: onPrice,
		stopCh:  make(chan struct{}),
	}

	// Start connection loop in background goroutine.
	go ws.connectLoop()

	return ws, nil
}

// StopWebSocket cleanly shuts down the websocket connection.
func (ws *BinanceWS) StopWebSocket() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.stopped {
		return
	}
	ws.stopped = true
	close(ws.stopCh)

	if ws.conn != nil {
		// Send close message with a short deadline, then close.
		_ = ws.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = ws.conn.Close()
		ws.conn = nil
	}
}

// buildStreamURL constructs the Binance combined stream URL for the given symbols.
func (ws *BinanceWS) buildStreamURL() string {
	var streams []string
	for _, sym := range ws.symbols {
		normalized := strings.ToLower(strings.TrimSpace(sym))
		// Ensure USDT suffix for Binance stream names.
		if !strings.HasSuffix(normalized, "usdt") {
			normalized += "usdt"
		}
		streams = append(streams, normalized+"@miniTicker")
	}
	u := url.URL{
		Scheme:   "wss",
		Host:     "stream.binance.com:9443",
		Path:     "/stream",
		RawQuery: "streams=" + strings.Join(streams, "/"),
	}
	return u.String()
}

// connectLoop manages the websocket connection lifecycle with reconnection.
func (ws *BinanceWS) connectLoop() {
	attempt := 0
	maxBackoff := 60 * time.Second

	for {
		// Check if stopped before attempting connection.
		select {
		case <-ws.stopCh:
			return
		default:
		}

		err := ws.connect()
		if err == nil {
			// Successfully connected — read messages.
			attempt = 0
			ws.readLoop()
		}

		// Check if stopped after disconnect.
		select {
		case <-ws.stopCh:
			return
		default:
		}

		// Exponential backoff: 1s, 2s, 4s, 8s, ... up to maxBackoff.
		attempt++
		backoff := time.Duration(math.Min(
			float64(time.Second)*math.Pow(2, float64(attempt-1)),
			float64(maxBackoff),
		))

		select {
		case <-ws.stopCh:
			return
		case <-time.After(backoff):
			// Continue to reconnect.
		}
	}
}

// connect establishes a new websocket connection.
func (ws *BinanceWS) connect() error {
	streamURL := ws.buildStreamURL()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(streamURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	// Set read deadline high — Binance sends pings, and we respond via
	// the default pong handler. If no message for 5 minutes, reconnect.
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})

	ws.mu.Lock()
	ws.conn = conn
	ws.mu.Unlock()

	return nil
}

// readLoop reads messages from the websocket and dispatches price updates.
func (ws *BinanceWS) readLoop() {
	for {
		select {
		case <-ws.stopCh:
			return
		default:
		}

		ws.mu.Lock()
		conn := ws.conn
		ws.mu.Unlock()

		if conn == nil {
			return
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			// Connection lost — return to trigger reconnect.
			ws.mu.Lock()
			if ws.conn == conn {
				_ = ws.conn.Close()
				ws.conn = nil
			}
			ws.mu.Unlock()
			return
		}

		// Reset read deadline on any message.
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// Parse combined stream format: {"stream":"btcusdt@miniTicker","data":{...}}
		var combined binanceCombinedStream
		if err := json.Unmarshal(msgBytes, &combined); err != nil {
			continue
		}

		if combined.Data.Symbol == "" || combined.Data.Close == "" {
			continue
		}

		price, err := strconv.ParseFloat(combined.Data.Close, 64)
		if err != nil {
			continue
		}

		ws.onPrice(combined.Data.Symbol, price)
	}
}
