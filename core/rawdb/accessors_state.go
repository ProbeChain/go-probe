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
