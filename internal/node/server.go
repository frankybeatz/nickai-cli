// Package node implements the Nick Node gRPC server — an always-on process
// for persistent strategy execution, price streaming, backtest offloading,
// and alert dispatch.
package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/node/pb"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// ---------------------------------------------------------------------------
// Running strategy
// ---------------------------------------------------------------------------

// RunningStrategy tracks a deployed strategy and its cancellation handle.
type RunningStrategy struct {
	Info   *pb.StrategyInfo
	cancel context.CancelFunc
}

// ---------------------------------------------------------------------------
// Backtest job
// ---------------------------------------------------------------------------

type backtestJob struct {
	ID     string
	Spec   *pb.BacktestSpec
	Status pb.BacktestJobStatus
	Result *pb.BacktestResult
	Error  string
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// Server implements pb.NickNodeServer.
type Server struct {
	pb.UnimplementedNickNodeServer

	startTime time.Time

	mu         sync.RWMutex
	strategies map[string]*RunningStrategy
	alerts     map[string]*pb.AlertInfo
	btJobs     map[string]*backtestJob

	// Price feed tracking.
	priceMu    sync.RWMutex
	priceFeeds map[string]bool // symbols being streamed

	grpcServer *grpc.Server
}

// NewServer creates a new NickNode server.
func NewServer() *Server {
	return &Server{
		startTime:  time.Now(),
		strategies: make(map[string]*RunningStrategy),
		alerts:     make(map[string]*pb.AlertInfo),
		btJobs:     make(map[string]*backtestJob),
		priceFeeds: make(map[string]bool),
	}
}

// Start binds the gRPC server to the given address and begins serving.
// It blocks until Stop is called or the listener is closed.
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterNickNodeServer(s.grpcServer, s)

	logging.Info("nick-node starting", "addr", addr, "version", Version)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully shuts down the server, stopping all strategies.
func (s *Server) Stop() {
	logging.Info("nick-node shutting down")

	// Stop all running strategies.
	s.mu.Lock()
	for id, rs := range s.strategies {
		if rs.cancel != nil {
			rs.cancel()
		}
		rs.Info.Status = pb.StrategyStatusStopped
		logging.Info("stopped strategy", "id", id)
	}
	s.mu.Unlock()

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// ---------------------------------------------------------------------------
// RPC: Ping
// ---------------------------------------------------------------------------

func (s *Server) Ping(_ context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Version:       Version,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}, nil
}

// ---------------------------------------------------------------------------
// RPC: DeployStrategy
// ---------------------------------------------------------------------------

func (s *Server) DeployStrategy(_ context.Context, req *pb.DeployStrategyRequest) (*pb.DeployStrategyResponse, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}
	if req.Spec.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	id := fmt.Sprintf("strat-%d", time.Now().UnixNano())

	info := &pb.StrategyInfo{
		ID:         id,
		Spec:       req.Spec,
		Status:     pb.StrategyStatusRunning,
		DeployedAt: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	rs := &RunningStrategy{
		Info:   info,
		cancel: cancel,
	}

	s.mu.Lock()
	s.strategies[id] = rs
	s.mu.Unlock()

	// Launch strategy monitoring goroutine.
	go s.runStrategy(ctx, rs)

	logging.Info("deployed strategy", "id", id, "symbol", req.Spec.Symbol, "name", req.Spec.Name)

	return &pb.DeployStrategyResponse{ID: id}, nil
}

// runStrategy monitors a strategy in a background goroutine. It polls the
// Binance REST API at the configured interval (default 1 minute) and evaluates
// entry/exit conditions. For now this is a monitoring skeleton — actual order
// execution will be wired in later.
func (s *Server) runStrategy(ctx context.Context, rs *RunningStrategy) {
	interval := 60 * time.Second
	if rs.Info.Spec.Interval != "" {
		switch rs.Info.Spec.Interval {
		case "1m":
			interval = 1 * time.Minute
		case "5m":
			interval = 5 * time.Minute
		case "15m":
			interval = 15 * time.Minute
		case "1h":
			interval = 1 * time.Hour
		case "4h":
			interval = 4 * time.Hour
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	symbol := rs.Info.Spec.Symbol
	logging.Info("strategy monitor started", "id", rs.Info.ID, "symbol", symbol, "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logging.Info("strategy monitor stopped", "id", rs.Info.ID)
			return
		case <-ticker.C:
			price, err := fetchLatestPrice(symbol)
			if err != nil {
				logging.Warn("strategy price fetch failed", "id", rs.Info.ID, "error", err)
				continue
			}
			logging.Debug("strategy tick", "id", rs.Info.ID, "symbol", symbol, "price", price)
			// TODO: evaluate entry/exit conditions against current price and indicators.
			// When conditions are met, emit signals or execute orders.
		}
	}
}

// ---------------------------------------------------------------------------
// RPC: ListStrategies
// ---------------------------------------------------------------------------

func (s *Server) ListStrategies(_ context.Context, _ *pb.ListStrategiesRequest) (*pb.ListStrategiesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*pb.StrategyInfo, 0, len(s.strategies))
	for _, rs := range s.strategies {
		list = append(list, rs.Info)
	}
	return &pb.ListStrategiesResponse{Strategies: list}, nil
}

// ---------------------------------------------------------------------------
// RPC: StopStrategy
// ---------------------------------------------------------------------------

func (s *Server) StopStrategy(_ context.Context, req *pb.StopStrategyRequest) (*pb.StopStrategyResponse, error) {
	if req.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rs, ok := s.strategies[req.ID]
	if !ok {
		// Try prefix match.
		for id, r := range s.strategies {
			if strings.HasPrefix(id, req.ID) {
				rs = r
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "strategy %q not found", req.ID)
	}

	if rs.cancel != nil {
		rs.cancel()
	}
	rs.Info.Status = pb.StrategyStatusStopped
	logging.Info("stopped strategy via RPC", "id", rs.Info.ID)

	return &pb.StopStrategyResponse{Stopped: true}, nil
}

// ---------------------------------------------------------------------------
// RPC: StreamPrices
// ---------------------------------------------------------------------------

func (s *Server) StreamPrices(req *pb.StreamPricesRequest, stream pb.NickNode_StreamPricesServer) error {
	if len(req.Symbols) == 0 {
		return status.Error(codes.InvalidArgument, "at least one symbol is required")
	}

	// Track connected symbols.
	s.priceMu.Lock()
	for _, sym := range req.Symbols {
		s.priceFeeds[strings.ToUpper(sym)] = true
	}
	s.priceMu.Unlock()

	defer func() {
		s.priceMu.Lock()
		for _, sym := range req.Symbols {
			delete(s.priceFeeds, strings.ToUpper(sym))
		}
		s.priceMu.Unlock()
	}()

	// Poll every 5 seconds (WebSocket upgrade comes later).
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			for _, sym := range req.Symbols {
				price, err := fetchLatestPrice(sym)
				if err != nil {
					logging.Warn("price fetch failed", "symbol", sym, "error", err)
					continue
				}
				tick := &pb.PriceTick{
					Symbol:    strings.ToUpper(sym),
					Price:     price,
					Timestamp: time.Now(),
				}
				if err := stream.Send(tick); err != nil {
					return err
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RPC: SubmitBacktest
// ---------------------------------------------------------------------------

func (s *Server) SubmitBacktest(_ context.Context, req *pb.SubmitBacktestRequest) (*pb.SubmitBacktestResponse, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}
	if req.Spec.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	jobID := fmt.Sprintf("bt-%d", time.Now().UnixNano())

	job := &backtestJob{
		ID:     jobID,
		Spec:   req.Spec,
		Status: pb.BacktestJobStatusPending,
	}

	s.mu.Lock()
	s.btJobs[jobID] = job
	s.mu.Unlock()

	// Run backtest asynchronously.
	go s.executeBacktest(job)

	logging.Info("backtest submitted", "job_id", jobID, "symbol", req.Spec.Symbol)
	return &pb.SubmitBacktestResponse{JobID: jobID}, nil
}

func (s *Server) executeBacktest(job *backtestJob) {
	s.mu.Lock()
	job.Status = pb.BacktestJobStatusRunning
	s.mu.Unlock()

	// Convert pb spec to backtest.Strategy.
	strat := backtestSpecToStrategy(job.Spec)

	result, err := backtest.Run(strat)
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		job.Status = pb.BacktestJobStatusFailed
		job.Error = err.Error()
		logging.Warn("backtest failed", "job_id", job.ID, "error", err)
		return
	}

	job.Status = pb.BacktestJobStatusCompleted
	job.Result = backtestResultToPB(result)
	logging.Info("backtest completed", "job_id", job.ID, "trades", result.TotalTrades)
}

// ---------------------------------------------------------------------------
// RPC: GetBacktestResult
// ---------------------------------------------------------------------------

func (s *Server) GetBacktestResult(_ context.Context, req *pb.GetBacktestResultRequest) (*pb.GetBacktestResultResponse, error) {
	if req.JobID == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.btJobs[req.JobID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "backtest job %q not found", req.JobID)
	}

	return &pb.GetBacktestResultResponse{
		Status: job.Status,
		Result: job.Result,
		Error:  job.Error,
	}, nil
}

// ---------------------------------------------------------------------------
// RPC: CreateAlert
// ---------------------------------------------------------------------------

func (s *Server) CreateAlert(_ context.Context, req *pb.CreateAlertRequest) (*pb.CreateAlertResponse, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}
	if req.Spec.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.Spec.Operator != ">" && req.Spec.Operator != "<" {
		return nil, status.Errorf(codes.InvalidArgument, "operator must be '>' or '<', got %q", req.Spec.Operator)
	}

	id := fmt.Sprintf("alert-%d", time.Now().UnixNano())

	info := &pb.AlertInfo{
		ID:        id,
		Spec:      req.Spec,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.alerts[id] = info
	s.mu.Unlock()

	// Start alert monitoring in background.
	go s.monitorAlert(info)

	logging.Info("alert created", "id", id, "symbol", req.Spec.Symbol, "op", req.Spec.Operator, "target", req.Spec.Target)
	return &pb.CreateAlertResponse{ID: id}, nil
}

func (s *Server) monitorAlert(info *pb.AlertInfo) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		current, exists := s.alerts[info.ID]
		s.mu.RUnlock()

		if !exists || current.Triggered {
			return
		}

		price, err := fetchLatestPrice(info.Spec.Symbol)
		if err != nil {
			logging.Warn("alert price fetch failed", "id", info.ID, "error", err)
			continue
		}

		triggered := false
		switch info.Spec.Operator {
		case ">":
			triggered = price > info.Spec.Target
		case "<":
			triggered = price < info.Spec.Target
		}

		if triggered {
			s.mu.Lock()
			if a, ok := s.alerts[info.ID]; ok {
				a.Triggered = true
			}
			s.mu.Unlock()
			logging.Info("alert triggered", "id", info.ID, "symbol", info.Spec.Symbol, "price", price, "target", info.Spec.Target)
			// TODO: dispatch notification (webhook, push, etc.)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// RPC: ListAlerts
// ---------------------------------------------------------------------------

func (s *Server) ListAlerts(_ context.Context, _ *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*pb.AlertInfo, 0, len(s.alerts))
	for _, a := range s.alerts {
		list = append(list, a)
	}
	return &pb.ListAlertsResponse{Alerts: list}, nil
}

// ---------------------------------------------------------------------------
// RPC: GetStatus
// ---------------------------------------------------------------------------

func (s *Server) GetStatus(_ context.Context, _ *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	s.mu.RLock()
	runningCount := int32(0)
	for _, rs := range s.strategies {
		if rs.Info.Status == pb.StrategyStatusRunning {
			runningCount++
		}
	}
	activeAlerts := int32(0)
	for _, a := range s.alerts {
		if !a.Triggered {
			activeAlerts++
		}
	}
	s.mu.RUnlock()

	s.priceMu.RLock()
	symbols := make([]string, 0, len(s.priceFeeds))
	for sym := range s.priceFeeds {
		symbols = append(symbols, sym)
	}
	s.priceMu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &pb.GetStatusResponse{
		Version:           Version,
		UptimeSeconds:     int64(time.Since(s.startTime).Seconds()),
		RunningStrategies: runningCount,
		ActiveAlerts:      activeAlerts,
		ConnectedSymbols:  symbols,
		MemoryBytes:       int64(memStats.Alloc),
		Goroutines:        int32(runtime.NumGoroutine()),
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fetchLatestPrice gets the latest price from Binance REST API.
func fetchLatestPrice(symbol string) (float64, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if !strings.HasSuffix(sym, "USDT") && !strings.HasSuffix(sym, "USDC") && !strings.HasSuffix(sym, "USD") {
		sym += "USDT"
	}

	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", sym)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("binance request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("binance API returned %d", resp.StatusCode)
	}

	var ticker struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ticker); err != nil {
		return 0, fmt.Errorf("failed to decode price: %w", err)
	}

	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price %q: %w", ticker.Price, err)
	}
	return price, nil
}

// backtestSpecToStrategy converts a pb.BacktestSpec to a backtest.Strategy.
func backtestSpecToStrategy(spec *pb.BacktestSpec) backtest.Strategy {
	strat := backtest.Strategy{
		Name:          spec.Name,
		Symbol:        spec.Symbol,
		StopLossPct:   spec.StopLossPct,
		TakeProfitPct: spec.TakeProfitPct,
		PositionSize:  spec.PositionSize,
		Period:        spec.Period,
		SlippageBps:   spec.SlippageBps,
		CommissionBps: spec.CommissionBps,
	}

	for _, c := range spec.EntryRules {
		strat.EntryRules = append(strat.EntryRules, backtest.Condition{
			Indicator:   c.Indicator,
			Operator:    c.Operator,
			Value:       c.Value,
			CompareWith: c.CompareWith,
		})
	}
	for _, c := range spec.ExitRules {
		strat.ExitRules = append(strat.ExitRules, backtest.Condition{
			Indicator:   c.Indicator,
			Operator:    c.Operator,
			Value:       c.Value,
			CompareWith: c.CompareWith,
		})
	}

	return strat
}

// backtestResultToPB converts a backtest.Result to a pb.BacktestResult.
func backtestResultToPB(r *backtest.Result) *pb.BacktestResult {
	if r == nil {
		return nil
	}

	pbr := &pb.BacktestResult{
		Strategy:     r.Strategy,
		Symbol:       r.Symbol,
		Period:       r.Period,
		TotalTrades:  int32(r.TotalTrades),
		WinRate:      r.WinRate,
		TotalReturn:  r.TotalReturn,
		SharpeRatio:  r.SharpeRatio,
		MaxDrawdown:  r.MaxDrawdown,
		ProfitFactor: r.ProfitFactor,
		BestTrade:    r.BestTrade,
		WorstTrade:   r.WorstTrade,
		EquityCurve:  r.EquityCurve,
	}

	for _, t := range r.Trades {
		pbr.Trades = append(pbr.Trades, &pb.BacktestTrade{
			EntryTime:  t.EntryTime,
			ExitTime:   t.ExitTime,
			EntryPrice: t.EntryPrice,
			ExitPrice:  t.ExitPrice,
			PnLPct:     t.PnLPct,
			Reason:     t.Reason,
		})
	}

	return pbr
}
