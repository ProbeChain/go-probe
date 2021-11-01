// Copyright 2014 The go-probeum Authors
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

package state

import (
	"bytes"
	"fmt"
	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/metrics"
	"github.com/probeum/go-probeum/rlp"
	"io"
	"math/big"
	"strconv"
	"time"
)

var emptyCodeHash = crypto.Keccak256(nil)

type Code []byte

func (c Code) String() string {
	return string(c) //strings.Join(Disassemble(c), " ")
}

type Storage map[common.Hash]common.Hash

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// stateObject represents an Probeum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of probeum address of the account
	db       *StateDB

	accountType byte

	regularAccount RegularAccount
	// PnsAccount PNS账号
	pnsAccount PnsAccount
	// AssetAccount 资产账户
	assetAccount AssetAccount
	// AuthorizeAccount 授权账户
	authorizeAccount AuthorizeAccount
	// 挂失账户
	lossAccount LossAccount

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites, reset for every transaction
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution
	fakeStorage    Storage // Fake storage which constructed by caller for debugging purpose.

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}

// Account is the Probeum consensus representation of accounts.
// These objects are stored in the main account trie.
//type RegularAccount struct {
//	Nonce    uint64
//	Balance  *big.Int
//	Root     common.Hash // merkle root of the storage trie
//	CodeHash []byte
//}

// RegularAccount 普通账户
type RegularAccount struct {
	VoteAccount common.Address
	VoteValue   *big.Int
	LossType    uint8
	Nonce       uint64
	Value       *big.Int
}

// PnsAccount PNS账号
type PnsAccount struct {
	Type  byte
	Owner common.Address
	Data  []byte
}

// AssetAccount 资产账户 和 合约账户
type AssetAccount struct {
	Type     byte
	CodeHash []byte
	//StorageRoot []byte
	StorageRoot common.Hash
	Value       *big.Int
	VoteAccount common.Address
	VoteValue   *big.Int
	Nonce       uint64
}

// AuthorizeAccount 授权账户
type AuthorizeAccount struct {
	Owner       common.Address
	PledgeValue *big.Int
	VoteValue   *big.Int
	Info        []byte
	ValidPeriod *big.Int
}

// LossAccount 挂失账户
type LossAccount struct {
	State       byte           // 业务状态 0:初始化，1:挂失申请，2：揭示中
	LossAccount common.Address // 挂失账户地址
	NewAccount  common.Address // 新账户地址
	Height      *big.Int       // 揭示时区块高度
	InfoDigest  []byte         // 挂失内容摘要
}

type Wrapper struct {
	accountType          byte
	regularAccount       RegularAccount
	pnsAccount           PnsAccount
	assetAccount         AssetAccount
	authorizeAccount     AuthorizeAccount
	lossAccount          LossAccount
	dPoSAccount          common.DPoSAccount
	dPoSCandidateAccount common.DPoSCandidateAccount
}

type RPCAccountInfo struct {
	Owner         *common.Address `json:"owner,omitempty"`
	LossAccount   *common.Address `json:"lossAccount,omitempty"`
	NewAccount    *common.Address `json:"newAccount,omitempty"`
	VoteAccount   *common.Address `json:"voteAccount,omitempty"`
	VoteValue     string          `json:"voteValue,omitempty"`
	PledgeValue   string          `json:"pledgeValue,omitempty"`
	Value         string          `json:"value,omitempty"`
	ValidPeriod   string          `json:"validPeriod,omitempty"`
	Height        string          `json:"height,omitempty"`
	Weight        string          `json:"weight,omitempty"`
	DelegateValue string          `json:"delegateValue,omitempty"`
	LossType      string          `json:"lossType,omitempty"`
	Nonce         string          `json:"nonce,omitempty"`
	Type          string          `json:"type,omitempty"`
	State         string          `json:"state,omitempty"`
	Data          string          `json:"data,omitempty"`
	CodeHash      string          `json:"codeHash,omitempty"`
	Info          string          `json:"info,omitempty"`
}

