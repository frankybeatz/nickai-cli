package node

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/nickai/cli/internal/node/pb"
)

// DefaultAddr is the default Nick Node server address.
const DefaultAddr = "localhost:9400"

// Client wraps a gRPC connection to a NickNode server.
type Client struct {
	conn   *grpc.ClientConn
	client pb.NickNodeClient
	addr   string
	token  string
}

// NewClient connects to a NickNode server at the given address.
// The address should be in "host:port" format.
func NewClient(addr string) (*Client, error) {
	if addr == "" {
		addr = DefaultAddr
	}
	token := os.Getenv("NICKAI_NODE_TOKEN")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
		grpc.WithBlock(),
	}
	if token != "" {
		dialOpts = append(dialOpts,
			grpc.WithUnaryInterceptor(authUnaryClientInterceptor(token)),
			grpc.WithStreamInterceptor(authStreamClientInterceptor(token)),
		)
	}

	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node at %s: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewNickNodeClient(conn),
		addr:   addr,
		token:  token,
	}, nil
}

func authUnaryClientInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-node-token", token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func authStreamClientInterceptor(token string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-node-token", token)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Addr returns the server address this client is connected to.
func (c *Client) Addr() string {
	return c.addr
}

// ---------------------------------------------------------------------------
// Ping
// ---------------------------------------------------------------------------

// Ping performs a health check against the node.
func (c *Client) Ping(ctx context.Context) (*pb.PingResponse, error) {
	return c.client.Ping(ctx, &pb.PingRequest{})
}

// ---------------------------------------------------------------------------
// Strategy
// ---------------------------------------------------------------------------

// DeployStrategy sends a strategy to the node for persistent execution.
func (c *Client) DeployStrategy(ctx context.Context, spec *pb.StrategySpec) (string, error) {
	resp, err := c.client.DeployStrategy(ctx, &pb.DeployStrategyRequest{Spec: spec})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ListStrategies returns all strategies on the node.
func (c *Client) ListStrategies(ctx context.Context) ([]*pb.StrategyInfo, error) {
	resp, err := c.client.ListStrategies(ctx, &pb.ListStrategiesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Strategies, nil
}

// StopStrategy stops a running strategy by ID.
func (c *Client) StopStrategy(ctx context.Context, id string) (bool, error) {
	resp, err := c.client.StopStrategy(ctx, &pb.StopStrategyRequest{ID: id})
	if err != nil {
		return false, err
	}
	return resp.Stopped, nil
}

// ---------------------------------------------------------------------------
// Price streaming
// ---------------------------------------------------------------------------

// StreamPrices opens a server-streaming connection for live price ticks.
// The returned channel receives ticks until the context is cancelled or the
// stream ends. The caller must cancel the context to stop the stream.
func (c *Client) StreamPrices(ctx context.Context, symbols []string) (<-chan *pb.PriceTick, <-chan error) {
	tickCh := make(chan *pb.PriceTick, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(tickCh)
		defer close(errCh)

		stream, err := c.client.StreamPrices(ctx, &pb.StreamPricesRequest{Symbols: symbols})
		if err != nil {
			errCh <- err
			return
		}

		for {
			tick, err := stream.Recv()
			if err != nil {
				if ctx.Err() == nil {
					errCh <- err
				}
				return
			}
			select {
			case tickCh <- tick:
			case <-ctx.Done():
				return
			}
		}
	}()

	return tickCh, errCh
}

// ---------------------------------------------------------------------------
// Backtest
// ---------------------------------------------------------------------------

// SubmitBacktest sends a backtest job to the node for async execution.
func (c *Client) SubmitBacktest(ctx context.Context, spec *pb.BacktestSpec) (string, error) {
	resp, err := c.client.SubmitBacktest(ctx, &pb.SubmitBacktestRequest{Spec: spec})
	if err != nil {
		return "", err
	}
	return resp.JobID, nil
}

// GetBacktestResult checks the status of a backtest job.
func (c *Client) GetBacktestResult(ctx context.Context, jobID string) (*pb.GetBacktestResultResponse, error) {
	return c.client.GetBacktestResult(ctx, &pb.GetBacktestResultRequest{JobID: jobID})
}

// ---------------------------------------------------------------------------
// Alerts
// ---------------------------------------------------------------------------

// CreateAlert registers a price alert on the node.
func (c *Client) CreateAlert(ctx context.Context, spec *pb.AlertSpec) (string, error) {
	resp, err := c.client.CreateAlert(ctx, &pb.CreateAlertRequest{Spec: spec})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ListAlerts returns all alerts registered on the node.
func (c *Client) ListAlerts(ctx context.Context) ([]*pb.AlertInfo, error) {
	resp, err := c.client.ListAlerts(ctx, &pb.ListAlertsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Alerts, nil
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

// GetStatus returns the node's current status.
func (c *Client) GetStatus(ctx context.Context) (*pb.GetStatusResponse, error) {
	return c.client.GetStatus(ctx, &pb.GetStatusRequest{})
}
