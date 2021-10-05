// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/globalconfig"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"math/big"
	"net"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type revision struct {
	id           int
	journalIndex int
}

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

const (
	TRIE_DEPTH = 10
	TRIE_PATH0 = "/stateTrie0"
	TRIE_PATH1 = "/stateTrie1"
	TRIE_PATH2 = "/stateTrie2"
	TRIE_PATH3 = "/stateTrie3"
	TRIE_PATH4 = "/stateTrie4"
	TRIE_PATH5 = "/stateTrie5"
)

type proofList [][]byte

func (n *proofList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

func (n *proofList) Delete(key []byte) error {
	panic("not supported")
}

type TotalTrie struct {
	//trie          Trie // storage trie, which becomes non-nil on first access
	regularTrie       Trie // storage trie, which becomes non-nil on first access
	pnsTrie           Trie // storage trie, which becomes non-nil on first access
	digitalTrie       Trie // storage trie, which becomes non-nil on first access
	contractTrie      Trie // storage trie, which becomes non-nil on first access
	authorizeTrie     Trie // storage trie, which becomes non-nil on first access
	lossTrie          Trie // storage trie, which becomes non-nil on first access
	dPosHash          common.Hash
	dPosCandidateHash common.Hash
}

func (t *TotalTrie) GetKey(key []byte) []byte {
	trieType, err := common.ValidAddress(common.BytesToAddress(key))
	if err != nil {
		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
		return nil
	}
	shaKey := common.ReBuildAddress(key)
	switch trieType {
	case common.ACC_TYPE_OF_GENERAL:
		return t.regularTrie.GetKey(shaKey)
	case common.ACC_TYPE_OF_PNS:
		return t.pnsTrie.GetKey(shaKey)
	case common.ACC_TYPE_OF_ASSET:
		return t.digitalTrie.GetKey(shaKey)
	case common.ACC_TYPE_OF_CONTRACT:
		return t.contractTrie.GetKey(shaKey)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return t.authorizeTrie.GetKey(shaKey)
	case common.ACC_TYPE_OF_LOSE:
		return t.lossTrie.GetKey(shaKey)
	default:
		return nil
	}
}

func (t *TotalTrie) TryGet(key []byte) ([]byte, error) {
	trieType, err := common.ValidAddress(common.BytesToAddress(key))
	if err != nil {
		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
		return nil, err
	}
	newKey := common.ReBuildAddress(key)
	switch trieType {
	case common.ACC_TYPE_OF_GENERAL:
		return t.regularTrie.TryGet(newKey)
	case common.ACC_TYPE_OF_PNS:
		return t.pnsTrie.TryGet(newKey)
	case common.ACC_TYPE_OF_ASSET:
		return t.digitalTrie.TryGet(newKey)
	case common.ACC_TYPE_OF_CONTRACT:
		return t.contractTrie.TryGet(newKey)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return t.authorizeTrie.TryGet(newKey)
	case common.ACC_TYPE_OF_LOSE:
		return t.lossTrie.TryGet(newKey)
	default:
		return nil, fmt.Errorf("trieType no exsist")
	}
}

func (t *TotalTrie) TryUpdate(key, value []byte) error {
	trieType, err := common.ValidAddress(common.BytesToAddress(key))
	if err != nil {
		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
		return err
	}
	newKey := common.ReBuildAddress(key)
	switch trieType {
	case common.ACC_TYPE_OF_GENERAL:
		return t.regularTrie.TryUpdate(newKey, value)
	case common.ACC_TYPE_OF_PNS:
		return t.pnsTrie.TryUpdate(newKey, value)
	case common.ACC_TYPE_OF_ASSET:
		return t.digitalTrie.TryUpdate(newKey, value)
	case common.ACC_TYPE_OF_CONTRACT:
		return t.contractTrie.TryUpdate(newKey, value)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return t.authorizeTrie.TryUpdate(newKey, value)
	case common.ACC_TYPE_OF_LOSE:
		return t.lossTrie.TryUpdate(newKey, value)
	default:
		return fmt.Errorf("trieType no exsist")
	}
}

func (t *TotalTrie) TryDelete(key []byte) error {
	trieType, err := common.ValidAddress(common.BytesToAddress(key))
	if err != nil {
		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
		return err
	}
	newKey := common.ReBuildAddress(key)
	switch trieType {
	case common.ACC_TYPE_OF_GENERAL:
		return t.regularTrie.TryDelete(newKey)
	case common.ACC_TYPE_OF_PNS:
		return t.pnsTrie.TryDelete(newKey)
	case common.ACC_TYPE_OF_ASSET:
		return t.digitalTrie.TryDelete(newKey)
	case common.ACC_TYPE_OF_CONTRACT:
		return t.contractTrie.TryDelete(newKey)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return t.authorizeTrie.TryDelete(newKey)
	case common.ACC_TYPE_OF_LOSE:
		return t.lossTrie.TryDelete(newKey)
	default:
		return fmt.Errorf("trieType no exsist")
	}
}

func (t *TotalTrie) Hash() common.Hash {
	hashes := t.GetTallHash()
	return BuildHash(hashes)
}

func (t *TotalTrie) GetTallHash() []common.Hash {
	hashes := []common.Hash{t.regularTrie.Hash(),
		t.pnsTrie.Hash(),
		t.digitalTrie.Hash(),
		t.contractTrie.Hash(),
		t.authorizeTrie.Hash(),
		t.lossTrie.Hash(),
		t.dPosHash,
		t.dPosCandidateHash,
	}
	return hashes
}

//func (s *StateDB) GetTallHash() []common.Hash {
//	hashes := []common.Hash{s.trie.regularTrie.Hash(),
//		s.trie.pnsTrie.Hash(),
//		s.trie.digitalTrie.Hash(),
//		s.trie.contractTrie.Hash(),
//		s.trie.authorizeTrie.Hash(),
//		s.trie.lossTrie.Hash()}
//	return hashes
//}

func BuildHash(hashes []common.Hash) common.Hash {
	num := big.NewInt(0) // 利用 x ⊕ 0 == x
	for _, hash := range hashes {
		curNum := new(big.Int).SetBytes(crypto.Keccak256(hash.Bytes()))
		num = new(big.Int).Xor(curNum, num)
	}
	hash := make([]byte, 32, 64)        // 哈希出来的长度为32byte
	hash = append(hash, num.Bytes()...) // 前面不足的补0，一共返回32位

	var ret [32]byte
	copy(ret[:], hash[32:64])

	return common.BytesToHash(ret[:])
}

func (t *TotalTrie) Commit(onleaf trie.LeafCallback) (root common.Hash, err error) {
	root0, err0 := t.regularTrie.Commit(onleaf)
	root1, err1 := t.pnsTrie.Commit(onleaf)
	if err1 != nil {
		err0 = err1
	}
	root2, err2 := t.digitalTrie.Commit(onleaf)
	if err2 != nil {
		err0 = err2
	}
	root3, err3 := t.contractTrie.Commit(onleaf)
	if err3 != nil {
		err0 = err3
	}
	root4, err4 := t.authorizeTrie.Commit(onleaf)
	if err4 != nil {
		err0 = err4
	}
	root5, err5 := t.lossTrie.Commit(onleaf)
	if err5 != nil {
		err0 = err5
	}
	root6 := t.dPosHash
	root7 := t.dPosCandidateHash

	hashes := []common.Hash{root0, root1, root2, root3, root4, root5, root6, root7}
	//hashes := []common.Hash{root0, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot}

	return BuildHash(hashes), err0
}

//func (t *TotalTrie) NodeIterator(start []byte) trie.NodeIterator {
//	trieType, err := common.ValidAddress(common.BytesToAddress(start))
//	if err != nil {
//		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
//		return nil
//	}
//	switch trieType {
//	case common.General:
//		return t.regularTrie.NodeIterator(start)
//	case common.Pns:
//		return t.pnsTrie.NodeIterator(start)
//	case common.Asset:
//		return t.digitalTrie.NodeIterator(start)
//	case common.Contract:
//		return t.contractTrie.NodeIterator(start)
//	case common.Authorize:
//		return t.authorizeTrie.NodeIterator(start)
//	case common.Lose:
//		return t.lossTrie.NodeIterator(start)
//	default:
//		return nil
//	}
//}

func (t *TotalTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	trieType, err := common.ValidAddress(common.BytesToAddress(key))
	if err != nil {
		log.Error("Failed to ValidAddress", "trieType", trieType, "err", err)
		return err
	}
	switch trieType {
	case common.ACC_TYPE_OF_GENERAL:
		return t.regularTrie.Prove(key, fromLevel, proofDb)
	case common.ACC_TYPE_OF_PNS:
		return t.pnsTrie.Prove(key, fromLevel, proofDb)
	case common.ACC_TYPE_OF_ASSET:
		return t.digitalTrie.Prove(key, fromLevel, proofDb)
	case common.ACC_TYPE_OF_CONTRACT:
		return t.contractTrie.Prove(key, fromLevel, proofDb)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return t.authorizeTrie.Prove(key, fromLevel, proofDb)
	case common.ACC_TYPE_OF_LOSE:
		return t.lossTrie.Prove(key, fromLevel, proofDb)
	default:
		return nil
	}
}

// StateDB structs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db           Database
	prefetcher   *triePrefetcher
	originalRoot common.Hash // The pre-state root, before any changes were made
	//trie         Trie
	trie TotalTrie

	hasher crypto.KeccakState

	snaps         *snapshot.Tree
	snap          snapshot.Snapshot
	snapDestructs map[common.Hash]struct{}
	snapAccounts  map[common.Hash][]byte
	snapStorage   map[common.Hash]map[common.Hash][]byte

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects        map[common.Address]*stateObject
	stateObjectsPending map[common.Address]struct{} // State objects finalized but not yet written to the trie
	stateObjectsDirty   map[common.Address]struct{} // State objects modified in the current execution

	dposList         *dposList
	markLossAccounts map[common.Hash][]common.Address

	/*// DPoSAccount DPoS账户 64
	dPoSAccounts []*common.DPoSAccount
	// DPoSCandidateAccount DPoS候选账户 64
	dPoSCandidateAccounts []*DPoSCandidateAccount

	// DPoSAccount DPoS账户 64
	oldDPoSAccounts []*common.DPoSAccount
	// DPoSCandidateAccount DPoS候选账户 64
	oldDPoSCandidateAccounts []*DPoSCandidateAccount*/

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	thash   common.Hash
	txIndex int
	logs    map[common.Hash][]*types.Log
	logSize uint

	preimages map[common.Hash][]byte

	// Per-transaction access list
	accessList *accessList

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	// Measurements gathered during execution for debugging purposes
	AccountReads         time.Duration
	AccountHashes        time.Duration
	AccountUpdates       time.Duration
	AccountCommits       time.Duration
	StorageReads         time.Duration
	StorageHashes        time.Duration
	StorageUpdates       time.Duration
	StorageCommits       time.Duration
	SnapshotAccountReads time.Duration
	SnapshotStorageReads time.Duration
	SnapshotCommits      time.Duration
}