// DecodeRLP decode bytes to account
func DecodeRLP(encodedBytes []byte, accountType byte) (*Wrapper, error) {
	var (
		wrapper Wrapper
		err     error
	)
	switch accountType {
	case common.ACC_TYPE_OF_GENERAL:
		var data RegularAccount
		err = rlp.DecodeBytes(encodedBytes, &data)
		wrapper.regularAccount = data
	case common.ACC_TYPE_OF_PNS:
		var data PnsAccount
		err = rlp.DecodeBytes(encodedBytes, &data)
		wrapper.pnsAccount = data
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		var data AssetAccount
		err = rlp.DecodeBytes(encodedBytes, &data)
		wrapper.assetAccount = data
	case common.ACC_TYPE_OF_AUTHORIZE:
		var data AuthorizeAccount
		err = rlp.DecodeBytes(encodedBytes, &data)
		wrapper.authorizeAccount = data
	case common.ACC_TYPE_OF_LOSE:
		var data LossAccount
		err = rlp.DecodeBytes(encodedBytes, &data)
		wrapper.lossAccount = data
	default:
		err = accounts.ErrUnknownAccount
	}
	wrapper.accountType = accountType
	return &wrapper, err
}

// newRegularAccount creates a state object.
func newObjectByWrapper(db *StateDB, address common.Address, wrapper *Wrapper) *stateObject {
	trie := *db.getStateObjectTireByAccountType(wrapper.accountType)
	return &stateObject{
		db:               db,
		address:          address,
		accountType:      wrapper.accountType,
		trie:             trie,
		addrHash:         crypto.Keccak256Hash(address[:]),
		regularAccount:   wrapper.regularAccount,
		pnsAccount:       wrapper.pnsAccount,
		assetAccount:     wrapper.assetAccount,
		authorizeAccount: wrapper.authorizeAccount,
		lossAccount:      wrapper.lossAccount,
		originStorage:    make(Storage),
		pendingStorage:   make(Storage),
		dirtyStorage:     make(Storage),
	}
}

