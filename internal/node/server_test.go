package node

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nickai/cli/internal/node/pb"
)

// startTestServer creates a server on a random port and returns the client
// connection plus a cleanup function.
func startTestServer(t *testing.T) (pb.NickNodeClient, func()) {
	t.Helper()

	srv := NewServer()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterNickNodeServer(grpcSrv, srv)
	srv.grpcServer = grpcSrv

	go grpcSrv.Serve(lis)

	conn, err := grpc.DialContext(
		context.Background(),
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	client := pb.NewNickNodeClient(conn)

	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return client, cleanup
}

func TestPing(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if resp.Version == "" {
		t.Error("expected non-empty version")
	}
	if resp.UptimeSeconds < 0 {
		t.Error("expected non-negative uptime")
	}
}

func TestDeployAndListStrategies(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy a strategy.
	deployResp, err := client.DeployStrategy(ctx, &pb.DeployStrategyRequest{
		Spec: &pb.StrategySpec{
			Name:   "Test RSI",
			Symbol: "BTCUSDT",
			EntryRules: []*pb.StrategyCondition{
				{Indicator: "rsi", Operator: "<", Value: 30},
			},
			ExitRules: []*pb.StrategyCondition{
				{Indicator: "rsi", Operator: ">", Value: 70},
			},
			Interval: "5m",
		},
	})
	if err != nil {
		t.Fatalf("DeployStrategy failed: %v", err)
	}
	if deployResp.ID == "" {
		t.Fatal("expected non-empty strategy ID")
	}

	// List strategies.
	listResp, err := client.ListStrategies(ctx, &pb.ListStrategiesRequest{})
	if err != nil {
		t.Fatalf("ListStrategies failed: %v", err)
	}
	if len(listResp.Strategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(listResp.Strategies))
	}
	if listResp.Strategies[0].ID != deployResp.ID {
		t.Errorf("strategy ID mismatch: got %s, want %s", listResp.Strategies[0].ID, deployResp.ID)
	}
	if listResp.Strategies[0].Status != pb.StrategyStatusRunning {
		t.Errorf("expected RUNNING status, got %v", listResp.Strategies[0].Status)
	}
	if listResp.Strategies[0].Spec.Name != "Test RSI" {
		t.Errorf("expected name 'Test RSI', got %q", listResp.Strategies[0].Spec.Name)
	}
}

func TestStopStrategy(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy first.
	deployResp, err := client.DeployStrategy(ctx, &pb.DeployStrategyRequest{
		Spec: &pb.StrategySpec{
			Name:   "Stop Me",
			Symbol: "ETHUSDT",
		},
	})
	if err != nil {
		t.Fatalf("DeployStrategy failed: %v", err)
	}

	// Stop it.
	stopResp, err := client.StopStrategy(ctx, &pb.StopStrategyRequest{ID: deployResp.ID})
	if err != nil {
		t.Fatalf("StopStrategy failed: %v", err)
	}
	if !stopResp.Stopped {
		t.Error("expected stopped=true")
	}

	// Verify status changed.
	listResp, err := client.ListStrategies(ctx, &pb.ListStrategiesRequest{})
	if err != nil {
		t.Fatalf("ListStrategies failed: %v", err)
	}
	if len(listResp.Strategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(listResp.Strategies))
	}
	if listResp.Strategies[0].Status != pb.StrategyStatusStopped {
		t.Errorf("expected STOPPED status, got %v", listResp.Strategies[0].Status)
	}
}

func TestStopStrategyNotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.StopStrategy(ctx, &pb.StopStrategyRequest{ID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestCreateAndListAlerts(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create an alert.
	createResp, err := client.CreateAlert(ctx, &pb.CreateAlertRequest{
		Spec: &pb.AlertSpec{
			Symbol:   "BTCUSDT",
			Operator: ">",
			Target:   100000,
		},
	})
	if err != nil {
		t.Fatalf("CreateAlert failed: %v", err)
	}
	if createResp.ID == "" {
		t.Fatal("expected non-empty alert ID")
	}

	// List alerts.
	listResp, err := client.ListAlerts(ctx, &pb.ListAlertsRequest{})
	if err != nil {
		t.Fatalf("ListAlerts failed: %v", err)
	}
	if len(listResp.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(listResp.Alerts))
	}
	if listResp.Alerts[0].ID != createResp.ID {
		t.Errorf("alert ID mismatch: got %s, want %s", listResp.Alerts[0].ID, createResp.ID)
	}
	if listResp.Alerts[0].Spec.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %q", listResp.Alerts[0].Spec.Symbol)
	}
	if listResp.Alerts[0].Spec.Target != 100000 {
		t.Errorf("expected target 100000, got %f", listResp.Alerts[0].Spec.Target)
	}
	if listResp.Alerts[0].Triggered {
		t.Error("expected alert to not be triggered yet")
	}
}

