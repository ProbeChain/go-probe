// Copyright 2016 The go-ethereum Authors
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

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type journalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	revert(*StateDB)

	// dirtied returns the Ethereum address modified by this journal entry.
	dirtied() *common.Address
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in case of an execution
// exception or revertal request.
type journal struct {
	entries []journalEntry         // Current changes tracked by the journal
	dirties map[common.Address]int // Dirty accounts and the number of changes
}

// newJournal create a new initialized journal.
func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Address]int),
	}
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// dirty handling too.
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. This method is an ugly hack to handle the RIPEMD
// precompile consensus exception.
func (j *journal) dirty(addr common.Address) {
	j.dirties[addr]++
}

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev         *stateObject
		prevdestruct bool
	}
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
		prevbalance *big.Int
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	addPreimageChange struct {
		hash common.Hash
	}
	touchChange struct {
		account *common.Address
	}
	// Changes to the access list
	accessListAddAccountChange struct {
		address *common.Address
	}
	accessListAddSlotChange struct {
		address *common.Address
		slot    *common.Hash
	}

	valueForAssetChange struct {
		account *common.Address
		prev    *big.Int
	}
	valueForRegularChange struct {
		account *common.Address
		prev    *big.Int
	}
	voteValueForRegularChange struct {
		account *common.Address
		prev    *big.Int
	}
	voteAccountForRegularChange struct {
		account *common.Address
		prev    common.Address
	}
	voteForRegularChange struct {
		account     *common.Address
		voteAccount common.Address
		voteValue   *big.Int
	}
	lossTypeForRegularChange struct {
		account *common.Address
		prev    uint8
	}
	nonceForRegularChange struct {
		account *common.Address
		prev    uint64
	}
	typeForPnsChange struct {
		account *common.Address
		prev    byte
	}
	ownerForPnsChange struct {
		account *common.Address
		prev    common.Address
	}
	dataForPnsChange struct {
		account *common.Address
		prev    []byte
	}
	typeForAssetChange struct {
		account *common.Address
		prev    byte
	}
	codeHashForAssetChange struct {
		account *common.Address
		prev    []byte
	}
	storageRootForAssetChange struct {
		account *common.Address
		prev    common.Hash
	}
	voteAccountForAssetChange struct {
		account *common.Address
		prev    common.Address
	}
	voteValueForAssetChange struct {
		account *common.Address
		prev    *big.Int
	}
	ownerForAuthorizeChange struct {
		account *common.Address
		prev    common.Address
	}
	pledgeValueForAuthorizeChange struct {
		account *common.Address
		prev    *big.Int
	}
	delegateValueForAuthorizeChange struct {
		account *common.Address
		prev    *big.Int
	}
	infoForAuthorizeChange struct {
		account *common.Address
		prev    []byte
	}
	validPeriodForAuthorizeChange struct {
		account *common.Address
		prev    *big.Int
	}
	stateForAuthorizeChange struct {
		account *common.Address
		prev    byte
	}
	stateForLossChange struct {
		account *common.Address
		prev    byte
	}
	lossAccountForLossChange struct {
		account *common.Address
		prev    common.Address
	}
	heightForLossChange struct {
		account *common.Address
		prev    *big.Int
	}
	infoDigestForLossChange struct {
		account *common.Address
		prev    []byte
	}

	redemptionForRegularChange struct {
		account     *common.Address
		voteAccount common.Address
		voteValue   *big.Int
		value       *big.Int
	}

	redemptionForAuthorizeChange struct {
		account     *common.Address
		pledgeValue *big.Int
		voteValue   *big.Int
	}
)

func (i infoDigestForLossChange) revert(db *StateDB) {
	db.getStateObject(*i.account).lossAccount.InfoDigest = i.prev
}

func (i infoDigestForLossChange) dirtied() *common.Address {
	return i.account
}

func (h heightForLossChange) revert(db *StateDB) {
	db.getStateObject(*h.account).lossAccount.Height = h.prev
}

func (h heightForLossChange) dirtied() *common.Address {
	return h.account
}