func GetHash(root common.Hash, db Database) []common.Hash {
	if root == (common.Hash{}) || root == emptyRoot {
		return []common.Hash{emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot}
	}
	hash := rawdb.ReadRootHashForNew(db.TrieDB().DiskDB(), root)
	if hash == nil {
		return []common.Hash{emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot}
	}
	fmt.Println("获取hash：", hash)
	return hash
}

// New creates a new state from a given trie.
func New(root common.Hash, db Database, snaps *snapshot.Tree) (*StateDB, error) {
	// 根据 root 获取六棵树hash数组
	totalTrie, err := OpenTotalTrie(root, db)
	//tr, err := db.OpenTrie(root)
	//fmt.Printf("OpenTrieRoot: %s,isErr:%t\n",root.String(),err != nil)
	if err != nil {
		return nil, err
	}
	sdb := &StateDB{
		db: db,
		//trie:                tr,
		trie:                totalTrie,
		originalRoot:        root,
		snaps:               snaps,
		stateObjects:        make(map[common.Address]*stateObject),
		stateObjectsPending: make(map[common.Address]struct{}),
		stateObjectsDirty:   make(map[common.Address]struct{}),
		logs:                make(map[common.Hash][]*types.Log),
		preimages:           make(map[common.Hash][]byte),
		dposList:            newDposList(),
		journal:             newJournal(),
		accessList:          newAccessList(),
		hasher:              crypto.NewKeccakState(),
	}
	if sdb.snaps != nil {
		if sdb.snap = sdb.snaps.Snapshot(root); sdb.snap != nil {
			sdb.snapDestructs = make(map[common.Hash]struct{})
			sdb.snapAccounts = make(map[common.Hash][]byte)
			sdb.snapStorage = make(map[common.Hash]map[common.Hash][]byte)
		}
	}
	return sdb, nil
}

func OpenTotalTrie(root common.Hash, db Database) (TotalTrie, error) {
	hash := GetHash(root, db)
	trGeneral, err := db.OpenBinTrie(hash[0], globalconfig.DataDir+TRIE_PATH0, TRIE_DEPTH)
	trPns, err1 := db.OpenBinTrie(hash[1], globalconfig.DataDir+TRIE_PATH1, TRIE_DEPTH)
	if err1 != nil {
		err = err1
	}
	trAsset, err2 := db.OpenBinTrie(hash[2], globalconfig.DataDir+TRIE_PATH2, TRIE_DEPTH)
	if err2 != nil {
		err = err2
	}
	trContract, err3 := db.OpenBinTrie(hash[3], globalconfig.DataDir+TRIE_PATH3, TRIE_DEPTH)
	if err3 != nil {
		err = err3
	}
	trAuthorize, err4 := db.OpenBinTrie(hash[4], globalconfig.DataDir+TRIE_PATH4, TRIE_DEPTH)
	if err4 != nil {
		err = err4
	}
	trLose, err5 := db.OpenBinTrie(hash[5], globalconfig.DataDir+TRIE_PATH5, TRIE_DEPTH)
	if err5 != nil {
		err = err5
	}
	totalTrie := TotalTrie{
		regularTrie:       trGeneral,
		pnsTrie:           trPns,
		digitalTrie:       trAsset,
		contractTrie:      trContract,
		authorizeTrie:     trAuthorize,
		lossTrie:          trLose,
		dPosHash:          hash[6],
		dPosCandidateHash: hash[7],
	}
	return totalTrie, err
}

// StartPrefetcher initializes a new trie prefetcher to pull in nodes from the
// state trie concurrently while the state is mutated so that when we reach the
// commit phase, most of the needed data is already hot.
func (s *StateDB) StartPrefetcher(namespace string) {
	if s.prefetcher != nil {
		s.prefetcher.close()
		s.prefetcher = nil
	}
	if s.snap != nil {
		s.prefetcher = newTriePrefetcher(s.db, s.originalRoot, namespace)
	}
}

