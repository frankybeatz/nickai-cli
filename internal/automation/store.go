package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RuleType identifies what drives the automation.
type RuleType string

const (
	RuleSchedule  RuleType = "schedule"
	RuleCondition RuleType = "condition"
	RulePortfolio RuleType = "portfolio"
	RuleIndicator RuleType = "indicator"
)

// AutoRule is a single automation rule.
type AutoRule struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Type        RuleType  `json:"type"`

	// Schedule fields.
	Schedule    string `json:"schedule,omitempty"`
	IntervalSec int    `json:"interval_sec,omitempty"`

	// Condition fields.
	Symbol   string  `json:"symbol,omitempty"`
	Operator string  `json:"operator,omitempty"`
	Target   float64 `json:"target,omitempty"`

	// Portfolio fields.
	MetricName string  `json:"metric_name,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`

	// Indicator-based conditions (from backtest strategy activation).
	IndicatorConditions []IndicatorCondition `json:"indicator_conditions,omitempty"`

	// Source strategy name (if activated from a backtest).
	SourceStrategy string `json:"source_strategy,omitempty"`

	// Action fields.
	Action       string  `json:"action"`
	ActionSymbol string  `json:"action_symbol"`
	ActionValue  float64 `json:"action_value"`
	ActionType   string  `json:"action_type"`

	// State.
	Status    string    `json:"status"`
	LastFired time.Time `json:"last_fired,omitempty"`
	FireCount int       `json:"fire_count"`
	MaxFires  int       `json:"max_fires,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	NextCheck time.Time `json:"next_check,omitempty"`
}

// IndicatorCondition is a single indicator-based rule for live strategy monitoring.
type IndicatorCondition struct {
	Indicator string  `json:"indicator"` // rsi, macd, macd_histogram, etc.
	Operator  string  `json:"operator"`  // <, >, crosses_above, crosses_below
	Value     float64 `json:"value"`
}

// IndicatorSnapshot holds computed indicator values for a symbol.
type IndicatorSnapshot struct {
	RSI            float64
	MACD           float64
	MACDSignal     float64
	MACDHistogram  float64
	BollingerUpper float64
	BollingerLower float64
	SMA20          float64
	SMA50          float64
	EMA12          float64
	EMA26          float64
	Price          float64
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "automations.json"), nil
}

// Load reads all automation rules from disk.
func Load() ([]AutoRule, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rules []AutoRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// Save writes all automation rules to disk.
func Save(rules []AutoRule) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Add appends a new rule and saves.
func Add(rule AutoRule) error {
	rules, _ := Load()
	rules = append(rules, rule)
	return Save(rules)
}

// Active returns only active rules.
func Active() ([]AutoRule, error) {
	all, err := Load()
	if err != nil {
		return nil, err
	}
	var active []AutoRule
	for _, r := range all {
		if r.Status == "active" {
			active = append(active, r)
		}
	}
	return active, nil
}

// Pause sets a rule to paused by ID prefix match.
func Pause(idPrefix string) error {
	rules, err := Load()
	if err != nil {
		return err
	}
	found := false
	for i := range rules {
		if strings.HasPrefix(rules[i].ID, idPrefix) {
			rules[i].Status = "paused"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no rule found with ID prefix: %s", idPrefix)
	}
	return Save(rules)
}

// Resume sets a paused rule back to active by ID prefix match.
func Resume(idPrefix string) error {
	rules, err := Load()
	if err != nil {
		return err
	}
	found := false
	for i := range rules {
		if strings.HasPrefix(rules[i].ID, idPrefix) {
			rules[i].Status = "active"
			rules[i].NextCheck = time.Now()
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no rule found with ID prefix: %s", idPrefix)
	}
	return Save(rules)
}

// Remove deletes a rule by ID prefix match.
func Remove(idPrefix string) error {
	rules, err := Load()
	if err != nil {
		return err
	}
	filtered := rules[:0]
	found := false
	for _, r := range rules {
		if !found && strings.HasPrefix(r.ID, idPrefix) {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("no rule found with ID prefix: %s", idPrefix)
	}
	return Save(filtered)
}

// MarkFired increments the fire count, updates last-fired and next-check.
func MarkFired(idPrefix string) error {
	rules, err := Load()
	if err != nil {
		return err
	}
	for i := range rules {
		if strings.HasPrefix(rules[i].ID, idPrefix) {
			rules[i].FireCount++
			rules[i].LastFired = time.Now()
			if rules[i].IntervalSec > 0 {
				rules[i].NextCheck = time.Now().Add(time.Duration(rules[i].IntervalSec) * time.Second)
			}
			// Check max fires.
			if rules[i].MaxFires > 0 && rules[i].FireCount >= rules[i].MaxFires {
				rules[i].Status = "completed"
			}
			break
		}
	}
	return Save(rules)
}

// ParseSchedule converts a human-friendly schedule string to interval seconds.
func ParseSchedule(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "hourly":
		return 3600, nil
	case "daily":
		return 86400, nil
	case "weekly":
		return 604800, nil
	}

	// Handle "weekly:monday" etc.
	if strings.HasPrefix(s, "weekly:") {
		return 604800, nil
	}

	// Handle "every Xh", "every Xm", "Xh", "Xm".
	s = strings.TrimPrefix(s, "every ")
	s = strings.TrimSpace(s)

	if strings.HasSuffix(s, "h") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("invalid schedule: %s", s)
		}
		return int(v * 3600), nil
	}
	if strings.HasSuffix(s, "m") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("invalid schedule: %s", s)
		}
		return int(v * 60), nil
	}
	if strings.HasSuffix(s, "s") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("invalid schedule: %s", s)
		}
		return int(v), nil
	}

	return 0, fmt.Errorf("unrecognized schedule format: %s (try: hourly, daily, weekly, 4h, 30m)", s)
}

// GetIndicatorValue returns a named indicator value from a snapshot.
func GetIndicatorValue(indicator string, snap IndicatorSnapshot) float64 {
	switch strings.ToLower(indicator) {
	case "rsi":
		return snap.RSI
	case "macd":
		return snap.MACD
	case "macd_signal":
		return snap.MACDSignal
	case "macd_histogram":
		return snap.MACDHistogram
	case "bollinger_upper":
		return snap.BollingerUpper
	case "bollinger_lower":
		return snap.BollingerLower
	case "sma20":
		return snap.SMA20
	case "sma50":
		return snap.SMA50
	case "ema12":
		return snap.EMA12
	case "ema26":
		return snap.EMA26
	case "price":
		return snap.Price
	default:
		return 0
	}
}

// EvalIndicatorConditions checks if all indicator conditions are met.
func EvalIndicatorConditions(conditions []IndicatorCondition, snap IndicatorSnapshot, prevSnap *IndicatorSnapshot) bool {
	for _, cond := range conditions {
		current := GetIndicatorValue(cond.Indicator, snap)
		switch cond.Operator {
		case "<":
			if !(current < cond.Value) {
				return false
			}
		case ">":
			if !(current > cond.Value) {
				return false
			}
		case "crosses_above":
			if prevSnap == nil {
				return false
			}
			prev := GetIndicatorValue(cond.Indicator, *prevSnap)
			if !(prev <= cond.Value && current > cond.Value) {
				return false
			}
		case "crosses_below":
			if prevSnap == nil {
				return false
			}
			prev := GetIndicatorValue(cond.Indicator, *prevSnap)
			if !(prev >= cond.Value && current < cond.Value) {
				return false
			}
		}
	}
	return true
}