func (l lossAccountForLossChange) revert(db *StateDB) {
	db.getStateObject(*l.account).lossAccount.LossAccount = l.prev
}

func (l lossAccountForLossChange) dirtied() *common.Address {
	return l.account
}

func (s stateForLossChange) revert(db *StateDB) {
	db.getStateObject(*s.account).lossAccount.State = s.prev
}

func (s stateForLossChange) dirtied() *common.Address {
	return s.account
}

func (s stateForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*s.account).authorizeAccount.State = s.prev
}

func (s stateForAuthorizeChange) dirtied() *common.Address {
	return s.account
}

func (v validPeriodForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*v.account).authorizeAccount.ValidPeriod = v.prev
}

func (v validPeriodForAuthorizeChange) dirtied() *common.Address {
	return v.account
}

func (i infoForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*i.account).authorizeAccount.Info = i.prev
}

func (i infoForAuthorizeChange) dirtied() *common.Address {
	return i.account
}

func (d delegateValueForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*d.account).authorizeAccount.VoteValue = d.prev
}

func (d delegateValueForAuthorizeChange) dirtied() *common.Address {
	return d.account
}

func (p pledgeValueForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*p.account).authorizeAccount.PledgeValue = p.prev
}

func (p pledgeValueForAuthorizeChange) dirtied() *common.Address {
	return p.account
}

func (o ownerForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*o.account).authorizeAccount.Owner = o.prev
}

func (o ownerForAuthorizeChange) dirtied() *common.Address {
	return o.account
}

func (v voteValueForAssetChange) revert(db *StateDB) {
	db.getStateObject(*v.account).assetAccount.VoteValue = v.prev
}

func (v voteValueForAssetChange) dirtied() *common.Address {
	return v.account
}

func (v voteAccountForAssetChange) revert(db *StateDB) {
	db.getStateObject(*v.account).assetAccount.VoteAccount = v.prev
}

func (v voteAccountForAssetChange) dirtied() *common.Address {
	return v.account
}

func (s storageRootForAssetChange) revert(db *StateDB) {
	db.getStateObject(*s.account).assetAccount.StorageRoot = s.prev
}

func (s storageRootForAssetChange) dirtied() *common.Address {
	return s.account
}

func (c codeHashForAssetChange) revert(db *StateDB) {
	db.getStateObject(*c.account).assetAccount.CodeHash = c.prev
}

func (c codeHashForAssetChange) dirtied() *common.Address {
	return c.account
}

func (t typeForAssetChange) revert(db *StateDB) {
	db.getStateObject(*t.account).assetAccount.Type = t.prev
}

func (t typeForAssetChange) dirtied() *common.Address {
	return t.account
}

func (d dataForPnsChange) revert(db *StateDB) {
	db.getStateObject(*d.account).pnsAccount.Data = d.prev
}

func (d dataForPnsChange) dirtied() *common.Address {
	return d.account
}

func (o ownerForPnsChange) revert(db *StateDB) {
	db.getStateObject(*o.account).pnsAccount.Owner = o.prev
}

func (o ownerForPnsChange) dirtied() *common.Address {
	return o.account
}

func (n typeForPnsChange) revert(db *StateDB) {
	db.getStateObject(*n.account).pnsAccount.Type = n.prev
}

func (n typeForPnsChange) dirtied() *common.Address {
	return n.account
}
func (n nonceForRegularChange) revert(db *StateDB) {
	db.getStateObject(*n.account).regularAccount.Nonce = n.prev
}

func (n nonceForRegularChange) dirtied() *common.Address {
	return n.account
}
func (l lossTypeForRegularChange) revert(db *StateDB) {
	db.getStateObject(*l.account).regularAccount.LossType = l.prev
}

func (l lossTypeForRegularChange) dirtied() *common.Address {
	return l.account
}

func (v voteAccountForRegularChange) revert(db *StateDB) {
	db.getStateObject(*v.account).regularAccount.VoteAccount = v.prev
}

func (v voteAccountForRegularChange) dirtied() *common.Address {
	return v.account
}