// StopPrefetcher terminates a running prefetcher and reports any leftover stats
// from the gathered metrics.
func (s *StateDB) StopPrefetcher() {
	if s.prefetcher != nil {
		s.prefetcher.close()
		s.prefetcher = nil
	}
}

// setError remembers the first non-nil error it is called with.
func (s *StateDB) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *StateDB) Error() error {
	return s.dbErr
}

func (s *StateDB) AddLog(log *types.Log) {
	s.journal.append(addLogChange{txhash: s.thash})

	log.TxHash = s.thash
	log.TxIndex = uint(s.txIndex)
	log.Index = s.logSize
	s.logs[s.thash] = append(s.logs[s.thash], log)
	s.logSize++
}

func (s *StateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	logs := s.logs[hash]
	for _, l := range logs {
		l.BlockHash = blockHash
	}
	return logs
}

func (s *StateDB) Logs() []*types.Log {
	var logs []*types.Log
	for _, lgs := range s.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := s.preimages[hash]; !ok {
		s.journal.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		s.preimages[hash] = pi
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (s *StateDB) Preimages() map[common.Hash][]byte {
	return s.preimages
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	s.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	if gas > s.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, s.refund))
	}
	s.refund -= gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (s *StateDB) Exist(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	return so == nil
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}

// GetAccountInfo retrieves the account detail info from the given address or nil if object not found
func (s *StateDB) GetAccountInfo(addr common.Address) interface{} {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.AccountInfo()
	}
	return nil
}

func (s *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// TxIndex returns the current transaction index set by Prepare.
func (s *StateDB) TxIndex() int {
	return s.txIndex
}

func (s *StateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code(s.db)
	}
	return nil
}

func (s *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.CodeSize(s.db)
	}
	return 0
}

func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

// GetState retrieves a value from the given account's storage trie.
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(s.db, hash)
	}
	return common.Hash{}
}

// GetProof returns the Merkle proof for a given account.
func (s *StateDB) GetProof(addr common.Address) ([][]byte, error) {
	return s.GetProofByHash(crypto.Keccak256Hash(addr.Bytes()))
}

// GetProofByHash returns the Merkle proof for a given account.
func (s *StateDB) GetProofByHash(addrHash common.Hash) ([][]byte, error) {
	var proof proofList
	err := s.trie.Prove(addrHash[:], 0, &proof)
	return proof, err
}

// GetStorageProof returns the Merkle proof for given storage slot.
func (s *StateDB) GetStorageProof(a common.Address, key common.Hash) ([][]byte, error) {
	var proof proofList
	trie := s.StorageTrie(a)
	if trie == nil {
		return proof, errors.New("storage trie for requested address does not exist")
	}
	err := trie.Prove(crypto.Keccak256(key.Bytes()), 0, &proof)
	return proof, err
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(s.db, hash)
	}
	return common.Hash{}
}

// Database retrieves the low level database supporting the lower level trie ops.
func (s *StateDB) Database() Database {
	return s.db
}

// StorageTrie returns the storage trie of an account.
// The return value is a copy and is nil for non-existent accounts.
func (s *StateDB) StorageTrie(addr common.Address) Trie {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(s)
	cpy.updateTrie(s.db)
	return cpy.getTrie(s.db)
}

func (s *StateDB) HasSuicided(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (s *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(s.db, key, value)
	}
}

// SetStorage replaces the entire storage for the specified account with given
// storage. This function should only be used for debugging.
func (s *StateDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetStorage(storage)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (s *StateDB) Suicide(addr common.Address) bool {
	obj := s.getStateObject(addr)
	if obj == nil {
		return false
	}
	switch obj.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		s.journal.append(regularSuicideChange{
			account:     &addr,
			suicide:     obj.suicided,
			voteAccount: obj.regularAccount.VoteAccount,
			voteValue:   obj.regularAccount.VoteValue,
			lossType:    obj.regularAccount.LossType,
			value:       obj.regularAccount.Value,
		})
	case common.ACC_TYPE_OF_PNS:
		s.journal.append(pnsSuicideChange{
			account: &addr,
			suicide: obj.suicided,
			pnsType: obj.pnsAccount.Type,
			owner:   obj.pnsAccount.Owner,
			data:    obj.pnsAccount.Data,
		})
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		s.journal.append(assetSuicideChange{
			account:     &addr,
			suicide:     obj.suicided,
			voteAccount: obj.assetAccount.VoteAccount,
			voteValue:   obj.assetAccount.VoteValue,
			assetType:   obj.assetAccount.Type,
			value:       obj.assetAccount.Value,
		})
	case common.ACC_TYPE_OF_AUTHORIZE:
		s.journal.append(authorizeSuicideChange{
			account:     &addr,
			suicide:     obj.suicided,
			owner:       obj.authorizeAccount.Owner,
			pledgeValue: obj.authorizeAccount.PledgeValue,
			voteValue:   obj.authorizeAccount.VoteValue,
			info:        obj.authorizeAccount.Info,
			validPeriod: obj.authorizeAccount.ValidPeriod,
			state:       obj.authorizeAccount.State,
		})
	case common.ACC_TYPE_OF_LOSE:
		s.journal.append(lossSuicideChange{
			account:     &addr,
			suicide:     obj.suicided,
			state:       obj.lossAccount.State,
			lossAccount: obj.lossAccount.LossAccount,
			newAccount:  obj.lossAccount.NewAccount,
			height:      obj.lossAccount.Height,
			infoDigest:  obj.lossAccount.InfoDigest,
		})
	case common.ACC_TYPE_OF_DPOS_CANDIDATE:
		s.journal.append(dPoSCandidateSuicideChange{
			account:       &addr,
			suicide:       obj.suicided,
			enode:         obj.dposCandidateAccount.Enode,
			owner:         obj.dposCandidateAccount.Owner,
			weight:        obj.dposCandidateAccount.Weight,
			delegateValue: obj.dposCandidateAccount.DelegateValue,
		})
	default:
	}
	obj.markSuicided()
	return true
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (s *StateDB) updateStateObject(obj *stateObject) {
	// Track the amount of time wasted on updating the account from the trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.AccountUpdates += time.Since(start) }(time.Now())
	}
	// Encode the account and update the account trie
	addr := obj.Address()

	data, err := rlp.EncodeToBytes(obj)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	if err = s.trie.TryUpdate(addr[:], data); err != nil {
		s.setError(fmt.Errorf("updateStateObject (%x) error: %v", addr[:], err))
	}

	// If state snapshotting is active, cache the data til commit. Note, this
	// update mechanism is not symmetric to the deletion, because whereas it is
	// enough to track account updates at commit time, deletions need tracking
	// at transaction boundary level to ensure we capture state clearing.
	if s.snap != nil {
		//s.snapAccounts[obj.addrHash] = snapshot.SlimAccountRLP(obj.regularAccount.Nonce, obj.regularAccount.Balance, obj.regularAccount.Root, obj.regularAccount.CodeHash)
		s.snapAccounts[obj.addrHash] = snapshot.SlimAccountRLP(obj.Nonce(), obj.Balance(), obj.assetAccount.StorageRoot, obj.assetAccount.CodeHash)
	}
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(obj *stateObject) {
	// Track the amount of time wasted on deleting the account from the trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.AccountUpdates += time.Since(start) }(time.Now())
	}
	// Delete the account from the trie
	addr := obj.Address()
	if err := s.trie.TryDelete(addr[:]); err != nil {
		s.setError(fmt.Errorf("deleteStateObject (%x) error: %v", addr[:], err))
	}
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context. If you need
// to differentiate between non-existent/just-deleted, use getDeletedStateObject.
func (s *StateDB) getStateObject(addr common.Address) *stateObject {
	if obj := s.getDeletedStateObject(addr); obj != nil && !obj.deleted {
		return obj
	}
	return nil
}

// getDeletedStateObject is similar to getStateObject, but instead of returning
// nil for a deleted state object, it returns the actual object with the deleted
// flag set. This is needed by the state journal to revert to the correct s-
// destructed object instead of wiping all knowledge about the state object.
func (s *StateDB) getDeletedStateObject(addr common.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		return obj
	}
	// If no live objects are available, attempt to use snapshots
	var (
		//data *RegularAccount
		err error
	)
	if s.snap != nil {
		//if metrics.EnabledExpensive {
		//	defer func(start time.Time) { s.SnapshotAccountReads += time.Since(start) }(time.Now())
		//}
		//var acc *snapshot.Account
		//if acc, err = s.snap.Account(crypto.HashData(s.hasher, addr.Bytes())); err == nil {
		//	if acc == nil {
		//		return nil
		//	}
		//	data = &RegularAccount{
		//		Nonce:    acc.Nonce,
		//		Balance:  acc.Balance,
		//		CodeHash: acc.CodeHash,
		//		Root:     common.BytesToHash(acc.Root),
		//	}
		//	if len(data.CodeHash) == 0 {
		//		data.CodeHash = emptyCodeHash
		//	}
		//	if data.Root == (common.Hash{}) {
		//		data.Root = emptyRoot
		//	}
		//}
	}
	// If snapshot unavailable or reading from it failed, load from the database
	//if s.snap == nil || err != nil {
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.AccountReads += time.Since(start) }(time.Now())
	}
	enc, err := s.trie.TryGet(addr.Bytes())
	if err != nil {
		s.setError(fmt.Errorf("getDeleteStateObject (%x) error: %v", addr.Bytes(), err))
		return nil
	}
	if enc == nil || len(enc) == 0 {
		return nil
	}

	//data := new(RegularAccount)
	//if err := rlp.DecodeBytes(enc, data); err != nil {
	//	log.Error("Failed to decode state object", "addr", addr, "err", err)
	//	return nil, true
	//}
	////}
	//// Insert into the live set
	//obj := newRegularAccount(s, addr, *data)

	obj, done := s.newAccountDataByAddr(addr, enc)
	if done {
		return obj
	}
	s.setStateObject(obj)
	return obj
}

