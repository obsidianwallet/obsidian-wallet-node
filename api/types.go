package api

import (
	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
	"github.com/obsidianwallet/obsidian-wallet-node/walletmanager"
)

type APIService struct {
	address   string
	incclient *incclient.IncClient
	wlm       *walletmanager.WalletManager
}

type NetworkController interface {
	GetCurrentNetwork() string
	GetNetworkList() []common.NetworkID
	AddNetwork(networkID common.NetworkID) error
	SwitchNetwork(network string) error
}
