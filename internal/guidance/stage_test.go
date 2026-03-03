package guidance

import (
	"strings"
	"testing"
)

func TestDetectStage_Fresh(t *testing.T) {
	ctx := StageContext{}
	if got := DetectStage(ctx); got != StageFresh {
		t.Errorf("expected fresh, got %s", got)
	}
}

func TestDetectStage_Configured(t *testing.T) {
	ctx := StageContext{HasAPIKey: true}
	if got := DetectStage(ctx); got != StageConfigured {
		t.Errorf("expected configured, got %s", got)
	}
}

func TestDetectStage_AIReady(t *testing.T) {
	ctx := StageContext{HasAPIKey: true, HasAIKey: true}
	if got := DetectStage(ctx); got != StageAIReady {
		t.Errorf("expected ai_ready, got %s", got)
	}
}

func TestDetectStage_Equipped(t *testing.T) {
	ctx := StageContext{HasAPIKey: true, HasAIKey: true, MCPCount: 2}
	if got := DetectStage(ctx); got != StageEquipped {
		t.Errorf("expected equipped, got %s", got)
	}
}

func TestDetectStage_Trading(t *testing.T) {
	ctx := StageContext{HasAPIKey: true, HasAIKey: true, MCPCount: 2, TradeCount: 5}
	if got := DetectStage(ctx); got != StageTrading {
		t.Errorf("expected trading, got %s", got)
	}
}

func TestDetectStage_Analyzing(t *testing.T) {
	ctx := StageContext{HasAPIKey: true, HasAIKey: true, MCPCount: 2, TradeCount: 5, HasAnalyzed: true}
	if got := DetectStage(ctx); got != StageAnalyzing {
		t.Errorf("expected analyzing, got %s", got)
	}
}

func TestDetectStage_Advanced(t *testing.T) {
	ctx := StageContext{HasAPIKey: true, HasAIKey: true, MCPCount: 2, TradeCount: 5, HasAnalyzed: true, HasBacktested: true}
	if got := DetectStage(ctx); got != StageAdvanced {
		t.Errorf("expected advanced, got %s", got)
	}
}

func TestActionsForStage_ReturnsCards(t *testing.T) {
	stages := []Stage{StageFresh, StageConfigured, StageAIReady, StageEquipped, StageTrading, StageAnalyzing, StageAdvanced}
	for _, s := range stages {
		cards := ActionsForStage(s, StageContext{TopPositions: []string{"BTC"}})
		if len(cards) == 0 {
			t.Errorf("stage %s returned no action cards", s)
		}
		for _, c := range cards {
			if c.Title == "" || c.Command == "" {
				t.Errorf("stage %s has card with empty title or command", s)
			}
		}
	}
}

func TestAdvancedActions_ZeroPortfolio(t *testing.T) {
	// PortfolioValue=0 should NOT produce a div-by-zero or "Inf%" card.
	ctx := StageContext{
		HasAPIKey: true, HasAIKey: true, MCPCount: 2,
		TradeCount: 10, HasAnalyzed: true, HasBacktested: true,
		CashBalance: 50000, PortfolioValue: 0,
	}
	cards := ActionsForStage(StageAdvanced, ctx)
	for _, c := range cards {
		if strings.Contains(c.Title, "Inf") || strings.Contains(c.Title, "NaN") {
			t.Errorf("advanced stage produced bad card title: %s", c.Title)
		}
	}
}

func TestAdvancedActions_EmptyPositions(t *testing.T) {
	ctx := StageContext{
		HasAPIKey: true, HasAIKey: true, MCPCount: 2,
		TradeCount: 10, HasAnalyzed: true, HasBacktested: true,
	}
	cards := ActionsForStage(StageAdvanced, ctx)
	if len(cards) == 0 {
		t.Error("expected at least one action card for advanced stage with empty positions")
	}
}

func TestNextStepAfterCommand(t *testing.T) {
	ctx := StageContext{TradeCount: 3, HasAnalyzed: true, TopPositions: []string{"BTC"}}

	// Test that every known command returns hints.
	commandsWithHints := []string{
		"price", "market", "watch", "chart",
		"analyze", "consensus", "analytics",
		"buy", "sell", "orders", "pnl", "history", "status", "snapshot", "balances",
		"backtest", "auto", "trigger", "alert", "strategy", "risk",
		"stock", "screen", "bet", "polymarket", "odds", "swap", "gas", "funding", "wallet",
		"mcp", "connect", "config", "model", "memory", "dashboard",
	}
	for _, cmd := range commandsWithHints {
		hints := NextStepAfterCommand(cmd, ctx)
		if len(hints) == 0 {
			t.Errorf("expected hints after %q, got none", cmd)
		}
	}

	// Unknown command returns nil.
	hints := NextStepAfterCommand("unknown_cmd", ctx)
	if hints != nil {
		t.Errorf("expected nil for unknown command, got %v", hints)
	}

	// First-trade user gets different hints after buy.
	newCtx := StageContext{TradeCount: 0}
	hints = NextStepAfterCommand("buy", newCtx)
	if len(hints) == 0 {
		t.Error("expected hints for first trade buy")
	}

	// No AI key: config should suggest setting it.
	noAICtx := StageContext{HasAPIKey: true}
	hints = NextStepAfterCommand("config", noAICtx)
	if len(hints) == 0 || !strings.Contains(hints[0], "anthropic_key") {
		t.Errorf("expected anthropic_key hint for config without AI key, got %v", hints)
	}
}

func TestStageOrdinalAndLabel(t *testing.T) {
	if got := StageOrdinal(StageFresh); got != 1 {
		t.Fatalf("StageOrdinal(fresh) = %d, want 1", got)
	}
	if got := StageOrdinal(StageAdvanced); got != 7 {
		t.Fatalf("StageOrdinal(advanced) = %d, want 7", got)
	}
	if got := StageLabel(StageAIReady); got == "" {
		t.Fatal("StageLabel(ai_ready) should not be empty")
	}
}

func TestOnboardingChecklistAndProgress(t *testing.T) {
	ctx := StageContext{
		HasAPIKey:     true,
		HasAIKey:      true,
		MCPCount:      1,
		TradeCount:    2,
		HasAnalyzed:   true,
		HasBacktested: false,
	}
	items := OnboardingChecklist(ctx)
	if len(items) != 6 {
		t.Fatalf("expected 6 checklist items, got %d", len(items))
	}
	done, total := JourneyProgress(ctx)
	if total != 6 {
		t.Fatalf("expected total=6, got %d", total)
	}
	if done != 5 {
		t.Fatalf("expected done=5, got %d", done)
	}
}
