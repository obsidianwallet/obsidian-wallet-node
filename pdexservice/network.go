package pdexservice

import (
	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
)

func (pdexServ *PDexService) Stop() error {
	return nil
}
func (pdexServ *PDexService) Start() error {
	return nil
}
func (pdexServ *PDexService) SwitchNetwork(networkParam common.NetworkID, incclient *incclient.IncClient) error {
	pdexServ.incclient = incclient
	return nil
}
