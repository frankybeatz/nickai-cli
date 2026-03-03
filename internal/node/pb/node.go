// Package pb contains the message types and gRPC service interface for the
// NickNode service. These types correspond to proto/nick/v1/node.proto and
// should be regenerated with protoc if the proto file changes. They were
// written by hand because protoc tooling was not available at build time.
//
// To regenerate from proto:
//
//	protoc --go_out=. --go-grpc_out=. proto/nick/v1/node.proto
package pb

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// Ping
// ---------------------------------------------------------------------------

type PingRequest struct{}

type PingResponse struct {
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// ---------------------------------------------------------------------------
// Strategy
// ---------------------------------------------------------------------------

type StrategyCondition struct {
	Indicator   string  `json:"indicator"`
	Operator    string  `json:"operator"`
	Value       float64 `json:"value"`
	CompareWith string  `json:"compare_with,omitempty"`
}

type StrategySpec struct {
	Name          string               `json:"name"`
	Symbol        string               `json:"symbol"`
	EntryRules    []*StrategyCondition  `json:"entry_rules"`
	ExitRules     []*StrategyCondition  `json:"exit_rules"`
	StopLossPct   float64              `json:"stop_loss_pct"`
	TakeProfitPct float64              `json:"take_profit_pct"`
	PositionSize  float64              `json:"position_size"`
	Interval      string               `json:"interval"`
}

type StrategyStatus int32

const (
	StrategyStatusUnspecified StrategyStatus = 0
	StrategyStatusRunning     StrategyStatus = 1
	StrategyStatusStopped     StrategyStatus = 2
	StrategyStatusErrored     StrategyStatus = 3
)

func (s StrategyStatus) String() string {
	switch s {
	case StrategyStatusRunning:
		return "RUNNING"
	case StrategyStatusStopped:
		return "STOPPED"
	case StrategyStatusErrored:
		return "ERRORED"
	default:
		return "UNSPECIFIED"
	}
}

type StrategyInfo struct {
	ID         string         `json:"id"`
	Spec       *StrategySpec  `json:"spec"`
	Status     StrategyStatus `json:"status"`
	DeployedAt time.Time      `json:"deployed_at"`
	Error      string         `json:"error,omitempty"`
}

type DeployStrategyRequest struct {
	Spec *StrategySpec `json:"spec"`
}

type DeployStrategyResponse struct {
	ID string `json:"id"`
}

type ListStrategiesRequest struct{}

type ListStrategiesResponse struct {
	Strategies []*StrategyInfo `json:"strategies"`
}

type StopStrategyRequest struct {
	ID string `json:"id"`
}

type StopStrategyResponse struct {
	Stopped bool `json:"stopped"`
}

// ---------------------------------------------------------------------------
// Price streaming
// ---------------------------------------------------------------------------

type StreamPricesRequest struct {
	Symbols []string `json:"symbols"`
}

type PriceTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume24H float64  `json:"volume_24h"`
	Timestamp time.Time `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// Backtest
// ---------------------------------------------------------------------------

type BacktestCondition struct {
	Indicator   string  `json:"indicator"`
	Operator    string  `json:"operator"`
	Value       float64 `json:"value"`
	CompareWith string  `json:"compare_with,omitempty"`
}

type BacktestSpec struct {
	Name          string              `json:"name"`
	Symbol        string              `json:"symbol"`
	EntryRules    []*BacktestCondition `json:"entry_rules"`
	ExitRules     []*BacktestCondition `json:"exit_rules"`
	StopLossPct   float64             `json:"stop_loss_pct"`
	TakeProfitPct float64             `json:"take_profit_pct"`
	PositionSize  float64             `json:"position_size"`
	Period        string              `json:"period"`
	SlippageBps   float64             `json:"slippage_bps"`
	CommissionBps float64             `json:"commission_bps"`
}

type BacktestTrade struct {
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnLPct     float64   `json:"pnl_pct"`
	Reason     string    `json:"reason"`
}

type BacktestResult struct {
	Strategy     string           `json:"strategy"`
	Symbol       string           `json:"symbol"`
	Period       string           `json:"period"`
	Trades       []*BacktestTrade `json:"trades"`
	TotalTrades  int32            `json:"total_trades"`
	WinRate      float64          `json:"win_rate"`
	TotalReturn  float64          `json:"total_return"`
	SharpeRatio  float64          `json:"sharpe_ratio"`
	MaxDrawdown  float64          `json:"max_drawdown"`
	ProfitFactor float64          `json:"profit_factor"`
	BestTrade    float64          `json:"best_trade"`
	WorstTrade   float64          `json:"worst_trade"`
	EquityCurve  []float64        `json:"equity_curve"`
}

type BacktestJobStatus int32

const (
	BacktestJobStatusUnspecified BacktestJobStatus = 0
	BacktestJobStatusPending    BacktestJobStatus = 1
	BacktestJobStatusRunning    BacktestJobStatus = 2
	BacktestJobStatusCompleted  BacktestJobStatus = 3
	BacktestJobStatusFailed     BacktestJobStatus = 4
)

func (s BacktestJobStatus) String() string {
	switch s {
	case BacktestJobStatusPending:
		return "PENDING"
	case BacktestJobStatusRunning:
		return "RUNNING"
	case BacktestJobStatusCompleted:
		return "COMPLETED"
	case BacktestJobStatusFailed:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

type SubmitBacktestRequest struct {
	Spec *BacktestSpec `json:"spec"`
}

type SubmitBacktestResponse struct {
	JobID string `json:"job_id"`
}

type GetBacktestResultRequest struct {
	JobID string `json:"job_id"`
}

type GetBacktestResultResponse struct {
	Status BacktestJobStatus `json:"status"`
	Result *BacktestResult   `json:"result,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Alerts
// ---------------------------------------------------------------------------

type AlertSpec struct {
	Symbol   string  `json:"symbol"`
	Operator string  `json:"operator"`
	Target   float64 `json:"target"`
}

type AlertInfo struct {
	ID        string    `json:"id"`
	Spec      *AlertSpec `json:"spec"`
	CreatedAt time.Time `json:"created_at"`
	Triggered bool      `json:"triggered"`
}

type CreateAlertRequest struct {
	Spec *AlertSpec `json:"spec"`
}

type CreateAlertResponse struct {
	ID string `json:"id"`
}

type ListAlertsRequest struct{}

type ListAlertsResponse struct {
	Alerts []*AlertInfo `json:"alerts"`
}

// ---------------------------------------------------------------------------
// Node status
// ---------------------------------------------------------------------------

type GetStatusRequest struct{}

type GetStatusResponse struct {
	Version           string   `json:"version"`
	UptimeSeconds     int64    `json:"uptime_seconds"`
	RunningStrategies int32    `json:"running_strategies"`
	ActiveAlerts      int32    `json:"active_alerts"`
	ConnectedSymbols  []string `json:"connected_symbols"`
	MemoryBytes       int64    `json:"memory_bytes"`
	Goroutines        int32    `json:"goroutines"`
}

// ---------------------------------------------------------------------------
// Service interface and registration
// ---------------------------------------------------------------------------

// NickNodeServer is the server-side interface for the NickNode service.
// Implementations must embed UnimplementedNickNodeServer.
type NickNodeServer interface {
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	DeployStrategy(context.Context, *DeployStrategyRequest) (*DeployStrategyResponse, error)
	ListStrategies(context.Context, *ListStrategiesRequest) (*ListStrategiesResponse, error)
	StopStrategy(context.Context, *StopStrategyRequest) (*StopStrategyResponse, error)
	StreamPrices(*StreamPricesRequest, NickNode_StreamPricesServer) error
	SubmitBacktest(context.Context, *SubmitBacktestRequest) (*SubmitBacktestResponse, error)
	GetBacktestResult(context.Context, *GetBacktestResultRequest) (*GetBacktestResultResponse, error)
	CreateAlert(context.Context, *CreateAlertRequest) (*CreateAlertResponse, error)
	ListAlerts(context.Context, *ListAlertsRequest) (*ListAlertsResponse, error)
	GetStatus(context.Context, *GetStatusRequest) (*GetStatusResponse, error)
}

// NickNode_StreamPricesServer is the server-side stream for StreamPrices.
type NickNode_StreamPricesServer interface {
	Send(*PriceTick) error
	grpc.ServerStream
}

// NickNode_StreamPricesClient is the client-side stream for StreamPrices.
type NickNode_StreamPricesClient interface {
	Recv() (*PriceTick, error)
	grpc.ClientStream
}

// UnimplementedNickNodeServer provides default implementations that return
// "unimplemented" errors. Embed this in concrete server implementations.
type UnimplementedNickNodeServer struct{}

func (UnimplementedNickNodeServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) DeployStrategy(context.Context, *DeployStrategyRequest) (*DeployStrategyResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) ListStrategies(context.Context, *ListStrategiesRequest) (*ListStrategiesResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) StopStrategy(context.Context, *StopStrategyRequest) (*StopStrategyResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) StreamPrices(*StreamPricesRequest, NickNode_StreamPricesServer) error {
	return grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) SubmitBacktest(context.Context, *SubmitBacktestRequest) (*SubmitBacktestResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) GetBacktestResult(context.Context, *GetBacktestResultRequest) (*GetBacktestResultResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) CreateAlert(context.Context, *CreateAlertRequest) (*CreateAlertResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) ListAlerts(context.Context, *ListAlertsRequest) (*ListAlertsResponse, error) {
	return nil, grpc.ErrServerStopped
}
func (UnimplementedNickNodeServer) GetStatus(context.Context, *GetStatusRequest) (*GetStatusResponse, error) {
	return nil, grpc.ErrServerStopped
}

// NickNodeClient is the client-side interface for the NickNode service.
type NickNodeClient interface {
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	DeployStrategy(ctx context.Context, in *DeployStrategyRequest, opts ...grpc.CallOption) (*DeployStrategyResponse, error)
	ListStrategies(ctx context.Context, in *ListStrategiesRequest, opts ...grpc.CallOption) (*ListStrategiesResponse, error)
	StopStrategy(ctx context.Context, in *StopStrategyRequest, opts ...grpc.CallOption) (*StopStrategyResponse, error)
	StreamPrices(ctx context.Context, in *StreamPricesRequest, opts ...grpc.CallOption) (NickNode_StreamPricesClient, error)
	SubmitBacktest(ctx context.Context, in *SubmitBacktestRequest, opts ...grpc.CallOption) (*SubmitBacktestResponse, error)
	GetBacktestResult(ctx context.Context, in *GetBacktestResultRequest, opts ...grpc.CallOption) (*GetBacktestResultResponse, error)
	CreateAlert(ctx context.Context, in *CreateAlertRequest, opts ...grpc.CallOption) (*CreateAlertResponse, error)
	ListAlerts(ctx context.Context, in *ListAlertsRequest, opts ...grpc.CallOption) (*ListAlertsResponse, error)
	GetStatus(ctx context.Context, in *GetStatusRequest, opts ...grpc.CallOption) (*GetStatusResponse, error)
}
