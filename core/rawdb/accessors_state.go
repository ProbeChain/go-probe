// Copyright 2020 The go-probeum Authors
// This file is part of the go-probeum library.
//
// The go-probeum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probeum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probeum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/rlp"
)

// ReadPreimage retrieves a single preimage of the provided hash.
func ReadPreimage(db probedb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(preimageKey(hash))
	return data
}

// WritePreimages writes the provided set of preimages to the database.
func WritePreimages(db probedb.KeyValueWriter, preimages map[common.Hash][]byte) {
	for hash, preimage := range preimages {
		if err := db.Put(preimageKey(hash), preimage); err != nil {
			log.Crit("Failed to store trie preimage", "err", err)
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
	preimageHitCounter.Inc(int64(len(preimages)))
}

// ReadCode retrieves the contract code of the provided code hash.
func ReadCode(db probedb.KeyValueReader, hash common.Hash) []byte {
	// Try with the legacy code scheme first, if not then try with current
	// scheme. Since most of the code will be found with legacy scheme.
	//
	// todo(rjl493456442) change the order when we forcibly upgrade the code
	// scheme with snapshot.
	data, _ := db.Get(hash[:])
	if len(data) != 0 {
		return data
	}
	return ReadCodeWithPrefix(db, hash)
}

// ReadCodeWithPrefix retrieves the contract code of the provided code hash.
// The main difference between this function and ReadCode is this function
// will only check the existence with latest scheme(with prefix).
func ReadCodeWithPrefix(db probedb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(codeKey(hash))
	return data
}

// WriteCode writes the provided contract code database.
func WriteCode(db probedb.KeyValueWriter, hash common.Hash, code []byte) {
	if err := db.Put(codeKey(hash), code); err != nil {
		log.Crit("Failed to store contract code", "err", err)
	}
}

// DeleteCode deletes the specified contract code from the database.
func DeleteCode(db probedb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(codeKey(hash)); err != nil {
		log.Crit("Failed to delete contract code", "err", err)
	}
}

// ReadTrieNode retrieves the trie node of the provided hash.
func ReadTrieNode(db probedb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(hash.Bytes())
	return data
}

// WriteTrieNode writes the provided trie node database.
func WriteTrieNode(db probedb.KeyValueWriter, hash common.Hash, node []byte) {
	if err := db.Put(hash.Bytes(), node); err != nil {
		log.Crit("Failed to store trie node", "err", err)
	}
}

// WriteAlters writes the provided trie node database.
func WriteAlters(db probedb.KeyValueWriter, hash common.Hash, node []byte) {
	if err := db.Put(AlterKey(hash), node); err != nil {
		log.Crit("Failed to store write alters", "err", err)
	}
}

// ReadAlters retrieves the alter hash.
func ReadAlters(db probedb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(AlterKey(hash))
	return data
}

// DelAlters del the provided trie node database.
func DelAlters(db probedb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(AlterKey(hash)); err != nil {
		log.Crit("Failed to delete alters", "err", err)
	}
}

// DeleteTrieNode deletes the specified trie node from the database.
func DeleteTrieNode(db probedb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(hash.Bytes()); err != nil {
		log.Crit("Failed to delete trie node", "err", err)
	}
}

func WriteRootHash(db probedb.KeyValueWriter, hash common.Hash, code []byte) {
	//if err := db.Put(hash.Bytes(), code); err != nil {
	//	log.Crit("Failed to store RootHash", "err", err)
	//}
	if err := db.Put(StateRootKey(hash), code); err != nil {
		log.Crit("Failed to store RootHash", "err", err)
	}
}

func WriteAllStateRootHash1(db probedb.Database, hashes []common.Hash, root common.Hash) {
	blockBatch := db.NewBatch()
	// add trie root
	arrdata, err := rlp.EncodeToBytes(hashes)
	if err != nil {
		log.Crit("Failed to EncodeToBytes", "err", err)
	}
	key := StateRootKey(root)
	//fmt.Printf("WriteAllStateRootHash-key：%v \n", key)
	if err := blockBatch.Put(key, arrdata); err != nil {
		log.Crit("Failed to store RootHash", "err", err)
	}
}

func WriteAllStateRootHash(db probedb.KeyValueWriter, hashes []common.Hash, root common.Hash) {
	// add trie root
	arrdata, err := rlp.EncodeToBytes(hashes)
	if err != nil {
		log.Crit("Failed to EncodeToBytes", "err", err)
	}
	key := StateRootKey(root)
	//fmt.Printf("WriteAllStateRootHash-key：%v \n", key)
	if err := db.Put(key, arrdata); err != nil {
		log.Crit("Failed to store RootHash", "err", err)
	}
}

func ReadRootHash(db probedb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(StateRootKey(hash))
	return data
}
func ReadRootHashForNew(db probedb.KeyValueReader, hash common.Hash) []common.Hash {
	var intarray []common.Hash
	//hash := rawdb.ReadRootHash(db.TrieDB().DiskDB(), root)
	key := StateRootKey(hash)
	//	fmt.Printf("ReadRootHashForNew-key：%v \n", key)
	data, _ := db.Get(key)
	rlp.DecodeBytes(data, &intarray)
	return intarray
}

func WriteDPos(db probedb.KeyValueWriter, dPosNo uint64, list []common.DPoSAccount) {
	// add trie root
	arr, err := rlp.EncodeToBytes(list)
	if err != nil {
		log.Crit("Failed to EncodeToBytes dPos", "err", err)
	}
	key := DposKey(dPosNo)
	if err := db.Put(key, arr); err != nil {
		log.Crit("Failed to store dPos", "err", err)
	}
}

func WriteDPosCandidate(db probedb.KeyValueWriter, list []common.DPoSCandidateAccount) {
	arr, err := rlp.EncodeToBytes(list)
	if err != nil {
		log.Crit("Failed to EncodeToBytes dPos candidate", "err", err)
	}
	if err := db.Put(DPosCandidateKey(), arr); err != nil {
		log.Crit("Failed to store dPos candidate", "err", err)
	}
}

func ReadDPos(db probedb.KeyValueReader, dkey uint64) []common.DPoSAccount {
	var arr []common.DPoSAccount
	key := DposKey(dkey)
	data, _ := db.Get(key)
	err := rlp.DecodeBytes(data, &arr)
	if err != nil {
		log.Warn("Failed to get dPos", "err", err)
	}

	return arr
}

func ReadDPosCandidate(db probedb.KeyValueReader) []common.DPoSCandidateAccount {
	var arr []common.DPoSCandidateAccount
	data, _ := db.Get(DPosCandidateKey())
	err := rlp.DecodeBytes(data, &arr)
	if err != nil {
		log.Warn("Failed to get dPos candidate", "err", err)
	}

	return arr
}