// newRegularAccount creates a state object.
func newRegularAccount(db *StateDB, address common.Address, data RegularAccount) *stateObject {
	if data.Value == nil {
		data.Value = new(big.Int)
	}
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		accountType:    common.ACC_TYPE_OF_GENERAL,
		regularAccount: data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// newPnsAccount creates a state object.
func newPnsAccount(db *StateDB, address common.Address, data PnsAccount) *stateObject {
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		accountType:    common.ACC_TYPE_OF_PNS,
		pnsAccount:     data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// newAssetAccount creates a state object.
func newAssetAccount(db *StateDB, address common.Address, data AssetAccount) *stateObject {
	if data.Value == nil {
		data.Value = new(big.Int)
	}
	if data.VoteValue == nil {
		data.VoteValue = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	if data.StorageRoot == (common.Hash{}) {
		data.StorageRoot = emptyRoot
	}
	accType, _ := common.ValidAddress(address)
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		accountType:    accType,
		assetAccount:   data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// newAuthorizeAccount creates a state object.
func newAuthorizeAccount(db *StateDB, address common.Address, data AuthorizeAccount) *stateObject {
	return &stateObject{
		db:               db,
		address:          address,
		addrHash:         crypto.Keccak256Hash(address[:]),
		accountType:      common.ACC_TYPE_OF_AUTHORIZE,
		authorizeAccount: data,
		originStorage:    make(Storage),
		pendingStorage:   make(Storage),
		dirtyStorage:     make(Storage),
	}
}

// newLossAccount creates a state object.
func newLossAccount(db *StateDB, address common.Address, data LossAccount) *stateObject {
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		accountType:    common.ACC_TYPE_OF_LOSE,
		lossAccount:    data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// newDPoSAccount creates a state object.
/*func newDPoSAccount(db *StateDB, address common.Address, data DPoSAccount) *stateObject {
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		accountType: 	common.ACC_TYPE_OF_PNS,
		dPoSAccount: 	data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}*/

// EncodeRLP implements rlp.Encoder.
/*func (s *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.regularAccount)
}*/

// EncodeRLP implements rlp.Encoder.
func (s *stateObject) EncodeRLP(w io.Writer) error {
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		return rlp.Encode(w, s.regularAccount)
	case common.ACC_TYPE_OF_PNS:
		return rlp.Encode(w, s.pnsAccount)
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		return rlp.Encode(w, s.assetAccount)
	case common.ACC_TYPE_OF_AUTHORIZE:
		return rlp.Encode(w, s.authorizeAccount)
	case common.ACC_TYPE_OF_LOSE:
		return rlp.Encode(w, s.lossAccount)
	default:
		return accounts.ErrUnknownAccount
	}
}

// setError remembers the first non-nil error it is called with.
func (s *stateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *stateObject) markSuicided() {
	s.suicided = true
}

func (s *stateObject) touch() {
	s.db.journal.append(touchChange{
		account: &s.address,
	})
	if s.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		s.db.journal.dirty(s.address)
	}
}

func (s *stateObject) getTrie(db Database) Trie {
	if s.trie == nil {
		// Try fetching from prefetcher first
		// We don't prefetch empty tries
		//if s.regularAccount.Root != emptyRoot && s.db.prefetcher != nil {
		if (common.Hash{}) != s.assetAccount.StorageRoot && s.assetAccount.StorageRoot != emptyRoot && s.db.prefetcher != nil {
			// When the miner is creating the pending state, there is no
			// prefetcher
			//s.trie = s.db.prefetcher.trie(s.regularAccount.Root)
			s.trie = s.db.prefetcher.trie(s.assetAccount.StorageRoot)
		}
		if s.trie == nil {
			var err error
			//s.trie, err = db.OpenStorageTrie(s.addrHash, s.regularAccount.Root)
			s.trie, err = db.OpenStorageTrie(s.addrHash, s.assetAccount.StorageRoot)
			if err != nil {
				s.trie, _ = db.OpenStorageTrie(s.addrHash, common.Hash{})
				s.setError(fmt.Errorf("can't create storage trie: %v", err))
			}
		}
	}
	return s.trie
}

// GetState retrieves a value from the account storage trie.
func (s *stateObject) GetState(db Database, key common.Hash) common.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a dirty value for this state entry, return it
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value
	}
	// Otherwise return the entry's original value
	return s.GetCommittedState(db, key)
}

// GetCommittedState retrieves a value from the committed account storage trie.
func (s *stateObject) GetCommittedState(db Database, key common.Hash) common.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If no live objects are available, attempt to use snapshots
	var (
		enc   []byte
		err   error
		meter *time.Duration
	)
	readStart := time.Now()
	if metrics.EnabledExpensive {
		// If the snap is 'under construction', the first lookup may fail. If that
		// happens, we don't want to double-count the time elapsed. Thus this
		// dance with the metering.
		defer func() {
			if meter != nil {
				*meter += time.Since(readStart)
			}
		}()
	}
	if s.db.snap != nil {
		if metrics.EnabledExpensive {
			meter = &s.db.SnapshotStorageReads
		}
		// If the object was destructed in *this* block (and potentially resurrected),
		// the storage has been cleared out, and we should *not* consult the previous
		// snapshot about any storage values. The only possible alternatives are:
		//   1) resurrect happened, and new slot values were set -- those should
		//      have been handles via pendingStorage above.
		//   2) we don't have new values, and can deliver empty response back
		if _, destructed := s.db.snapDestructs[s.addrHash]; destructed {
			return common.Hash{}
		}
		enc, err = s.db.snap.Storage(s.addrHash, crypto.Keccak256Hash(key.Bytes()))
	}
	// If snapshot unavailable or reading from it failed, load from the database
	if s.db.snap == nil || err != nil {
		if meter != nil {
			// If we already spent time checking the snapshot, account for it
			// and reset the readStart
			*meter += time.Since(readStart)
			readStart = time.Now()
		}
		if metrics.EnabledExpensive {
			meter = &s.db.StorageReads
		}
		newKey := common.ReBuildAddress(key.Bytes())
		//if enc, err = s.getTrie(db).TryGet(key.Bytes()); err != nil {
		if enc, err = s.getTrie(db).TryGet(newKey); err != nil {
			s.setError(err)
			return common.Hash{}
		}
	}
	var value common.Hash
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			s.setError(err)
		}
		value.SetBytes(content)
	}
	s.originStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (s *stateObject) SetState(db Database, key, value common.Hash) {
	// If the fake storage is set, put the temporary state update here.
	if s.fakeStorage != nil {
		s.fakeStorage[key] = value
		return
	}
	// If the new value is the same as old, don't set
	prev := s.GetState(db, key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

// SetStorage replaces the entire state storage with the given one.
//
// After this function is called, all original state will be ignored and state
// lookup only happens in the fake state storage.
//
// Note this function should only be used for debugging purpose.
func (s *stateObject) SetStorage(storage map[common.Hash]common.Hash) {
	// Allocate fake storage if it's nil.
	if s.fakeStorage == nil {
		s.fakeStorage = make(Storage)
	}
	for key, value := range storage {
		s.fakeStorage[key] = value
	}
	// Don't bother journal since this function should only be used for
	// debugging and the `fake` storage won't be committed to database.
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (s *stateObject) finalise(prefetch bool) {
	slotsToPrefetch := make([][]byte, 0, len(s.dirtyStorage))
	for key, value := range s.dirtyStorage {
		s.pendingStorage[key] = value
		if value != s.originStorage[key] {
			slotsToPrefetch = append(slotsToPrefetch, common.CopyBytes(key[:])) // Copy needed for closure
		}
	}
	//if s.db.prefetcher != nil && prefetch && len(slotsToPrefetch) > 0 && s.regularAccount.Root != emptyRoot {
	if s.db.prefetcher != nil && prefetch && len(slotsToPrefetch) > 0 && (common.Hash{}) != s.assetAccount.StorageRoot && s.assetAccount.StorageRoot != emptyRoot {
		//s.db.prefetcher.prefetch(s.regularAccount.Root, slotsToPrefetch)
		s.db.prefetcher.prefetch(s.assetAccount.StorageRoot, slotsToPrefetch)
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
// It will return nil if the trie has not been loaded and no changes have been made
func (s *stateObject) updateTrie(db Database) Trie {
	// Make sure all dirty slots are finalized into the pending storage area
	s.finalise(false) // Don't prefetch any more, pull directly if need be
	if len(s.pendingStorage) == 0 {
		return s.trie
	}
	// Track the amount of time wasted on updating the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageUpdates += time.Since(start) }(time.Now())
	}
	// The snapshot storage map for the object
	var storage map[common.Hash][]byte
	// Insert all the pending updates into the trie
	tr := s.getTrie(db)
	hasher := s.db.hasher

	usedStorage := make([][]byte, 0, len(s.pendingStorage))
	for key, value := range s.pendingStorage {
		newKey := common.ReBuildAddress(key.Bytes())
		// Skip noop changes, persist actual changes
		if value == s.originStorage[key] {
			continue
		}
		s.originStorage[key] = value

		var v []byte
		if (value == common.Hash{}) {
			//s.setError(tr.TryDelete(key[:]))
			s.setError(tr.TryDelete(newKey[:]))
		} else {
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ = rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
			//s.setError(tr.TryUpdate(key[:], v))
			s.setError(tr.TryUpdate(newKey[:], v))
		}
		// If state snapshotting is active, cache the data til commit
		if s.db.snap != nil {
			if storage == nil {
				// Retrieve the old storage map, if available, create a new one otherwise
				if storage = s.db.snapStorage[s.addrHash]; storage == nil {
					storage = make(map[common.Hash][]byte)
					s.db.snapStorage[s.addrHash] = storage
				}
			}
			storage[crypto.HashData(hasher, key[:])] = v // v will be nil if value is 0x00
		}
		usedStorage = append(usedStorage, common.CopyBytes(key[:])) // Copy needed for closure
	}
	if s.db.prefetcher != nil {
		//s.db.prefetcher.used(s.regularAccount.Root, usedStorage)
		s.db.prefetcher.used(s.assetAccount.StorageRoot, usedStorage)
	}
	if len(s.pendingStorage) > 0 {
		s.pendingStorage = make(Storage)
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (s *stateObject) updateRoot(db Database) {
	// If nothing changed, don't bother with hashing anything
	if s.updateTrie(db) == nil {
		return
	}
	// Track the amount of time wasted on hashing the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageHashes += time.Since(start) }(time.Now())
	}
	//s.regularAccount.Root = s.trie.Hash()
	s.assetAccount.StorageRoot = s.trie.Hash()
}

// CommitTrie the storage trie of the object to db.
// This updates the trie root.
func (s *stateObject) CommitTrie(db Database) error {
	// If nothing changed, don't bother with hashing anything
	if s.updateTrie(db) == nil {
		return nil
	}
	if s.dbErr != nil {
		return s.dbErr
	}
	// Track the amount of time wasted on committing the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageCommits += time.Since(start) }(time.Now())
	}
	root, err := s.trie.Commit(nil)
	if err == nil {
		s.assetAccount.StorageRoot = root
		//s.regularAccount.Root = root
	}
	return err
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateObject) AddBalance(amount *big.Int) {
	// EIP161: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		/*		if s.empty() {
				s.touch()
			}*/
		return
	}
	s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.Balance(), amount))
}

