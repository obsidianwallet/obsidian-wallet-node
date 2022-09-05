package walletmanager

import (
	"time"

	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
)

func (wlm *WalletManager) Stop() error {
	return nil
}
func (wlm *WalletManager) Start() error {
	return nil
}

func (wlm *WalletManager) SwitchNetwork(networkParam common.NetworkID, incclient *incclient.IncClient) error {
	wlm.networkLock.Lock()
	defer wlm.networkLock.Unlock()
	// stop sync coins
	wlm.coinsyncmng.stop()

	// stop scan coins
	for _, accountRT := range wlm.accounts {
		accountRT.stop()
	}

	//re-initialized coinsyncmng
	coinSyncMng := CoinSyncManager{
		currentNetwork:   networkParam,
		wlm:              wlm,
		currentSyncShard: make(map[int]bool),
		currentSyncState: make(map[int]map[string]uint64),
		chainCoinState:   make(map[int]map[string]uint64),
		stopCh:           make(chan struct{}),
	}
	wlm.coinsyncmng = &coinSyncMng

	//re-initialized account
	for _, accountRT := range wlm.accounts {
		err := accountRT.start()
		if err != nil {
			return err
		}
	}

	return nil
}

func (csm *CoinSyncManager) stop() {
	close(csm.stopCh)
	for {
		isStopped := true
		for _, v := range csm.currentSyncShard {
			if v {
				isStopped = false
			}
		}
		if isStopped {
			break
		}
		time.Sleep(time.Second)
	}
}

func (wlm *WalletManager) GetCurrentNetwork() common.NetworkID {
	return wlm.currentNetwork
}
