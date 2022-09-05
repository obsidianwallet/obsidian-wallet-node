package common

var DefaultConfig = Config{
	ServingAddress: "0.0.0.0:8989",
	Networks:       []NetworkID{MainnetID},
}

var MainnetID = NetworkID{
	Name:        "mainnet",
	RPCs:        []string{"https://lb-fullnode.incognito.org/fullnode"},
	ServiceURLs: []string{"https://api-coinservice.incognito.org"},
}