func TestCreateAlertValidation(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Missing spec.
	_, err := client.CreateAlert(ctx, &pb.CreateAlertRequest{})
	if err == nil {
		t.Fatal("expected error for nil spec")
	}

	// Missing symbol.
	_, err = client.CreateAlert(ctx, &pb.CreateAlertRequest{
		Spec: &pb.AlertSpec{Operator: ">", Target: 100},
	})
	if err == nil {
		t.Fatal("expected error for missing symbol")
	}

	// Invalid operator.
	_, err = client.CreateAlert(ctx, &pb.CreateAlertRequest{
		Spec: &pb.AlertSpec{Symbol: "BTC", Operator: "==", Target: 100},
	})
	if err == nil {
		t.Fatal("expected error for invalid operator")
	}
}

func TestGetStatus(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy a strategy so we have something to count.
	_, err := client.DeployStrategy(ctx, &pb.DeployStrategyRequest{
		Spec: &pb.StrategySpec{Name: "Status Test", Symbol: "SOLUSDT"},
	})
	if err != nil {
		t.Fatalf("DeployStrategy failed: %v", err)
	}

	// Create an alert.
	_, err = client.CreateAlert(ctx, &pb.CreateAlertRequest{
		Spec: &pb.AlertSpec{Symbol: "BTCUSDT", Operator: ">", Target: 999999},
	})
	if err != nil {
		t.Fatalf("CreateAlert failed: %v", err)
	}

	resp, err := client.GetStatus(ctx, &pb.GetStatusRequest{})
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if resp.Version == "" {
		t.Error("expected non-empty version")
	}
	if resp.UptimeSeconds < 0 {
		t.Error("expected non-negative uptime")
	}
	if resp.RunningStrategies != 1 {
		t.Errorf("expected 1 running strategy, got %d", resp.RunningStrategies)
	}
	if resp.ActiveAlerts != 1 {
		t.Errorf("expected 1 active alert, got %d", resp.ActiveAlerts)
	}
	if resp.Goroutines <= 0 {
		t.Error("expected positive goroutine count")
	}
	if resp.MemoryBytes <= 0 {
		t.Error("expected positive memory usage")
	}
}

func TestServerStartStop(t *testing.T) {
	srv := NewServer()

	// Pick a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close() // Release it so Start can bind.

	// Start in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(addr)
	}()

	// Give it a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Connect and ping.
	conn, err := grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewNickNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if resp.Version == "" {
		t.Error("expected non-empty version")
	}

	// Stop server.
	srv.Stop()

	// Server.Start should have returned after Stop.
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop within timeout")
	}
}

func TestDeployMultipleStrategies(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	ids := make([]string, len(symbols))

	for i, sym := range symbols {
		resp, err := client.DeployStrategy(ctx, &pb.DeployStrategyRequest{
			Spec: &pb.StrategySpec{Name: sym + " Monitor", Symbol: sym},
		})
		if err != nil {
			t.Fatalf("DeployStrategy(%s) failed: %v", sym, err)
		}
		ids[i] = resp.ID
	}

	// List should show all 3.
	listResp, err := client.ListStrategies(ctx, &pb.ListStrategiesRequest{})
	if err != nil {
		t.Fatalf("ListStrategies failed: %v", err)
	}
	if len(listResp.Strategies) != 3 {
		t.Fatalf("expected 3 strategies, got %d", len(listResp.Strategies))
	}

	// Stop one.
	_, err = client.StopStrategy(ctx, &pb.StopStrategyRequest{ID: ids[1]})
	if err != nil {
		t.Fatalf("StopStrategy failed: %v", err)
	}

	// Check status.
	status, err := client.GetStatus(ctx, &pb.GetStatusRequest{})
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.RunningStrategies != 2 {
		t.Errorf("expected 2 running strategies after stop, got %d", status.RunningStrategies)
	}
}

func TestDeployStrategyValidation(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Nil spec.
	_, err := client.DeployStrategy(ctx, &pb.DeployStrategyRequest{})
	if err == nil {
		t.Fatal("expected error for nil spec")
	}

	// Empty symbol.
	_, err = client.DeployStrategy(ctx, &pb.DeployStrategyRequest{
		Spec: &pb.StrategySpec{Name: "No Symbol"},
	})
	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}
