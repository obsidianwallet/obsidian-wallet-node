package walletmanager

import (
	"sync"
	"time"

	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/incognitochain/go-incognito-sdk-v2/wallet"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
	"github.com/obsidianwallet/obsidian-wallet-node/database"
)

type WalletManager struct {
	networkLock    sync.RWMutex
	currentNetwork common.NetworkID
	incclient      *incclient.IncClient
	db             *database.Database

	lock     sync.RWMutex
	accounts map[string]*RuntimeAccount

	coinsyncmng *CoinSyncManager
}

type AccountType int

const (
	Masterless AccountType = iota
	WatchOnly
)

type Account struct {
	Name           string
	Note           string
	Type           AccountType
	PrivateKey     string
	PaymentAddress string
	OTAKey         string
	ViewKey        string
	IsEncrypted    bool
	WatchTokens    map[string]struct{}
}

type RuntimeAccount struct {
	account Account
	wlk     *wallet.KeyWallet

	lock      sync.RWMutex
	coinstate AccountCoinState

	currentNetwork common.NetworkID

	wlm       *WalletManager
	stopCh    chan struct{}
	isStopped bool
}

type AccountCoinState struct {
	scannedCoinIndex map[string]uint64
	PRVUTXOList      []string
	TokenUTXOList    map[string][]string
}

type CoinSyncManager struct {
	currentNetwork common.NetworkID
	wlm            *WalletManager

	currentSyncShard     map[int]bool
	currentSyncState     map[int]map[string]uint64
	chainCoinState       map[int]map[string]uint64
	lastChainStateUpdate time.Time
	lock                 sync.RWMutex

	stopCh chan struct{}
}

type Contact struct {
	Name    string
	Address string
	Note    string
}