func (v voteForRegularChange) revert(db *StateDB) {
	regularAccount := db.getStateObject(*v.account).regularAccount
	regularAccount.VoteAccount = v.voteAccount
	regularAccount.VoteValue = v.voteValue
}

func (v voteForRegularChange) dirtied() *common.Address {
	return v.account
}

func (v voteValueForRegularChange) revert(db *StateDB) {
	db.getStateObject(*v.account).regularAccount.VoteValue = v.prev
}

func (v voteValueForRegularChange) dirtied() *common.Address {
	return v.account
}

func (ch valueForAssetChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setValueForAsset(ch.prev)
}

func (ch valueForAssetChange) dirtied() *common.Address {
	return ch.account
}

func (ch valueForRegularChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setValueForRegular(ch.prev)
}

func (ch valueForRegularChange) dirtied() *common.Address {
	return ch.account
}

func (ch createObjectChange) revert(s *StateDB) {
	delete(s.stateObjects, *ch.account)
	delete(s.stateObjectsDirty, *ch.account)
}

func (ch createObjectChange) dirtied() *common.Address {
	return ch.account
}

func (ch resetObjectChange) revert(s *StateDB) {
	s.setStateObject(ch.prev)
	if !ch.prevdestruct && s.snap != nil {
		delete(s.snapDestructs, ch.prev.addrHash)
	}
}

func (ch resetObjectChange) dirtied() *common.Address {
	return nil
}

func (ch suicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

func (ch suicideChange) dirtied() *common.Address {
	return ch.account
}

var ripemd = common.HexToAddress("0x00000000000000000000000000000000000000000000000003")

func (ch touchChange) revert(s *StateDB) {
}

func (ch touchChange) dirtied() *common.Address {
	return ch.account
}

func (ch balanceChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setBalance(ch.prev)
}

func (ch balanceChange) dirtied() *common.Address {
	return ch.account
}

func (ch nonceChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch nonceChange) dirtied() *common.Address {
	return ch.account
}

func (ch codeChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch codeChange) dirtied() *common.Address {
	return ch.account
}

func (ch storageChange) revert(s *StateDB) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prevalue)
}

func (ch storageChange) dirtied() *common.Address {
	return ch.account
}

func (ch refundChange) revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) dirtied() *common.Address {
	return nil
}

func (ch addLogChange) revert(s *StateDB) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}

func (ch addLogChange) dirtied() *common.Address {
	return nil
}

func (ch addPreimageChange) revert(s *StateDB) {
	delete(s.preimages, ch.hash)
}

func (ch addPreimageChange) dirtied() *common.Address {
	return nil
}

func (ch accessListAddAccountChange) revert(s *StateDB) {
	/*
		One important invariant here, is that whenever a (addr, slot) is added, if the
		addr is not already present, the add causes two journal entries:
		- one for the address,
		- one for the (address,slot)
		Therefore, when unrolling the change, we can always blindly delete the
		(addr) at this point, since no storage adds can remain when come upon
		a single (addr) change.
	*/
	s.accessList.DeleteAddress(*ch.address)
}

func (ch accessListAddAccountChange) dirtied() *common.Address {
	return nil
}

func (ch accessListAddSlotChange) revert(s *StateDB) {
	s.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) dirtied() *common.Address {
	return nil
}

func (ch redemptionForRegularChange) revert(s *StateDB) {
	regularAccount := s.getStateObject(*ch.account).regularAccount
	regularAccount.VoteAccount = ch.voteAccount
	regularAccount.VoteValue = ch.voteValue
	regularAccount.Value = ch.value
}

func (ch redemptionForRegularChange) dirtied() *common.Address {
	return ch.account
}

func (ch redemptionForAuthorizeChange) revert(s *StateDB) {
	authorizeAccount := s.getStateObject(*ch.account).authorizeAccount
	authorizeAccount.PledgeValue = ch.pledgeValue
	authorizeAccount.VoteValue = ch.voteValue
}

func (ch redemptionForAuthorizeChange) dirtied() *common.Address {
	return ch.account
}