func (s *StateDB) setStateObject(object *stateObject) {
	/*if obj := s.stateObjects[object.Address()]; obj == nil {
		fmt.Printf("添加账号信息setStateObject，addr:%s,balance:%s,nonce:%d,code:%s,codeHashEmpty:%t\n",
			object.address.String(),object.Balance().String(),object.Nonce(),object.code.String(), bytes.Equal(object.CodeHash(), emptyCodeHash))
	}*/
	s.stateObjects[object.Address()] = object
}

// GetOrNewStateObject retrieves a state object or create a new state object if nil.
func (s *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject, _ = s.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (s *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
	prev = s.getDeletedStateObject(addr) // Note, prev might have been deleted, we need that!

	var prevdestruct bool
	if s.snap != nil && prev != nil {
		_, prevdestruct = s.snapDestructs[prev.addrHash]
		if !prevdestruct {
			s.snapDestructs[prev.addrHash] = struct{}{}
		}
	}
	//newobj = newRegularAccount(s, addr, RegularAccount{})
	newobj, done := s.newAccountDataByAddr(addr, nil)
	if done {
		return newobj, nil
	}
	if prev == nil {
		s.journal.append(createObjectChange{account: &addr})
	} else {
		s.journal.append(resetObjectChange{prev: prev, prevdestruct: prevdestruct})
	}
	s.setStateObject(newobj)
	if prev != nil && !prev.deleted {
		return newobj, prev
	}
	return newobj, nil
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(account ++ (nonce + 1))
//   2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (s *StateDB) CreateAccount(addr common.Address) {
	newObj, prev := s.createObject(addr)
	if prev != nil {
		//newObj.setBalance(prev.regularAccount.Balance)
		newObj.setBalance(prev.Balance())
	}
}

func (s *StateDB) setMarkLossAccount(address common.Address) {
	var arr = s.markLossAccounts[address.Last12BytesToHash()]
	if len(arr) == 0 {
		s.markLossAccounts[address.Last12BytesToHash()] = []common.Address{address}
	} else {
		var exists bool
		for _, addr := range arr {
			if addr == address {
				exists = true
				break
			}
		}
		if !exists {
			s.markLossAccounts[address.Last12BytesToHash()] = append(arr, address)
		}
	}
}

func (s *StateDB) CreateDPoSCandidateAccount(ower common.Address, addr common.Address, jsonData []byte) {
	stateObject := s.getStateObject(addr)
	if nil != stateObject {
		return
	}
	var dposMap map[string]interface{}
	err := json.Unmarshal(jsonData, &dposMap)
	if err != nil {
		fmt.Println(err.Error())
	}
	remoteIp := dposMap["ip"].(string)
	remotePort := dposMap["port"].(string)
	var enode bytes.Buffer
	enode.WriteString("enode://")
	enode.WriteString(ower.String()[2:])
	enode.WriteString("@")
	enode.WriteString(remoteIp)
	enode.WriteString(":")
	enode.WriteString(remotePort)
	stateObject.dposCandidateAccount.Enode = common.BytesToDposEnode([]byte(enode.String()))
	stateObject.dposCandidateAccount.Owner = ower
	stateObject.dposCandidateAccount.Weight = common.InetAtoN(remoteIp)
	s.dposList.dPoSCandidateAccounts.PutOnTop(stateObject.dposCandidateAccount)
}

func InetAtoN(ip string) *big.Int {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret
}

func (s *StateDB) GetMarkLossAccounts(mark common.Hash) []common.Address {
	return s.markLossAccounts[mark]
}

func (s *StateDB) DelMarkLossAccounts(mark common.Hash) {
	delete(s.markLossAccounts, mark)
}

func (db *StateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) error {
	so := db.getStateObject(addr)
	if so == nil {
		return nil
	}
	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))

	for it.Next() {
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		if value, dirty := so.dirtyStorage[key]; dirty {
			if !cb(key, value) {
				return nil
			}
			continue
		}

		if len(it.Value) > 0 {
			_, content, _, err := rlp.Split(it.Value)
			if err != nil {
				return err
			}
			if !cb(key, common.BytesToHash(content)) {
				return nil
			}
		}
	}
	return nil
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (s *StateDB) Copy() *StateDB {
	// Copy all the basic fields, initialize the memory ones
	var regularTrie, pnsTrie, digitalTrie, contractTrie, authorizeTrie, lossTrie Trie
	if s.trie.regularTrie != nil {
		regularTrie = s.db.CopyTrie(s.trie.regularTrie)
	}
	if s.trie.pnsTrie != nil {
		pnsTrie = s.db.CopyTrie(s.trie.pnsTrie)
	}
	if s.trie.digitalTrie != nil {
		digitalTrie = s.db.CopyTrie(s.trie.digitalTrie)
	}
	if s.trie.contractTrie != nil {
		contractTrie = s.db.CopyTrie(s.trie.contractTrie)
	}
	if s.trie.authorizeTrie != nil {
		authorizeTrie = s.db.CopyTrie(s.trie.authorizeTrie)
	}
	if s.trie.lossTrie != nil {
		lossTrie = s.db.CopyTrie(s.trie.lossTrie)
	}
	state := &StateDB{
		db: s.db,
		//trie:                s.db.CopyTrie(s.trie),
		trie: TotalTrie{
			regularTrie:       regularTrie,
			pnsTrie:           pnsTrie,
			digitalTrie:       digitalTrie,
			contractTrie:      contractTrie,
			authorizeTrie:     authorizeTrie,
			lossTrie:          lossTrie,
			dPosHash:          s.trie.dPosHash,
			dPosCandidateHash: s.trie.dPosCandidateHash,
		},
		stateObjects:        make(map[common.Address]*stateObject, len(s.journal.dirties)),
		stateObjectsPending: make(map[common.Address]struct{}, len(s.stateObjectsPending)),
		stateObjectsDirty:   make(map[common.Address]struct{}, len(s.journal.dirties)),
		refund:              s.refund,
		logs:                make(map[common.Hash][]*types.Log, len(s.logs)),
		logSize:             s.logSize,
		preimages:           make(map[common.Hash][]byte, len(s.preimages)),
		journal:             newJournal(),
		hasher:              crypto.NewKeccakState(),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range s.journal.dirties {
		// As documented [here](https://github.com/ethereum/go-ethereum/pull/16485#issuecomment-380438527),
		// and in the Finalise-method, there is a case where an object is in the journal but not
		// in the stateObjects: OOG after touch on ripeMD prior to Byzantium. Thus, we need to check for
		// nil
		if object, exist := s.stateObjects[addr]; exist {
			// Even though the original object is dirty, we are not copying the journal,
			// so we need to make sure that anyside effect the journal would have caused
			// during a commit (or similar op) is already applied to the copy.
			state.stateObjects[addr] = object.deepCopy(state)

			state.stateObjectsDirty[addr] = struct{}{}   // Mark the copy dirty to force internal (code/state) commits
			state.stateObjectsPending[addr] = struct{}{} // Mark the copy pending to force external (account) commits
		}
	}
	// Above, we don't copy the actual journal. This means that if the copy is copied, the
	// loop above will be a no-op, since the copy's journal is empty.
	// Thus, here we iterate over stateObjects, to enable copies of copies
	for addr := range s.stateObjectsPending {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = s.stateObjects[addr].deepCopy(state)
		}
		state.stateObjectsPending[addr] = struct{}{}
	}
	for addr := range s.stateObjectsDirty {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = s.stateObjects[addr].deepCopy(state)
		}
		state.stateObjectsDirty[addr] = struct{}{}
	}
	for hash, logs := range s.logs {
		cpy := make([]*types.Log, len(logs))
		for i, l := range logs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		state.logs[hash] = cpy
	}
	for hash, preimage := range s.preimages {
		state.preimages[hash] = preimage
	}
	// Do we need to copy the access list? In practice: No. At the start of a
	// transaction, the access list is empty. In practice, we only ever copy state
	// _between_ transactions/blocks, never in the middle of a transaction.
	// However, it doesn't cost us much to copy an empty list, so we do it anyway
	// to not blow up if we ever decide copy it in the middle of a transaction
	state.accessList = s.accessList.Copy()

	// If there's a prefetcher running, make an inactive copy of it that can
	// only access data but does not actively preload (since the user will not
	// know that they need to explicitly terminate an active copy).
	if s.prefetcher != nil {
		state.prefetcher = s.prefetcher.copy()
	}
	if s.snaps != nil {
		// In order for the miner to be able to use and make additions
		// to the snapshot tree, we need to copy that aswell.
		// Otherwise, any block mined by ourselves will cause gaps in the tree,
		// and force the miner to operate trie-backed only
		state.snaps = s.snaps
		state.snap = s.snap
		// deep copy needed
		state.snapDestructs = make(map[common.Hash]struct{})
		for k, v := range s.snapDestructs {
			state.snapDestructs[k] = v
		}
		state.snapAccounts = make(map[common.Hash][]byte)
		for k, v := range s.snapAccounts {
			state.snapAccounts[k] = v
		}
		state.snapStorage = make(map[common.Hash]map[common.Hash][]byte)
		for k, v := range s.snapStorage {
			temp := make(map[common.Hash][]byte)
			for kk, vv := range v {
				temp[kk] = vv
			}
			state.snapStorage[k] = temp
		}
	}
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	id := s.nextRevisionId
	s.nextRevisionId++
	s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= revid
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.journal.revert(s, snapshot)
	s.validRevisions = s.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	return s.refund
}

