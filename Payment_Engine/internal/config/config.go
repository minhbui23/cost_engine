package config

// Config contains all configuration parameters for Payment Engine
type Config struct {
	CostFile        string // Path to costs.json file
	StreampaydPath  string // Path to streampayd executable
	ChainID         string // --chain-id
	ProviderAddress string
	KeyringBackend  string  // --keyring-backend
	StreamDuration  string  // --duration for stream-send
	StakeUnit       string  // Streampay currency (eg: "stake")
	CostToStakeRate float64 // Conversion rate from CostUnit to StakeUnit
	MinStakeAmount  int64   // Minimum stake amount to send (avoid sending 0)
	DryRun          bool    // If true, simulate only, do not execute command streampayd tx
}
