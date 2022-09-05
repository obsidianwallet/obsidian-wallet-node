package walletmanager

import (
	"encoding/json"
	"time"

	"github.com/incognitochain/go-incognito-sdk-v2/coin"
	"github.com/rs/zerolog/log"

	"github.com/obsidianwallet/obsidian-wallet-node/database"
)

func (csm *CoinSyncManager) StartSyncCoinsProcess() error {
	err := csm.loadSyncStates()
	if err != nil {
		return err
	}
	return nil
}

func (csm *CoinSyncManager) SyncShard(shardid int) error {
	csm.lock.Lock()
	defer csm.lock.Unlock()
	if _, ok := csm.currentSyncShard[shardid]; ok {
		return nil
	}
	csm.syncShard(shardid)
	return nil
}

func (csm *CoinSyncManager) updateChainState() error {
	if time.Since(csm.lastChainStateUpdate) < 25*time.Second {
		return nil
	}
	csm.lock.Lock()
	defer csm.lock.Unlock()
	newState := make(map[int]map[string]uint64)
	chainState, err := csm.wlm.incclient.GetOTACoinLength()
	if err != nil {
		return err
	}
	for token, shard := range chainState {
		for shardid, coinIdx := range shard {
			if _, ok := newState[int(shardid)]; !ok {
				newState[int(shardid)] = make(map[string]uint64)
			}
			newState[int(shardid)][token] = coinIdx
		}
	}
	csm.chainCoinState = newState
	return nil
}

func (csm *CoinSyncManager) syncShard(shardid int) {
	csm.currentSyncShard[shardid] = true
	go func() {
		for {
			select {
			case <-csm.stopCh:
				csm.lock.Lock()
				defer csm.lock.Unlock()
				csm.currentSyncShard[shardid] = false
				return
			default:
				err := csm.updateChainState()
				if err != nil {
					log.Fatal().Msg(err.Error())
				}
				csm.lock.RLock()
				sync, ok := csm.currentSyncShard[shardid]
				if !ok || !sync {
					csm.lock.RUnlock()
					return
				}
				csm.lock.RUnlock()

				chainState := csm.chainCoinState[shardid]
				currentState := csm.currentSyncState[shardid]
				for tokenID, chainIdx := range chainState {
					currentIdx, ok := currentState[tokenID]
					if !ok {
						if err := csm.retrieveAndSaveCoins(0, chainIdx, byte(shardid), tokenID); err != nil {
							log.Fatal().Msgf("retrieveAndSaveCoins of shard %d failed, err: %v", shardid, err)
						}
					} else {
						if err := csm.retrieveAndSaveCoins(currentIdx, chainIdx, byte(shardid), tokenID); err != nil {
							log.Fatal().Msgf("retrieveAndSaveCoins of shard %d failed, err: %v", shardid, err)
						}
					}
				}
			}
			time.Sleep(20 * time.Second)
		}
	}()
}

func (csm *CoinSyncManager) retrieveAndSaveCoins(from, to uint64, shardID byte, tokenID string) error {
	start := from
	end := from + maxRetrieveCoins
	if end > to {
		end = to
	}
	for {
		coinIdxs := buildCoinIdxList(start, end)
		coinList, err := csm.wlm.incclient.GetOTACoinsByIndices(shardID, tokenID, coinIdxs)
		if err != nil {
			return err
		}
		listlen := uint64(len(coinList))
		if uint64(listlen) != end-start {
			panic("missing coin list")
		}
		for idx, coin := range coinList {
			coinPubkey := coin.GetPublicKey().ToBytesS()
			refKeys, err := buildCoinDBKeys(shardID, tokenID, idx, coinPubkey)
			if err != nil {
				return err
			}
			dataObj := database.Object{
				Key:   coin.GetPublicKey().ToBytesS(),
				Value: coin.Bytes(),
			}
			refObjs := []database.Object{}
			for _, refKey := range refKeys {
				refObjs = append(refObjs, database.Object{
					Key:   refKey,
					Value: coin.GetPublicKey().ToBytesS(),
				},
				)
			}
			refObjs = append(refObjs, dataObj)
			err = csm.wlm.db.DB.Set([]byte(dbCoinDataPrefix), refObjs)
			if err != nil {
				return err
			}
		}
		err = csm.updateStateSyncState(int(shardID), tokenID, end)
		if err != nil {
			return err
		}
		err = csm.storeSyncState(int(shardID))
		if err != nil {
			return err
		}
		start = end
		if end < to {
			if end+maxRetrieveCoins > to {
				end = to
			} else {
				end += maxRetrieveCoins
			}
		} else {
			break
		}
	}
	return nil
}

func (csm *CoinSyncManager) stopSyncShard(shardid int) error {
	csm.lock.Lock()
	defer csm.lock.Unlock()
	csm.currentSyncShard[shardid] = false
	return nil
}

func (csm *CoinSyncManager) updateStateSyncState(shardid int, tokenID string, idx uint64) error {
	csm.lock.Lock()
	defer csm.lock.Unlock()
	csm.currentSyncState[shardid][tokenID] = idx
	return nil
}

func (csm *CoinSyncManager) storeSyncState(shardid int) error {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	var obj database.Object
	obj.Key = []byte{byte(shardid)}
	dataBytes, err := json.Marshal(csm.currentSyncState[shardid])
	if err != nil {
		return err
	}
	obj.Value = dataBytes
	return csm.wlm.db.DB.Set([]byte(dbSyncStateDataPrefix), []database.Object{obj})
}

func (csm *CoinSyncManager) loadSyncStates() error {
	shardsState := make(map[int]map[string]uint64)
	loadstate := func(k []byte, v []byte) (bool, error) {
		shardID := int(k[0])
		state := make(map[string]uint64)
		err := json.Unmarshal(v, &state)
		if err != nil {
			return true, err
		}
		shardsState[shardID] = state
		return false, nil
	}
	err := csm.wlm.db.DB.ReadIteratorNonCopy([]byte(dbSyncStateDataPrefix), false, loadstate)
	if err != nil {
		return err
	}
	csm.currentSyncState = shardsState
	return nil
}

func (csm *CoinSyncManager) GetCoinPubkeyByIndices(shardid int, tokenID string, from uint64, to uint64) ([][]byte, error) {
	var result [][]byte
	for i := from; i <= to; i++ {
		key := buildCoinIdxKey(byte(shardid), tokenID, i)
		value, err := csm.wlm.db.DB.Get([]byte(dbCoinDataPrefix), key)
		if err != nil {
			log.Printf("coin idx not found: shardid %v tokenid %v from %v to %v i %v err %v", shardid, tokenID, from, to, i, err)
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (csm *CoinSyncManager) GetCoinByPubkey(pubkeys [][]byte) ([]coin.CoinV2, error) {
	var result []coin.CoinV2
	for _, pubkey := range pubkeys {
		value, err := csm.wlm.db.DB.Get([]byte(dbCoinDataPrefix), pubkey)
		if err != nil {
			log.Printf("coin not found: pubkey %v err %v", pubkey, err)
			return nil, err
		}

		outCoin := coin.CoinV2{}
		err = outCoin.SetBytes(value)
		if err != nil {
			return nil, err
		}

		result = append(result, outCoin)
	}
	return result, nil
}
