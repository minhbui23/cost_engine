package config

// Config contains all configuration parameters for Payment Engine
type Config struct {
	ApiUrl    string // URL of the API endpoint
	ApiWindow string
	ApiStep   string

	//grpc config
	GrpcAddress string

	//key management
	KeyDirectory string

	//blockchain & transaction para
	ChainID         string // --chain-id
	ProviderAddress string
	KeyringBackend  string  // --keyring-backend
	StakeUnit       string  // Streampay currency (eg: "stake")
	CostToStakeRate float64 // Conversion rate from CostUnit to StakeUnit
	MinStakeAmount  int64   // Minimum stake amount to send (avoid sending 0)

	GasLimit     uint64 // Gas limit for transactions (e.g., 200000)
	GasFeeAmount int64  // Amount for gas fee (e.g., 10)
	GasFeeDenom  string

	DryRun bool // If true, simulate only, do not execute command streampayd tx
}
