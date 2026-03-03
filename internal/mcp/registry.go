package mcp

import "strings"

// TrustTier indicates the verification level of a registry entry.
type TrustTier string

const (
	TierVerified  TrustTier = "verified"  // audited, known-safe
	TierCommunity TrustTier = "community" // popular, open source, unaudited
)

// Capability describes what a server can do.
type Capability string

const (
	CapReadData  Capability = "read-data"
	CapTrade     Capability = "trade"
	CapOnChain   Capability = "on-chain"
	CapAnalytics Capability = "analytics"
)

// RegistryEntry describes a known MCP server in the curated directory.
type RegistryEntry struct {
	Name         string            // short name used with /mcp add
	DisplayName  string            // human-readable name
	Description  string            // one-liner
	Repo         string            // GitHub URL
	Command      string            // command to run
	Args         []string          // default args
	EnvKeys      []string          // required env vars (user must supply)
	EnvHints     map[string]string // human-readable hint per env var
	Tier         TrustTier         // trust level
	Capabilities []Capability      // what it can do
	Tags         []string          // searchable tags
}

// CuratedRegistry is the built-in directory of known MCP servers.
// Entries are ordered by relevance to trading workflows.
var CuratedRegistry = []RegistryEntry{
	// --- Trading & Exchanges ---
	{
		Name:        "ccxt",
		DisplayName: "CCXT Exchange Trading",
		Description: "Trade on 100+ crypto exchanges — Binance, Coinbase, Kraken, and more",
		Repo:        "https://github.com/doggybee/mcp-server-ccxt",
		Command:     "npx",
		Args:        []string{"-y", "mcp-server-ccxt"},
		EnvKeys:     []string{"EXCHANGE_ID", "EXCHANGE_API_KEY", "EXCHANGE_SECRET"},
		EnvHints: map[string]string{
			"EXCHANGE_ID":      "exchange name (binance, coinbase, kraken...)",
			"EXCHANGE_API_KEY": "your exchange API key",
			"EXCHANGE_SECRET":  "your exchange API secret",
		},
		Tier:        TierVerified,
		Capabilities: []Capability{CapReadData, CapTrade},
		Tags:        []string{"trading", "exchange", "crypto", "binance", "coinbase", "spot", "futures"},
	},
	{
		Name:        "alpaca",
		DisplayName: "Alpaca Markets",
		Description: "Trade stocks, ETFs, options, and crypto on Alpaca",
		Repo:        "https://github.com/alpacahq/alpaca-mcp-server",
		Command:     "npx",
		Args:        []string{"-y", "@alpacahq/alpaca-mcp-server"},
		EnvKeys:     []string{"ALPACA_API_KEY", "ALPACA_SECRET_KEY"},
		Tier:        TierVerified,
		Capabilities: []Capability{CapReadData, CapTrade},
		Tags:        []string{"trading", "stocks", "etf", "options", "crypto", "equities"},
	},
	{
		Name:        "binance",
		DisplayName: "Binance",
		Description: "Dedicated Binance trading and market data",
		Repo:        "https://github.com/AnalyticAce/binance-mcp-server",
		Command:     "npx",
		Args:        []string{"-y", "binance-mcp-server"},
		EnvKeys:     []string{"BINANCE_API_KEY", "BINANCE_API_SECRET"},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapTrade},
		Tags:        []string{"trading", "exchange", "binance", "crypto"},
	},
	{
		Name:        "polymarket",
		DisplayName: "Polymarket (The Graph)",
		Description: "Query Polymarket prediction markets — odds, traders, positions via subgraphs",
		Repo:        "https://github.com/PaulieB14/graph-polymarket-mcp",
		Command:     "npx",
		Args:        []string{"-y", "graph-polymarket-mcp"},
		EnvKeys:     []string{},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapAnalytics},
		Tags:        []string{"prediction", "polymarket", "betting", "markets", "odds"},
	},

	{
		Name:        "hyperliquid",
		DisplayName: "Hyperliquid",
		Description: "Hyperliquid perp prices, candles, and L2 order books — no API key",
		Repo:        "https://github.com/mektigboy/server-hyperliquid",
		Command:     "npx",
		Args:        []string{"-y", "@mektigboy/server-hyperliquid"},
		EnvKeys:     []string{},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData},
		Tags:        []string{"hyperliquid", "perps", "futures", "orderbook", "crypto"},
	},

	// --- Blockchain & On-Chain Data ---
	{
		Name:        "onchain",
		DisplayName: "Bankless Onchain",
		Description: "Query on-chain data — ERC20 tokens, transactions, smart contracts",
		Repo:        "https://github.com/bankless/onchain-mcp",
		Command:     "npx",
		Args:        []string{"-y", "@bankless/onchain-mcp"},
		EnvKeys:     []string{},
		Tier:        TierVerified,
		Capabilities: []Capability{CapReadData, CapOnChain},
		Tags:        []string{"blockchain", "ethereum", "onchain", "erc20", "defi"},
	},
	{
		Name:        "solana",
		DisplayName: "Solana Actions",
		Description: "40+ Solana-specific actions — tokens, DeFi, NFTs",
		Repo:        "https://github.com/sendai/solana-mcp",
		Command:     "npx",
		Args:        []string{"-y", "@sendai/solana-mcp"},
		EnvKeys:     []string{"SOLANA_RPC_URL"},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapOnChain, CapTrade},
		Tags:        []string{"solana", "defi", "nft", "spl", "jupiter"},
	},
	{
		Name:        "evm",
		DisplayName: "EVM Blockchain",
		Description: "Access 30+ EVM-compatible blockchains",
		Repo:        "https://github.com/tatum-io/evm-mcp-server",
		Command:     "npx",
		Args:        []string{"-y", "@tatum/evm-mcp-server"},
		EnvKeys:     []string{"TATUM_API_KEY"},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapOnChain},
		Tags:        []string{"evm", "ethereum", "polygon", "arbitrum", "base", "blockchain"},
	},

	// --- DeFi ---
	{
		Name:        "defillama",
		DisplayName: "DeFi Llama",
		Description: "DeFi TVL, yields, DEX volumes, fees, and stablecoin data — no API key",
		Repo:        "https://github.com/nic0xflamel/defillama-mcp-server",
		Command:     "npx",
		Args:        []string{"-y", "@nic0xflamel/defillama-mcp-server"},
		EnvKeys:     []string{},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapAnalytics},
		Tags:        []string{"defi", "tvl", "yields", "dex", "stablecoins", "fees"},
	},
	{
		Name:        "jupiter",
		DisplayName: "Jupiter DEX (Solana)",
		Description: "Solana DeFi trades via Jupiter aggregator",
		Repo:        "https://github.com/jupiter-mcp/jupiter-ultra-mcp",
		Command:     "npx",
		Args:        []string{"-y", "jupiter-ultra-mcp-server"},
		EnvKeys:     []string{"SOLANA_PRIVATE_KEY"},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapTrade, CapOnChain},
		Tags:        []string{"solana", "defi", "dex", "swap", "jupiter"},
	},

	// --- Market Data & Analysis ---
	{
		Name:        "tradingview",
		DisplayName: "TradingView",
		Description: "Technical analysis — indicators, charts, screeners",
		Repo:        "https://github.com/atilaahmettaner/tradingview-mcp",
		Command:     "npx",
		Args:        []string{"-y", "tradingview-mcp-server"},
		EnvKeys:     []string{},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapAnalytics},
		Tags:        []string{"charts", "technical", "indicators", "screener", "analysis"},
	},
	{
		Name:        "coinmarketcap",
		DisplayName: "CoinMarketCap",
		Description: "Crypto prices, rankings, Fear & Greed, and 50+ data tools",
		Repo:        "https://github.com/gcharang/coinmarketcap-mcp",
		Command:     "npx",
		Args:        []string{"-y", "coinmarketcap-mcp"},
		EnvKeys:     []string{"COINMARKETCAP_API_KEY"},
		EnvHints:    map[string]string{"COINMARKETCAP_API_KEY": "free key from coinmarketcap.com/api"},
		Tier:        TierCommunity,
		Capabilities: []Capability{CapReadData, CapAnalytics},
		Tags:        []string{"prices", "rankings", "marketcap", "fear-greed", "crypto", "data"},
	},
	{
		Name:        "brave-search",
		DisplayName: "Brave Search",
		Description: "Web search for news, research, and market sentiment",
		Repo:        "https://github.com/modelcontextprotocol/servers",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
		EnvKeys:     []string{"BRAVE_API_KEY"},
		EnvHints:    map[string]string{"BRAVE_API_KEY": "free key from brave.com/search/api"},
		Tier:        TierVerified,
		Capabilities: []Capability{CapReadData},
		Tags:        []string{"search", "news", "research", "sentiment", "web"},
	},
}

// SearchRegistry returns entries matching the query string against
// name, description, and tags. Empty query returns all entries.
func SearchRegistry(query string) []RegistryEntry {
	if query == "" {
		return CuratedRegistry
	}
	var results []RegistryEntry
	q := strings.ToLower(query)
	for _, entry := range CuratedRegistry {
		if matchesQuery(entry, q) {
			results = append(results, entry)
		}
	}
	return results
}

func matchesQuery(e RegistryEntry, q string) bool {
	if strings.Contains(strings.ToLower(e.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.DisplayName), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Description), q) {
		return true
	}
	for _, tag := range e.Tags {
		if strings.Contains(tag, q) {
			return true
		}
	}
	return false
}

// GetEntry returns a registry entry by name, or nil if not found.
func GetEntry(name string) *RegistryEntry {
	for i := range CuratedRegistry {
		if CuratedRegistry[i].Name == name {
			return &CuratedRegistry[i]
		}
	}
	return nil
}