func (s *stateObject) SetBalance(amount *big.Int) {
	if s.accountType == common.ACC_TYPE_OF_GENERAL || s.accountType == common.ACC_TYPE_OF_ASSET || s.accountType == common.ACC_TYPE_OF_CONTRACT {
		s.db.journal.append(balanceChange{
			account: &s.address,
			//prev:    new(big.Int).Set(s.regularAccount.Balance),
			prev: new(big.Int).Set(s.Balance()),
		})
		s.setBalance(amount)
	}
}

func (s *stateObject) setBalance(amount *big.Int) {
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		s.setValueForRegular(amount)
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		s.setValueForAsset(amount)
	default:
	}
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	//stateObject := newRegularAccount(db, s.address, s.regularAccount)
	stateObject := s.getNewStateObjectByAddr(db, s.address)
	if s.trie != nil {
		stateObject.trie = db.db.CopyTrie(s.trie)
	}
	stateObject.code = s.code
	stateObject.dirtyStorage = s.dirtyStorage.Copy()
	stateObject.originStorage = s.originStorage.Copy()
	stateObject.pendingStorage = s.pendingStorage.Copy()
	stateObject.suicided = s.suicided
	stateObject.dirtyCode = s.dirtyCode
	stateObject.deleted = s.deleted
	return stateObject
}

