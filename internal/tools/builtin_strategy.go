package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/memory"
	"github.com/nickai/cli/internal/sanitize"
	"github.com/nickai/cli/internal/strategy"
)

// makeCreateTWAP creates a TWAP strategy and persists it.
func makeCreateTWAP() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Symbol     string  `json:"symbol"`
			Side       string  `json:"side"`
			TotalValue float64 `json:"total_value"`
			Duration   string  `json:"duration"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		symbol := strings.ToUpper(input.Symbol)
		side := strings.ToLower(input.Side)
		if side != "buy" && side != "sell" {
			return ErrorJSON("side must be buy or sell"), nil
		}
		if input.TotalValue <= 0 {
			return ErrorJSON("total_value must be positive"), nil
		}

		dur, err := strategy.ParseDuration(input.Duration)
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}

		sliceCount, intervalSec := strategy.CalcSlices(dur)
		sliceValue := input.TotalValue / float64(sliceCount)

		s := strategy.TWAPStrategy{
			ID:          randomToolID(8),
			Symbol:      symbol,
			Side:        side,
			TotalValue:  input.TotalValue,
			Duration:    input.Duration,
			IntervalSec: intervalSec,
			SliceCount:  sliceCount,
			SliceValue:  sliceValue,
			Executed:    0,
			Status:      "active",
			CreatedAt:   time.Now(),
			NextSliceAt: time.Now().Add(time.Duration(intervalSec) * time.Second),
		}

		if err := strategy.Add(s); err != nil {
			return ErrorJSON("failed to save strategy: " + err.Error()), nil
		}

		return ToJSON(map[string]any{
			"status":      "created",
			"id":          s.ID,
			"symbol":      symbol,
			"side":        side,
			"total_value": fmt.Sprintf("$%.0f", input.TotalValue),
			"duration":    input.Duration,
			"slice_count": sliceCount,
			"slice_value": fmt.Sprintf("$%.2f", sliceValue),
			"interval":    fmt.Sprintf("%dm", intervalSec/60),
			"first_slice": s.NextSliceAt.Format("3:04 PM"),
			"note":        "Strategy is active. Each slice will require confirmation before executing.",
		}), nil
	}
}

// makeCreateAutomation creates an automation rule from AI input.
func makeCreateAutomation() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Description string  `json:"description"`
			Type        string  `json:"type"`
			Schedule    string  `json:"schedule"`
			Symbol      string  `json:"symbol"`
			Operator    string  `json:"operator"`
			Target      float64 `json:"target"`
			MetricName  string  `json:"metric_name"`
			Threshold   float64 `json:"threshold"`
			Action      string  `json:"action"`
			ActionSym   string  `json:"action_symbol"`
			ActionVal   float64 `json:"action_value"`
			ActionType  string  `json:"action_type"`
			MaxFires    int     `json:"max_fires"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		// Sanitize description to prevent prompt injection when displayed in system prompt.
		input.Description = sanitize.SanitizeForPrompt(input.Description)

		ruleType := automation.RuleType(input.Type)
		if ruleType != automation.RuleSchedule && ruleType != automation.RuleCondition && ruleType != automation.RulePortfolio {
			return ErrorJSON("type must be schedule, condition, or portfolio"), nil
		}

		actionType := input.ActionType
		if actionType == "" {
			actionType = "market"
		}

		rule := automation.AutoRule{
			ID:           randomToolID(8),
			Description:  input.Description,
			Type:         ruleType,
			Schedule:     input.Schedule,
			Symbol:       strings.ToUpper(input.Symbol),
			Operator:     input.Operator,
			Target:       input.Target,
			MetricName:   input.MetricName,
			Threshold:    input.Threshold,
			Action:       input.Action,
			ActionSymbol: strings.ToUpper(input.ActionSym),
			ActionValue:  input.ActionVal,
			ActionType:   actionType,
			Status:       "active",
			MaxFires:     input.MaxFires,
			CreatedAt:    time.Now(),
		}

		// Parse schedule if applicable.
		if ruleType == automation.RuleSchedule && input.Schedule != "" {
			intervalSec, err := automation.ParseSchedule(input.Schedule)
			if err != nil {
				return ErrorJSON(err.Error()), nil
			}
			rule.IntervalSec = intervalSec
			rule.NextCheck = time.Now().Add(time.Duration(intervalSec) * time.Second)
		}

		if err := automation.Add(rule); err != nil {
			return ErrorJSON("failed to save automation: " + err.Error()), nil
		}

		return ToJSON(map[string]any{
			"status":      "created",
			"id":          rule.ID,
			"description": rule.Description,
			"type":        rule.Type,
			"action":      fmt.Sprintf("%s %s $%.0f", rule.Action, rule.ActionSymbol, rule.ActionValue),
			"schedule":    rule.Schedule,
			"note":        "Rule is active. Each fire will require user confirmation.",
		}), nil
	}
}