// Finalise finalises the state by removing the s destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	addressesToPrefetch := make([][]byte, 0, len(s.journal.dirties))
	for addr := range s.journal.dirties {
		obj, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		if obj.suicided {
			obj.deleted = true

			// If state snapshotting is active, also mark the destruction there.
			// Note, we can't do this only at the end of a block because multiple
			// transactions within the same block might self destruct and then
			// ressurrect an account; but the snapshotter needs both events.
			if s.snap != nil {
				s.snapDestructs[obj.addrHash] = struct{}{} // We need to maintain account deletions explicitly (will remain set indefinitely)
				delete(s.snapAccounts, obj.addrHash)       // Clear out any previously updated account data (may be recreated via a ressurrect)
				delete(s.snapStorage, obj.addrHash)        // Clear out any previously updated storage data (may be recreated via a ressurrect)
			}
		} else {
			obj.finalise(true) // Prefetch slots in the background
		}
		s.stateObjectsPending[addr] = struct{}{}
		s.stateObjectsDirty[addr] = struct{}{}

		// At this point, also ship the address off to the precacher. The precacher
		// will start loading tries, and when the change is eventually committed,
		// the commit-phase will be a lot faster
		addressesToPrefetch = append(addressesToPrefetch, common.CopyBytes(addr[:])) // Copy needed for closure
	}
	if s.prefetcher != nil && len(addressesToPrefetch) > 0 {
		s.prefetcher.prefetch(s.originalRoot, addressesToPrefetch)
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Finalise all the dirty storage states and write them into the tries
	s.Finalise(deleteEmptyObjects)

	// If there was a trie prefetcher operating, it gets aborted and irrevocably
	// modified after we start retrieving tries. Remove it from the statedb after
	// this round of use.
	//
	// This is weird pre-byzantium since the first tx runs with a prefetcher and
	// the remainder without, but pre-byzantium even the initial prefetcher is
	// useless, so no sleep lost.
	//prefetcher := s.prefetcher
	//if s.prefetcher != nil {
	//	defer func() {
	//		s.prefetcher.close()
	//		s.prefetcher = nil
	//	}()
	//}
	// Although naively it makes sense to retrieve the account trie and then do
	// the contract storage and account updates sequentially, that short circuits
	// the account prefetcher. Instead, let's process all the storage updates
	// first, giving the account prefeches just a few more milliseconds of time
	// to pull useful data from disk.
	for addr := range s.stateObjectsPending {
		if obj := s.stateObjects[addr]; !obj.deleted {
			obj.updateRoot(s.db)
		}
	}
	// Now we're about to start to write changes to the trie. The trie is so far
	// _untouched_. We can check with the prefetcher, if it can give us a trie
	// which has the same root, but also has some content loaded into it.
	//if prefetcher != nil {
	//	if trie := prefetcher.trie(s.originalRoot); trie != nil {
	//		//s.trie = trie
	//	}
	//}
	usedAddrs := make([][]byte, 0, len(s.stateObjectsPending))
	for addr := range s.stateObjectsPending {
		if obj := s.stateObjects[addr]; obj.deleted {
			s.deleteStateObject(obj)
		} else {
			s.updateStateObject(obj)
		}
		usedAddrs = append(usedAddrs, common.CopyBytes(addr[:])) // Copy needed for closure
	}
	//if prefetcher != nil {
	//	prefetcher.used(s.originalRoot, usedAddrs)
	//}
	if len(s.stateObjectsPending) > 0 {
		s.stateObjectsPending = make(map[common.Address]struct{})
	}
	// Track the amount of time wasted on hashing the account trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.AccountHashes += time.Since(start) }(time.Now())
	}
	return s.trie.Hash()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (s *StateDB) Prepare(thash common.Hash, ti int) {
	s.thash = thash
	s.txIndex = ti
	s.accessList = newAccessList()
}

