package common

type Config struct {
	ServingAddress string
	UseNetwork     string
	Networks       []NetworkID
}

type NetworkID struct {
	Name        string
	RPCs        []string
	ServiceURLs []string
}

type CoinOwnerData struct {
	Pubkey   string
	Keyimage string
	Value    uint64
	Rk       string
}
