package ai

import (
	"testing"
)

func TestParseVerdict_ValidJSON(t *testing.T) {
	input := `{"verdict": "BUY", "confidence": "High", "reasoning": "Strong upward momentum with RSI oversold."}`

	verdict, confidence, reasoning := parseVerdict(input)

	if verdict != "BUY" {
		t.Errorf("verdict: got %q, want %q", verdict, "BUY")
	}
	if confidence != "High" {
		t.Errorf("confidence: got %q, want %q", confidence, "High")
	}
	if reasoning != "Strong upward momentum with RSI oversold." {
		t.Errorf("reasoning: got %q, want expected string", reasoning)
	}
}

func TestParseVerdict_MessyJSON(t *testing.T) {
	// Model wraps JSON in markdown code fences.
	input := "Here is my analysis:\n```json\n{\"verdict\": \"SELL\", \"confidence\": \"Medium\", \"reasoning\": \"Bearish divergence on MACD.\"}\n```"

	verdict, confidence, reasoning := parseVerdict(input)

	if verdict != "SELL" {
		t.Errorf("verdict: got %q, want %q", verdict, "SELL")
	}
	if confidence != "Medium" {
		t.Errorf("confidence: got %q, want %q", confidence, "Medium")
	}
	if reasoning != "Bearish divergence on MACD." {
		t.Errorf("reasoning: got %q, want expected string", reasoning)
	}
}

func TestParseVerdict_PlainText(t *testing.T) {
	input := "Based on the data, I recommend a HOLD. Confidence is Low because the market is uncertain."

	verdict, confidence, _ := parseVerdict(input)

	if verdict != "HOLD" {
		t.Errorf("verdict: got %q, want %q", verdict, "HOLD")
	}
	if confidence != "Low" {
		t.Errorf("confidence: got %q, want %q", confidence, "Low")
	}
}

func TestComputeConsensus_AllBuy(t *testing.T) {
	verdicts := []ModelVerdict{
		{Model: "m1", Verdict: "BUY"},
		{Model: "m2", Verdict: "BUY"},
		{Model: "m3", Verdict: "BUY"},
	}

	consensus, agreement := computeConsensus(verdicts, 0.67)

	if consensus != "BUY" {
		t.Errorf("consensus: got %q, want %q", consensus, "BUY")
	}
	if agreement != "3/3" {
		t.Errorf("agreement: got %q, want %q", agreement, "3/3")
	}
}

func TestComputeConsensus_TwoBuyOneSell(t *testing.T) {
	verdicts := []ModelVerdict{
		{Model: "m1", Verdict: "BUY"},
		{Model: "m2", Verdict: "BUY"},
		{Model: "m3", Verdict: "SELL"},
	}

	consensus, agreement := computeConsensus(verdicts, 0.67)

	if consensus != "BUY" {
		t.Errorf("consensus: got %q, want %q", consensus, "BUY")
	}
	if agreement != "2/3" {
		t.Errorf("agreement: got %q, want %q", agreement, "2/3")
	}
}

func TestComputeConsensus_NoConsensus(t *testing.T) {
	verdicts := []ModelVerdict{
		{Model: "m1", Verdict: "BUY"},
		{Model: "m2", Verdict: "SELL"},
		{Model: "m3", Verdict: "HOLD"},
	}

	consensus, agreement := computeConsensus(verdicts, 0.67)

	if consensus != "NO_CONSENSUS" {
		t.Errorf("consensus: got %q, want %q", consensus, "NO_CONSENSUS")
	}
	if agreement != "1/3" {
		t.Errorf("agreement: got %q, want %q", agreement, "1/3")
	}
}

func TestComputeConsensus_TwoHoldOneBuy(t *testing.T) {
	verdicts := []ModelVerdict{
		{Model: "m1", Verdict: "HOLD"},
		{Model: "m2", Verdict: "HOLD"},
		{Model: "m3", Verdict: "BUY"},
	}

	consensus, agreement := computeConsensus(verdicts, 0.67)

	if consensus != "HOLD" {
		t.Errorf("consensus: got %q, want %q", consensus, "HOLD")
	}
	if agreement != "2/3" {
		t.Errorf("agreement: got %q, want %q", agreement, "2/3")
	}
}

func TestComputeConsensus_WithErrors(t *testing.T) {
	verdicts := []ModelVerdict{
		{Model: "m1", Verdict: "BUY"},
		{Model: "m2", Verdict: "BUY"},
		{Model: "m3", Error: "API timeout"},
	}

	consensus, agreement := computeConsensus(verdicts, 0.67)

	if consensus != "BUY" {
		t.Errorf("consensus: got %q, want %q", consensus, "BUY")
	}
	if agreement != "2/2" {
		t.Errorf("agreement: got %q, want %q", agreement, "2/2")
	}
}
