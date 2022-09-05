package walletmanager

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/incognitochain/go-incognito-sdk-v2/coin"
	"github.com/incognitochain/go-incognito-sdk-v2/common"
	"github.com/incognitochain/go-incognito-sdk-v2/common/base58"
	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	wcommon "github.com/obsidianwallet/obsidian-wallet-node/common"
	"github.com/obsidianwallet/obsidian-wallet-node/database"
)

func (rtacc *RuntimeAccount) scanCoins() {
	for {
		select {
		case <-rtacc.stopCh:
			rtacc.isStopped = true
			return
		default:
			rtacc.isStopped = false
			shardID := getAddressShardID(rtacc.wlk.KeySet.PaymentAddress.Pk[:], 8)
			var wg sync.WaitGroup
			for tokenID, currentIndex := range rtacc.coinstate.scannedCoinIndex {
				wg.Add(1)
				go func(tkID string, cIdx uint64) {
					defer func() {
						wg.Done()

					}()
					nextIndex := cIdx + 100
					coinPubkeyList, err := rtacc.wlm.coinsyncmng.GetCoinPubkeyByIndices(shardID, tkID, cIdx, nextIndex)
					if err != nil {
						log.Fatalln(err)
					}
					coinList, err := rtacc.wlm.coinsyncmng.GetCoinByPubkey(coinPubkeyList)
					if err != nil {
						log.Fatalln(err)
					}
					coinOwnerData, err := rtacc.checkCoinOwner(coinList)
					if err != nil {
						log.Fatalln(err)
					}
					_ = coinOwnerData
				}(tokenID, currentIndex)
			}

			time.Sleep(scanCoinsInterval)
		}
	}
}

func (rtacc *RuntimeAccount) checkBalance() {
	for {
		rtacc.lock.Lock()
		shardID := getAddressShardID(rtacc.wlk.KeySet.PaymentAddress.Pk[:], 8)

		newPRVList, err := checkKeyImage(byte(shardID), common.PRVCoinID.String(), rtacc.coinstate.PRVUTXOList, rtacc.wlm.incclient)
		if err != nil {
			log.Fatalln(err)
		}
		rtacc.coinstate.PRVUTXOList = newPRVList

		for tokenID, keyimageList := range rtacc.coinstate.TokenUTXOList {
			newList, err := checkKeyImage(byte(shardID), common.ConfidentialAssetID.String(), keyimageList, rtacc.wlm.incclient)
			if err != nil {
				log.Fatalln(err)
			}
			rtacc.coinstate.TokenUTXOList[tokenID] = newList
		}

		rtacc.lock.Unlock()
		if err = rtacc.saveAccountInfo(); err != nil {
			log.Fatalln(err)
		}
		time.Sleep(scanCoinsInterval)
	}
}

func checkKeyImage(shardID byte, tokenID string, keyimageList []string, incclient *incclient.IncClient) ([]string, error) {
	var newList []string
	spentList, err := incclient.CheckCoinsSpent(byte(shardID), tokenID, keyimageList)
	if err != nil {
		log.Fatalln(err)
	}
	newlist := []string{}
	for idx, v := range spentList {
		if !v {
			newlist = append(newlist, keyimageList[idx])
		}
	}

	return newList, nil
}

func (rtacc *RuntimeAccount) checkCoinOwner(coinList []coin.CoinV2) ([]wcommon.CoinOwnerData, error) {
	var result []wcommon.CoinOwnerData

	for _, coin := range coinList {
		isOwner, rK := coin.DoesCoinBelongToKeySet(&rtacc.wlk.KeySet)
		if isOwner {
			_, err := coin.Decrypt(&rtacc.wlk.KeySet)
			if err != nil {
				return nil, err
			}
			coinOwnerData := wcommon.CoinOwnerData{
				Pubkey:   coin.GetPublicKey().String(),
				Keyimage: base58.Base58Check{}.Encode(coin.GetKeyImage().ToBytesS(), common.ZeroByte),
				Value:    coin.GetValue(),
				Rk:       rK.String(),
			}
			result = append(result, coinOwnerData)
		}
	}

	return result, nil
}

func (rtacc *RuntimeAccount) loadAccountInfo() error {
	rtacc.lock.Lock()
	defer rtacc.lock.Unlock()
	pubkey, _ := rtacc.wlk.GetPublicKey()
	infoKey := buildAccountInfoKey(rtacc.currentNetwork.Name, pubkey)

	value, err := rtacc.wlm.db.DB.Get([]byte{}, infoKey)
	if err != nil {

		return err
	}
	var coinstate AccountCoinState

	if err := json.Unmarshal(value, &coinstate); err != nil {
		return err
	}

	if len(coinstate.scannedCoinIndex) == 0 {
		coinstate.scannedCoinIndex = make(map[string]uint64)
		coinstate.scannedCoinIndex[common.PRVCoinID.String()] = 0
		coinstate.scannedCoinIndex[common.ConfidentialAssetID.String()] = 0
	}
	if len(coinstate.TokenUTXOList) == 0 {
		coinstate.TokenUTXOList = make(map[string][]string)
	}
	rtacc.coinstate = coinstate
	return nil
}

func (rtacc *RuntimeAccount) saveAccountInfo() error {
	rtacc.wlm.incclient.GetMiningInfo()
	rtacc.lock.Lock()
	defer rtacc.lock.Unlock()
	pubkey, _ := rtacc.wlk.GetPublicKey()
	infoKey := buildAccountInfoKey(rtacc.currentNetwork.Name, pubkey)

	stateBytes, err := json.Marshal(rtacc.coinstate)
	if err != nil {
		return err
	}

	objData := database.Object{
		Key:   infoKey,
		Value: stateBytes,
	}
	rtacc.wlm.db.DB.Set([]byte{}, []database.Object{objData})
	return nil
}

func (rtacc *RuntimeAccount) stop() {
	close(rtacc.stopCh)
	for {
		if rtacc.isStopped {
			break
		}
		time.Sleep(time.Second)
	}
	return
}

func buildAccountInfoKey(networkName string, accountPubkey string) []byte {
	key := []byte{}
	key = append(key, []byte(dbAccountInfoPrefix)...)
	return key
}

func buildAccountDataKey(networkName string, accountPubkey string) []byte {
	key := []byte{}
	key = append(key, []byte(dbAccountDataPrefix)...)
	return key
}

func (rtacc *RuntimeAccount) AddWatchToken(tokenID string) error {
	rtacc.lock.Lock()
	defer rtacc.lock.Unlock()
	rtacc.account.WatchTokens[tokenID] = struct{}{}
	return nil
}

func (rtacc *RuntimeAccount) RemoveWatchToken(tokenID string) error {
	rtacc.lock.Lock()
	defer rtacc.lock.Unlock()
	delete(rtacc.account.WatchTokens, tokenID)
	return nil
}

func (rtacc *RuntimeAccount) start() error {
	return nil
}
