package walletmanager

import (
	"encoding/json"
	"errors"

	"github.com/incognitochain/go-incognito-sdk-v2/common"
	"github.com/incognitochain/go-incognito-sdk-v2/wallet"
	"github.com/obsidianwallet/obsidian-wallet-node/database"
)

func InitWallet(db *database.Database) (*WalletManager, error) {
	coinSyncMng := CoinSyncManager{
		currentSyncShard: make(map[int]bool),
		currentSyncState: make(map[int]map[string]uint64),
		chainCoinState:   make(map[int]map[string]uint64),
		stopCh:           make(chan struct{}),
	}
	wallet := &WalletManager{db: db, accounts: make(map[string]*RuntimeAccount), coinsyncmng: &coinSyncMng}
	coinSyncMng.wlm = wallet
	err := wallet.loadAccounts()
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

func (wlm *WalletManager) addAccount(account Account) (string, error) {
	wlm.lock.Lock()
	defer wlm.lock.Unlock()
	accRT := RuntimeAccount{
		account: account,
	}
	accPubkey := ""
	switch account.Type {
	case Masterless:
		wlk, err := wallet.Base58CheckDeserialize(account.PrivateKey)
		if err != nil {
			return accPubkey, err
		}
		if len(wlk.KeySet.PrivateKey) == 0 {
			return accPubkey, errors.New("invalid key")
		}
		accRT.wlm = wlm
		accRT.wlk = wlk
		accPubkey, err = wlk.GetPublicKey()
		if err != nil {
			return accPubkey, errors.New("invalid key")
		}
	case WatchOnly:

	default:
		return accPubkey, errors.New("invalid wallet type")
	}

	if _, exist := wlm.accounts[accPubkey]; exist {
		return accPubkey, errors.New("account already exists")
	}

	wlm.accounts[accPubkey] = &accRT
	return accPubkey, nil
}

func (wlm *WalletManager) AddNewAccount(account Account) error {
	accPubkey, err := wlm.addAccount(account)
	if err != nil {
		return err
	}
	return wlm.saveAccountToDB(account, accPubkey)
}

func (wlm *WalletManager) GetAccountInstance(account string) *RuntimeAccount {
	wlm.lock.RLock()
	defer wlm.lock.RUnlock()
	acc, exist := wlm.accounts[account]
	if exist {
		return acc
	}
	return nil
}

func (wlm *WalletManager) GetAccountBalance(account string) map[string]uint64 {
	result := make(map[string]uint64)
	return result
}

func (wlm *WalletManager) loadAccounts() error {
	action := func(k []byte, v []byte) (bool, error) {
		var acc Account
		err := json.Unmarshal(v, &acc)
		if err != nil {
			return true, err
		}
		_, err = wlm.addAccount(acc)
		if err != nil {
			return true, err
		}
		return false, nil
	}
	return wlm.db.DB.ReadIteratorNonCopy([]byte(dbAccountInfoPrefix), false, action)
}

func (wlm *WalletManager) saveAccountToDB(account Account, pubkey string) error {
	accountBytes, err := json.Marshal(account)
	if err != nil {
		return err
	}
	dbObj := database.Object{
		Key:   []byte(pubkey),
		Value: accountBytes,
	}
	return wlm.db.DB.Set([]byte(dbAccountInfoPrefix), []database.Object{dbObj})
}

func (wlm *WalletManager) deleteAccountFromDB(pubkey string) error {
	err := wlm.db.DB.Delete([]byte(dbAccountInfoPrefix), []byte(pubkey))
	if err != nil {
		return err
	}
	return nil
}

func (wlm *WalletManager) ListAccounts() ([]Account, error) {
	wlm.lock.RLock()
	defer wlm.lock.RUnlock()
	var accounts []Account
	for _, accountRT := range wlm.accounts {
		accounts = append(accounts, accountRT.account)
	}
	return accounts, nil
}

func (wlm *WalletManager) getLastestShardCoinIndex(shardid int) (map[string]uint64, error) {
	result := make(map[string]uint64)
	prvID := common.PRVCoinID.String()
	pTokenID := common.ConfidentialAssetID.String()

	prvIDIdx, err := wlm.incclient.GetOTACoinLengthByShard(byte(shardid), prvID)
	if err != nil {
		return nil, err
	}
	tkIDIdx, err := wlm.incclient.GetOTACoinLengthByShard(byte(shardid), pTokenID)
	if err != nil {
		return nil, err
	}
	result[prvID] = prvIDIdx
	result[pTokenID] = tkIDIdx
	return result, nil
}
