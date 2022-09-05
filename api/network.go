package api

import (
	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
)

func (api *APIService) Stop() error {
	return nil
}
func (api *APIService) Start() error {
	return nil
}
func (api *APIService) SwitchNetwork(networkParam common.NetworkID, incclient *incclient.IncClient) error {
	api.incclient = incclient
	return nil
}