// makeBacktestStrategy creates the backtest executor.
func makeBacktestStrategy() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Preset     string              `json:"preset"`
			Symbol     string              `json:"symbol"`
			Entry      []backtest.Condition `json:"entry_conditions"`
			Exit       []backtest.Condition `json:"exit_conditions"`
			StopLoss   float64             `json:"stop_loss_pct"`
			TakeProfit float64             `json:"take_profit_pct"`
			PosSize    float64             `json:"position_size"`
			Period     string              `json:"period"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		var strat backtest.Strategy

		// If preset specified, load it as base.
		if input.Preset != "" {
			preset := backtest.GetPreset(input.Preset)
			if preset == nil {
				return ErrorJSON("unknown preset: " + input.Preset + ". Available: rsi-reversal, macd-crossover, bollinger-bounce, golden-cross, momentum, fear-and-greed, dip-buyer"), nil
			}
			strat = preset.Strategy
		}

		// Override with provided values.
		if input.Symbol != "" {
			strat.Symbol = strings.ToUpper(input.Symbol)
		}
		if len(input.Entry) > 0 {
			strat.EntryRules = input.Entry
		}
		if len(input.Exit) > 0 {
			strat.ExitRules = input.Exit
		}
		if input.StopLoss > 0 {
			strat.StopLossPct = input.StopLoss
		}
		if input.TakeProfit > 0 {
			strat.TakeProfitPct = input.TakeProfit
		}
		if input.PosSize > 0 {
			strat.PositionSize = input.PosSize
		}
		if input.Period != "" {
			strat.Period = input.Period
		}

		if strat.Symbol == "" {
			return ErrorJSON("symbol is required"), nil
		}
		if strat.Period == "" {
			strat.Period = "180d"
		}

		result, err := backtest.Run(strat)
		if err != nil {
			return ErrorJSON("backtest failed: " + err.Error()), nil
		}

		return ToJSON(result), nil
	}
}

func makeSaveMemory() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Type    string   `json:"type"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		// Sanitize content before persisting to prevent prompt injection on recall.
		input.Content = sanitize.SanitizeForPrompt(input.Content)

		store, _ := memory.Load()
		store.Add(memory.Entry{
			ID:         randomToolID(8),
			Type:       memory.MemoryType(input.Type),
			Content:    input.Content,
			Tags:       input.Tags,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
			Score:      5,
		})
		store.Prune(50)
		if err := store.Save(); err != nil {
			return ErrorJSON("failed to save: " + err.Error()), nil
		}
		return ToJSON(map[string]string{"status": "saved", "content": input.Content}), nil
	}
}

func makeRecallMemory() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		store, _ := memory.Load()
		results := store.Search(input.Query)
		if len(results) == 0 {
			return ToJSON(map[string]string{"status": "no_matches", "message": "No memories matching: " + input.Query}), nil
		}
		_ = store.Save() // persist AccessedAt updates
		return ToJSON(map[string]any{"matches": results, "count": len(results)}), nil
	}
}

func makeActivateStrategy() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			StrategyName string `json:"strategy_name"`
			Symbol       string `json:"symbol"`
			Conditions   []struct {
				Indicator string  `json:"indicator"`
				Operator  string  `json:"operator"`
				Value     float64 `json:"value"`
			} `json:"entry_conditions"`
			Action      string  `json:"action"`
			ActionValue float64 `json:"action_value"`
			MaxFires    int     `json:"max_fires"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		symbol := strings.ToUpper(input.Symbol)
		if input.MaxFires == 0 {
			input.MaxFires = 1
		}

		var conditions []automation.IndicatorCondition
		for _, c := range input.Conditions {
			conditions = append(conditions, automation.IndicatorCondition{
				Indicator: c.Indicator,
				Operator:  c.Operator,
				Value:     c.Value,
			})
		}

		desc := fmt.Sprintf("Live strategy: %s on %s", input.StrategyName, symbol)
		if input.StrategyName == "" {
			desc = fmt.Sprintf("Live monitor: %s %s", input.Action, symbol)
		}

		rule := automation.AutoRule{
			ID:                  randomToolID(8),
			Description:         desc,
			Type:                automation.RuleIndicator,
			Symbol:              symbol,
			IndicatorConditions: conditions,
			SourceStrategy:      input.StrategyName,
			Action:              input.Action,
			ActionSymbol:        symbol,
			ActionValue:         input.ActionValue,
			ActionType:          "market",
			Status:              "active",
			MaxFires:            input.MaxFires,
			CreatedAt:           time.Now(),
		}

		if err := automation.Add(rule); err != nil {
			return ErrorJSON("failed to save: " + err.Error()), nil
		}

		condStrs := make([]string, len(conditions))
		for i, c := range conditions {
			condStrs[i] = fmt.Sprintf("%s %s %.2f", c.Indicator, c.Operator, c.Value)
		}

		return ToJSON(map[string]any{
			"status":     "activated",
			"id":         rule.ID,
			"conditions": condStrs,
			"action":     fmt.Sprintf("%s $%.0f %s when all conditions met", input.Action, input.ActionValue, symbol),
			"note":       "Indicators checked every 60s. Each fire requires confirmation.",
		}), nil
	}
}
