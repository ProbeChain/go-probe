package trie

import "github.com/probeum/go-probeum/common"

// WrapDatabase is an intermediate write layer between the trie data structures and
// the disk database.
type WrapDatabase struct {
	*Database
	trie *Trie
}

func NewWrapDatabase(db *Database, trie *Trie) *WrapDatabase {
	return &WrapDatabase{
		Database: db,
		trie:     trie,
	}
}

// Binary 是否是二叉树提交过来的数据
func (db *WrapDatabase) Binary() bool {
	return db.trie.Binary()
}

func (db *WrapDatabase) Origin() *Database {
	return db.Database
}

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (db *WrapDatabase) node(hash common.Hash) node {
	// Retrieve the node from the clean cache if available
	if db.cleans != nil {
		if enc := db.cleans.Get(nil, hash[:]); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			if db.Binary() {
				return mustDecodeBinaryNode(hash[:], enc)
			} else {
				return mustDecodeNode(hash[:], enc)
			}
		}
	}
	// Retrieve the node from the dirty cache if available
	db.lock.RLock()
	dirty := db.dirties[hash]
	db.lock.RUnlock()

	if dirty != nil {
		memcacheDirtyHitMeter.Mark(1)
		memcacheDirtyReadMeter.Mark(int64(dirty.size))
		return dirty.obj(hash)
	}
	memcacheDirtyMissMeter.Mark(1)

	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := db.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		return nil
	}
	if db.cleans != nil {
		db.cleans.Set(hash[:], enc)
		memcacheCleanMissMeter.Mark(1)
		memcacheCleanWriteMeter.Mark(int64(len(enc)))
	}
	if db.Binary() {
		return mustDecodeBinaryNode(hash[:], enc)
	} else {
		return mustDecodeNode(hash[:], enc)
	}
}
