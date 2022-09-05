package walletmanager

import "time"

const (
	dbAccountInfoPrefix   = "wlmacc-info-"
	dbAccountDataPrefix   = "wlmacc-data-"
	dbCoinDataPrefix      = "coin-"
	dbSyncStateDataPrefix = "sync-state-"
)

const (
	maxRetrieveCoins = 1000
)

const (
	scanCoinsInterval = 15 * time.Second
)
