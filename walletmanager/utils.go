package walletmanager

import (
	"math/big"
)

func getAddressShardID(pubkey []byte, totalShard int) int {
	b := pubkey[len(pubkey)-1]
	shardid := int(b) % totalShard
	return shardid
}

func buildCoinIdxList(from uint64, to uint64) []uint64 {
	var idxList []uint64
	for i := from; i <= to; i++ {
		idxList = append(idxList, i)
	}
	return idxList
}

func buildCoinDBKeys(shardID byte, tokenID string, idx uint64, pubkey []byte) (refKeys [][]byte, err error) {

	idxKey := buildCoinIdxKey(shardID, tokenID, idx)

	refKeys = append(refKeys, idxKey)
	return
}

func buildCoinIdxKey(shardID byte, tokenID string, idx uint64) (key []byte) {
	idxBig := big.Int{}
	idxBig.SetUint64(idx)
	key = append(key, shardID)
	key = append(key, []byte(tokenID)...)
	key = append(key, idxBig.Bytes()...)
	return
}

func buildCoinOwnerKeys(shardID byte, tokenID string, idx uint64, pubkey []byte) (refKeys [][]byte, err error) {

	idxKey := buildCoinIdxKey(shardID, tokenID, idx)

	refKeys = append(refKeys, idxKey)
	return
}