func (s *StateDB) clearJournalAndRefund() {
	if len(s.journal.entries) > 0 {
		s.journal = newJournal()
		s.refund = 0
	}
	s.validRevisions = s.validRevisions[:0] // Snapshots can be created without journal entires
}

// Commit writes the state to the underlying in-memory trie database.
func (s *StateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	if s.dbErr != nil {
		return common.Hash{}, fmt.Errorf("commit aborted due to earlier error: %v", s.dbErr)
	}
	// Finalize any pending changes and merge everything into the tries
	s.IntermediateRoot(deleteEmptyObjects)

	// Commit objects to the trie, measuring the elapsed time
	codeWriter := s.db.TrieDB().DiskDB().NewBatch()
	for addr := range s.stateObjectsDirty {
		if obj := s.stateObjects[addr]; !obj.deleted {
			// Write any contract code associated with the state object
			if obj.code != nil && obj.dirtyCode {
				rawdb.WriteCode(codeWriter, common.BytesToHash(obj.CodeHash()), obj.code)
				obj.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie
			if err := obj.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
		}
	}
	if len(s.stateObjectsDirty) > 0 {
		s.stateObjectsDirty = make(map[common.Address]struct{})
	}
	if codeWriter.ValueSize() > 0 {
		if err := codeWriter.Write(); err != nil {
			log.Crit("Failed to commit dirty codes", "error", err)
		}
	}
	// Write the account trie changes, measuing the amount of wasted time
	var start time.Time
	if metrics.EnabledExpensive {
		start = time.Now()
	}
	// The onleaf func is called _serially_, so we can reuse the same account
	// for unmarshalling every time.
	var account AssetAccount
	root, err := s.trie.Commit(func(_ [][]byte, _ []byte, leaf []byte, parent common.Hash) error {
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.StorageRoot != emptyRoot {
			s.db.TrieDB().Reference(account.StorageRoot, parent)
		}
		return nil
	})

	if metrics.EnabledExpensive {
		s.AccountCommits += time.Since(start)
	}
	// If snapshotting is enabled, update the snapshot tree with this new version
	if s.snap != nil {
		if metrics.EnabledExpensive {
			defer func(start time.Time) { s.SnapshotCommits += time.Since(start) }(time.Now())
		}
		// Only update if there's a state transition (skip empty Clique blocks)
		if parent := s.snap.Root(); parent != root {
			if err := s.snaps.Update(root, parent, s.snapDestructs, s.snapAccounts, s.snapStorage); err != nil {
				log.Warn("Failed to update snapshot tree", "from", parent, "to", root, "err", err)
			}
			// Keep 128 diff layers in the memory, persistent layer is 129th.
			// - head layer is paired with HEAD state
			// - head-1 layer is paired with HEAD-1 state
			// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
			if err := s.snaps.Cap(root, 128); err != nil {
				log.Warn("Failed to cap snapshot tree", "root", root, "layers", 128, "err", err)
			}
		}
		s.snap, s.snapDestructs, s.snapAccounts, s.snapStorage = nil, nil, nil, nil
	}
	return root, err
}

// PrepareAccessList handles the preparatory steps for executing a state transition with
// regards to both EIP-2929 and EIP-2930:
//
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// This method should only be called if Berlin/2929+2930 is applicable at the current number.
func (s *StateDB) PrepareAccessList(sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
	s.AddAddressToAccessList(sender)
	if dst != nil {
		s.AddAddressToAccessList(*dst)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		s.AddAddressToAccessList(addr)
	}
	for _, el := range list {
		s.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			s.AddSlotToAccessList(el.Address, key)
		}
	}
}

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr common.Address) {
	if s.accessList.AddAddress(addr) {
		s.journal.append(accessListAddAccountChange{&addr})
	}
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	addrMod, slotMod := s.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		s.journal.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		s.journal.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr common.Address) bool {
	return s.accessList.ContainsAddress(addr)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return s.accessList.Contains(addr, slot)
}

// GetRegular 获取普通账户
func (s *StateDB) GetRegular(addr common.Address) *RegularAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return &stateObject.regularAccount
	}
	return nil
}

// GetPns PNS账号
func (s *StateDB) GetPns(addr common.Address) *PnsAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return &stateObject.pnsAccount
	}
	return nil
}

// GetAsset 资产账户
func (s *StateDB) GetAsset(addr common.Address) AssetAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.assetAccount
	}
	return AssetAccount{}
}

// GetContract 合约账户
func (s *StateDB) GetContract(addr common.Address) AssetAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.assetAccount
	}
	return AssetAccount{}
}

// GetAuthorize 授权账户
func (s *StateDB) GetAuthorize(addr common.Address) *AuthorizeAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return &stateObject.authorizeAccount
	}
	return nil
}

// GetLoss 挂失账户
func (s *StateDB) GetLoss(addr common.Address) *LossAccount {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return &stateObject.lossAccount
	}
	return nil
}

func (s *StateDB) ModifyLossType(context vm.TxContext) {
	s.SubBalance(context.From, context.Value)
	stateObject := s.getStateObject(context.From)
	if stateObject != nil {
		stateObject.db.journal.append(lossTypeForRegularChange{
			account: &stateObject.address,
			prev:    stateObject.regularAccount.LossType,
		})
		stateObject.regularAccount.LossType = uint8(*context.LossType)
	}
}
func (s *StateDB) Vote(context vm.TxContext) {
	fmt.Printf("Vote, sender:%s,to:%s,amount:%s\n", context.From.String(), context.To.String(), context.Value.String())
	s.SubBalance(context.From, context.Value)
	fromObj := s.getStateObject(context.From)
	if fromObj != nil {
		var lastVoteValue = new(big.Int).SetUint64(0)
		if fromObj.regularAccount.VoteValue != nil {
			lastVoteValue = fromObj.regularAccount.VoteValue
		}
		fromObj.db.journal.append(voteForRegularChange{
			account:     &fromObj.address,
			voteAccount: fromObj.regularAccount.VoteAccount,
			voteValue:   lastVoteValue,
		})
		fromObj.regularAccount.VoteAccount = *context.To
		fromObj.regularAccount.VoteValue = new(big.Int).Add(context.Value, lastVoteValue)
	}

	toObj := s.getStateObject(*context.To)
	if toObj != nil {
		toObj.db.journal.append(voteValueForAuthorizeChange{
			account: &toObj.address,
			prev:    toObj.authorizeAccount.VoteValue,
		})
		toObj.authorizeAccount.VoteValue = new(big.Int).Add(toObj.authorizeAccount.VoteValue, context.Value)
	}
}