func (s *stateObject) getNewStateObjectByAddr(db *StateDB, address common.Address) *stateObject {
	accountType, err := common.ValidAddress(address)
	if err != nil {
		return nil
	}
	var (
		state *stateObject
	)
	switch accountType {
	case common.ACC_TYPE_OF_GENERAL:
		state = newRegularAccount(db, s.address, s.regularAccount)
	case common.ACC_TYPE_OF_PNS:
		state = newPnsAccount(db, s.address, s.pnsAccount)
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		state = newAssetAccount(db, s.address, s.assetAccount)
	case common.ACC_TYPE_OF_AUTHORIZE:
		state = newAuthorizeAccount(db, s.address, s.authorizeAccount)
	case common.ACC_TYPE_OF_LOSE:
		state = newLossAccount(db, s.address, s.lossAccount)
	//case common.ACC_TYPE_OF_DPOS:
	//
	//case common.ACC_TYPE_OF_DPOS_CANDIDATE:
	default:
		state = nil
	}
	return state
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code(db Database) []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := db.ContractCode(s.addrHash, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.setError(fmt.Errorf("can't load code hash %x: %v", s.CodeHash(), err))
	}
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none. This method is an almost mirror of Code, but uses a cache
// inside the database to avoid loading codes seen recently.
func (s *stateObject) CodeSize(db Database) int {
	if s.code != nil {
		return len(s.code)
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return 0
	}
	size, err := db.ContractCodeSize(s.addrHash, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.setError(fmt.Errorf("can't load code size %x: %v", s.CodeHash(), err))
	}
	return size
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code(s.db.db)
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	//s.regularAccount.CodeHash = codeHash[:]
	s.assetAccount.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	if s.accountType == common.ACC_TYPE_OF_GENERAL || s.accountType == common.ACC_TYPE_OF_ASSET || s.accountType == common.ACC_TYPE_OF_CONTRACT {
		s.db.journal.append(nonceChange{
			account: &s.address,
			//prev:    s.regularAccount.Nonce,
			prev: s.Nonce(),
		})
		s.setNonce(nonce)
	}
}

func (s *stateObject) setNonce(nonce uint64) {
	//s.regularAccount.Nonce = nonce
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		s.regularAccount.Nonce = nonce
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		s.assetAccount.Nonce = nonce
	default:
	}
}