func (s *StateDB) Register(context vm.TxContext) {
	fmt.Printf("Register, sender:%s,new:%s,pledge:%s\n", context.From.String(), context.New.String(), context.Value.String())
	s.SubBalance(context.From, context.Value)
	obj, _ := s.createObject(*context.New)
	switch byte(*context.AccType) {
	case common.ACC_TYPE_OF_PNS:
		obj.pnsAccount.Owner = context.From
		obj.pnsAccount.Data = context.Data
		obj.pnsAccount.Type = byte(0)
	case common.ACC_TYPE_OF_ASSET:
	//case common.ACC_TYPE_OF_CONTRACT:
	case common.ACC_TYPE_OF_AUTHORIZE:
		createAccountFee := common.AmountOfPledgeForCreateAccount(uint8(*context.AccType))
		pledgeValue := new(big.Int).Sub(context.Value, new(big.Int).SetUint64(createAccountFee))
		obj.authorizeAccount.PledgeValue = pledgeValue
		obj.authorizeAccount.Owner = context.From
		obj.authorizeAccount.ValidPeriod = context.Height
		obj.authorizeAccount.VoteValue = pledgeValue
		obj.authorizeAccount.Info = context.Data
	case common.ACC_TYPE_OF_LOSE:
		obj.lossAccount.State = common.LOSS_STATE_OF_INIT
		s.setMarkLossAccount(*context.New)
	case common.ACC_TYPE_OF_DPOS:
	case common.ACC_TYPE_OF_DPOS_CANDIDATE:
	}

}

func (s *StateDB) Cancellation(context vm.TxContext) {
	fmt.Printf("Cancellation, sender:%s,to:%s,new:%s,value:%s\n", context.From, context.To, context.New, context.Value)
	s.AddBalance(*context.New, context.Value)
	//s.DeleteStateObjectByAddr(*context.To)
	s.Suicide(*context.To)
}

func (s *StateDB) Transfer(context vm.TxContext) {
	fmt.Printf("Transfer, sender:%s,to:%s,amount:%s\n", context.From.String(), context.To.String(), context.Value.String())
	s.SubBalance(context.From, context.Value)
	s.AddBalance(*context.To, context.Value)
}

//todo
func (s *StateDB) ExchangeAsset(context vm.TxContext) {

}
func (s *StateDB) SendLossReport(blockNumber *big.Int, context vm.TxContext) {
	fmt.Printf("SendLossReport, sender:%s,mark:%s,infoDigest:%s\n", context.From, context.Mark, context.InfoDigest)
	s.SubBalance(context.From, context.Value)
	var addrs = s.GetMarkLossAccounts(common.BytesToHash(context.Mark))
	if len(addrs) > 0 {
		for _, addr := range addrs {
			stateObject := s.getStateObject(addr)
			if stateObject != nil && stateObject.lossAccount.State == common.LOSS_STATE_OF_INIT {
				stateObject.db.journal.append(sendLossReportChange{
					account:    &stateObject.address,
					infoDigest: stateObject.lossAccount.InfoDigest,
					state:      stateObject.lossAccount.State,
					height:     stateObject.lossAccount.Height,
				})
				stateObject.lossAccount.InfoDigest = context.InfoDigest
				stateObject.lossAccount.State = common.LOSS_STATE_OF_APPLY
				stateObject.lossAccount.Height = blockNumber
			}
		}
	}
}

func (s *StateDB) RevealLossReport(blockNumber *big.Int, context vm.TxContext) {
	s.SubBalance(context.From, context.Value)
	s.AddBalance(*context.Old, context.Value)
	stateObject := s.getStateObject(*context.To)
	if stateObject != nil && stateObject.lossAccount.State == common.LOSS_STATE_OF_APPLY {
		stateObject.db.journal.append(revealLossReportChange{
			account:     &stateObject.address,
			lossAccount: stateObject.lossAccount.LossAccount,
			newAccount:  stateObject.lossAccount.NewAccount,
			height:      stateObject.lossAccount.Height,
			state:       stateObject.lossAccount.State,
		})
		stateObject.lossAccount.LossAccount = *context.Old
		stateObject.lossAccount.NewAccount = *context.New
		stateObject.lossAccount.State = common.LOSS_STATE_OF_NOTICE
		stateObject.lossAccount.Height = blockNumber
	}
}

func (s *StateDB) TransferLostAccount(context vm.TxContext) {
	stateObject := s.getStateObject(*context.To)
	if stateObject != nil && stateObject.lossAccount.State == common.LOSS_STATE_OF_NOTICE {
		balance := s.GetBalance(stateObject.lossAccount.LossAccount)
		if balance.Sign() > 0 {
			stateObject.db.journal.append(transferLostAccountChange{
				account: &stateObject.address,
				state:   stateObject.lossAccount.State,
			})
			stateObject.lossAccount.State = common.LOSS_STATE_OF_SUCCESS
			s.AddBalance(stateObject.lossAccount.NewAccount, balance)
			s.SetBalance(stateObject.lossAccount.LossAccount, new(big.Int).SetInt64(0))
		}
	}
}

//todo
func (s *StateDB) TransferLostAssetAccount(context vm.TxContext) {

}

func (s *StateDB) RemoveLossReport(context vm.TxContext) {
	s.AddBalance(context.From, context.Value)
	//s.DeleteStateObjectByAddr(*context.To)
	s.Suicide(*context.To)
}

func (s *StateDB) RejectLossReport(context vm.TxContext) {
	s.AddBalance(context.From, context.Value)
	//s.DeleteStateObjectByAddr(*context.To)
	s.Suicide(*context.To)
}

func (s *StateDB) ModifyPnsOwner(context vm.TxContext) {
	s.SubBalance(context.From, context.Value)
	stateObject := s.getStateObject(*context.To)
	if stateObject != nil {
		stateObject.db.journal.append(modifyPnsOwnerChange{
			account: &stateObject.address,
			owner:   stateObject.pnsAccount.Owner,
		})
		stateObject.pnsAccount.Owner = *context.New
	}
}

func (s *StateDB) ModifyPnsContent(context vm.TxContext) {
	s.SubBalance(context.From, context.Value)
	stateObject := s.getStateObject(*context.To)
	if stateObject != nil {
		stateObject.db.journal.append(modifyPnsContentChange{
			account: &stateObject.address,
			pnsType: stateObject.pnsAccount.Type,
			data:    stateObject.pnsAccount.Data,
		})
		stateObject.pnsAccount.Type = byte(*context.PnsType)
		stateObject.pnsAccount.Data = context.Data
	}
}

func (s *StateDB) RedemptionForRegular(addr common.Address) (common.Address, *big.Int) {
	stateObject := s.getStateObject(addr)
	var VoteAccount common.Address
	var voteValue *big.Int
	if stateObject != nil {
		stateObject.db.journal.append(redemptionForRegularChange{
			account:     &stateObject.address,
			voteAccount: stateObject.regularAccount.VoteAccount,
			voteValue:   stateObject.regularAccount.VoteValue,
			value:       stateObject.regularAccount.Value,
		})
		VoteAccount = stateObject.regularAccount.VoteAccount
		voteValue = stateObject.regularAccount.VoteValue
		stateObject.regularAccount.Value = new(big.Int).Add(stateObject.regularAccount.Value, voteValue)
		stateObject.regularAccount.VoteAccount = common.Address{}
		stateObject.regularAccount.VoteValue = new(big.Int).SetUint64(0)
	}
	return VoteAccount, voteValue
}
func (s *StateDB) RedemptionForAuthorize(addr common.Address, voteValue *big.Int) {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		stateObject.db.journal.append(redemptionForAuthorizeChange{
			account:     &stateObject.address,
			pledgeValue: stateObject.authorizeAccount.PledgeValue,
			voteValue:   stateObject.authorizeAccount.VoteValue,
		})
		if voteValue == nil {
			stateObject.authorizeAccount.VoteValue = new(big.Int).Sub(stateObject.authorizeAccount.VoteValue, stateObject.authorizeAccount.PledgeValue)
			stateObject.authorizeAccount.PledgeValue = new(big.Int).SetUint64(0)
		} else {
			stateObject.authorizeAccount.VoteValue = new(big.Int).Sub(stateObject.authorizeAccount.VoteValue, voteValue)
		}
	}
}