func (s *stateObject) CodeHash() []byte {
	//return s.regularAccount.CodeHash
	return s.assetAccount.CodeHash
}

func (s *stateObject) Balance() *big.Int {
	//return s.regularAccount.Value
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		return s.regularAccount.Value
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		return s.assetAccount.Value
	case common.ACC_TYPE_OF_AUTHORIZE:
		return s.authorizeAccount.VoteValue
	default:
		return new(big.Int)
	}
}

func (s *stateObject) AccountInfo() *RPCAccountInfo {
	accountInfo := new(RPCAccountInfo)
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		accountInfo.VoteAccount = &s.regularAccount.VoteAccount
		accountInfo.VoteValue = s.regularAccount.VoteValue.String()
		accountInfo.LossType = strconv.Itoa(int(s.regularAccount.LossType))
		accountInfo.Nonce = strconv.Itoa(int(s.regularAccount.Nonce))
		accountInfo.Value = s.regularAccount.Value.String()
	case common.ACC_TYPE_OF_PNS:
		accountInfo.Type = strconv.Itoa(int(s.pnsAccount.Type))
		accountInfo.Owner = &s.pnsAccount.Owner
		//data := hexutil.Bytes(s.pnsAccount.Data)
		accountInfo.Data = string(s.pnsAccount.Data)
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		accountInfo.Type = strconv.Itoa(int(s.assetAccount.Type))
		codeHash := hexutil.Bytes(s.assetAccount.CodeHash)
		accountInfo.CodeHash = codeHash.String()
		accountInfo.Value = s.assetAccount.Value.String()
		accountInfo.VoteAccount = &s.assetAccount.VoteAccount
		accountInfo.VoteValue = s.assetAccount.VoteValue.String()
		accountInfo.Nonce = strconv.Itoa(int(s.assetAccount.Nonce))
	case common.ACC_TYPE_OF_AUTHORIZE:
		accountInfo.Owner = &s.authorizeAccount.Owner
		accountInfo.PledgeValue = s.authorizeAccount.PledgeValue.String()
		accountInfo.VoteValue = s.authorizeAccount.VoteValue.String()
		//info := hexutil.Bytes(s.authorizeAccount.Info)
		accountInfo.Info = string(s.authorizeAccount.Info)
		accountInfo.ValidPeriod = s.authorizeAccount.ValidPeriod.String()
		//accountInfo.State = strconv.Itoa(int(s.authorizeAccount.State))
	case common.ACC_TYPE_OF_LOSE:
		accountInfo.State = strconv.Itoa(int(s.lossAccount.State))
		accountInfo.LossAccount = &s.lossAccount.LossAccount
		accountInfo.NewAccount = &s.lossAccount.NewAccount
		accountInfo.Height = s.lossAccount.Height.String()
		//infoDigest := hexutil.Bytes(s.lossAccount.InfoDigest)
		accountInfo.Data = string(s.lossAccount.InfoDigest)
	}
	return accountInfo
}

func (s *stateObject) Nonce() uint64 {
	//return s.regularAccount.Nonce
	switch s.accountType {
	case common.ACC_TYPE_OF_GENERAL:
		return s.regularAccount.Nonce
	case common.ACC_TYPE_OF_ASSET, common.ACC_TYPE_OF_CONTRACT:
		return s.assetAccount.Nonce
	default:
		return 0
	}
}

// Never called, but must be present to allow stateObject to be used
// as a vm.Account interface that also satisfies the vm.ContractRef
// interface. Interfaces are awesome.
func (s *stateObject) Value() *big.Int {
	panic("Value on stateObject should never be called")
}

// setValueForAsset only set asset account value
func (s *stateObject) setValueForAsset(value *big.Int) {
	s.assetAccount.Value = value
}

// setValueForRegular only set regular account value
func (s *stateObject) setValueForRegular(value *big.Int) {
	s.regularAccount.Value = value
}