func (s *StateDB) Redemption(context vm.TxContext) {
	s.SubBalance(context.From, context.Value)
	s1 := s.getStateObject(context.From)
	if s1 != nil {
		regularAccount := s1.regularAccount
		s2 := s.getStateObject(*context.To)
		if s2 != nil {
			authorizeAccount := s1.authorizeAccount
			if context.From == authorizeAccount.Owner {
				s.RedemptionForAuthorize(*context.To, nil)
			}
			if *context.To == regularAccount.VoteAccount {
				s.RedemptionForRegular(context.From)
				s.RedemptionForAuthorize(*context.To, regularAccount.VoteValue)
			}
		}
	}
}

func (s *StateDB) ApplyToBeDPoSNode(context vm.TxContext) {
	stateObject := s.getStateObject(*context.To)
	if nil == stateObject {
		return
	}
	var dposMap map[string]interface{}
	err := json.Unmarshal(context.Data, &dposMap)
	if err != nil {
		s.setError(fmt.Errorf("getDeleteStateObject (%x) error: %v", context.To.Bytes(), err))
		return
	}
	remoteIp := dposMap["ip"].(string)
	remotePort := dposMap["port"].(string)
	var enode bytes.Buffer
	enode.WriteString("enode://")
	enode.WriteString(context.From.String()[2:])
	enode.WriteString("@")
	enode.WriteString(remoteIp)
	enode.WriteString(":")
	enode.WriteString(remotePort)
	stateObject.dposCandidateAccount.Enode = common.BytesToDposEnode([]byte(enode.String()))
	stateObject.dposCandidateAccount.Owner = context.From
}

// DeleteStateObjectByAddr removes the given object from the state trie.
/*func (s *StateDB) DeleteStateObjectByAddr(addr common.Address) {
state := s.stateObjects[addr]
if state != nil {
	state.deleted = true
}

// Track the amount of time wasted on deleting the account from the trie
/*	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.AccountUpdates += time.Since(start) }(time.Now())
	}
	// Delete the account from the trie
	if err := s.trie.TryDelete(addr[:]); err != nil {
		s.setError(fmt.Errorf("deleteStateObject (%x) error: %v", addr[:], err))
	}*/
//}*/

func (s *StateDB) newAccountDataByAddr(addr common.Address, enc []byte) (*stateObject, bool) {
	accountType, err := common.ValidAddress(addr)
	if err != nil {
		log.Error("Failed to ValidAddress", "addr", addr, "err", err)
		return nil, true
	}
	switch accountType {
	case common.ACC_TYPE_OF_GENERAL:
		data := new(RegularAccount)
		if enc != nil {
			if err := rlp.DecodeBytes(enc, data); err != nil {
				log.Error("Failed to decode state object", "addr", addr, "err", err)
				return nil, true
			}
		}
		return newRegularAccount(s, addr, *data), false
	case common.ACC_TYPE_OF_PNS:
		data := new(PnsAccount)
		if enc != nil {
			if err := rlp.DecodeBytes(enc, data); err != nil {
				log.Error("Failed to decode state object", "addr", addr, "err", err)
				return nil, true
			}
		}
		return newPnsAccount(s, addr, *data), false
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		data := new(AssetAccount)
		if enc != nil {
			if err := rlp.DecodeBytes(enc, data); err != nil {
				log.Error("Failed to decode state object", "addr", addr, "err", err)
				return nil, true
			}
		}
		return newAssetAccount(s, addr, *data), false
	case common.ACC_TYPE_OF_AUTHORIZE:
		data := new(AuthorizeAccount)
		if enc != nil {
			if err := rlp.DecodeBytes(enc, data); err != nil {
				log.Error("Failed to decode state object", "addr", addr, "err", err)
				return nil, true
			}
		}
		return newAuthorizeAccount(s, addr, *data), false
	case common.ACC_TYPE_OF_LOSE:
		data := new(LossAccount)
		if enc != nil {
			if err := rlp.DecodeBytes(enc, data); err != nil {
				log.Error("Failed to decode state object", "addr", addr, "err", err)
				return nil, true
			}
		}
		return newLossAccount(s, addr, *data), false
	case common.ACC_TYPE_OF_DPOS:
		return nil, true
	case common.ACC_TYPE_OF_DPOS_CANDIDATE:
		return nil, true
	default:
		return nil, true
	}
}

// getStateObjectTireByAccountType return stateObject's tire
func (s *StateDB) getStateObjectTireByAccountType(accountType byte) *Trie {
	switch accountType {
	case common.ACC_TYPE_OF_GENERAL:
		return &s.trie.regularTrie
	case common.ACC_TYPE_OF_PNS:
		return &s.trie.pnsTrie
	case common.ACC_TYPE_OF_ASSET:
		return &s.trie.digitalTrie
	case common.ACC_TYPE_OF_CONTRACT:
		return &s.trie.contractTrie
	case common.ACC_TYPE_OF_AUTHORIZE:
		return &s.trie.authorizeTrie
	case common.ACC_TYPE_OF_LOSE:
		return &s.trie.lossTrie
	case common.ACC_TYPE_OF_DPOS:
		return nil
	case common.ACC_TYPE_OF_DPOS_CANDIDATE:
		return nil
	default:
		return nil
	}
	//return &s.trie
}

func (s *StateDB) GetStateDbTrie() *TotalTrie {
	return &s.trie
}
func (s *StateDB) GetDpostList() []common.DPoSAccount {
	/*var dPoSAccounts = make([]common.DPoSAccount, s.dPoSCandidateList.Limit)
	i := 0
	for element := s.dPoSCandidateList.List.Front(); element != nil; element = element.Next() {
		dPoSCandidateAccount := element.Value.(DPoSCandidateAccount)
		dPoSAccount := &common.DPoSAccount{dPoSCandidateAccount.Enode, dPoSCandidateAccount.Owner}
		dPoSAccounts[i] = *dPoSAccount
		i++
	}
	return dPoSAccounts*/
	return s.dposList.dPoSCandidateAccounts.GetDpostList()
}

func (s *StateDB) ChangDpostAccount(dposAccounts []common.DPoSAccount) {
	for i, dposAccount := range dposAccounts {
		s.dposList.oldDPoSAccounts[i] = s.dposList.dPoSAccounts[i]
		s.dposList.dPoSAccounts[i] = dposAccount
	}
}

func (s *StateDB) GetDPosHashByRoot(root common.Hash, db Database) common.Hash {
	hashes := GetHash(root, db)

	return hashes[6]
}

func (s *StateDB) GetDPosCandidateHashByRoot(root common.Hash, db Database) common.Hash {
	hashes := GetHash(root, db)

	return hashes[7]
}

func (s *StateDB) IntermediateRootForDPos(dPosHash common.Hash) common.Hash {
	//  dPosHash
	s.trie.dPosHash = dPosHash
	return s.trie.Hash()
}

func (s *StateDB) IntermediateRootForDPosCandidate(dPosCandidateHash common.Hash) common.Hash {
	s.trie.dPosCandidateHash = dPosCandidateHash
	return s.trie.Hash()
}

func (s *StateDB) PrintTrie() {
	//s.trie.Print()
}
