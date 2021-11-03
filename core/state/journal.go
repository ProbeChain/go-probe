// Copyright 2016 The go-probeum Authors
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
	"math/big"

	"github.com/probeum/go-probeum/common"
)

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type journalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	revert(*StateDB)

	// dirtied returns the Probeum address modified by this journal entry.
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

	regularSuicideChange struct {
		account     *common.Address
		suicide     bool
		voteAccount common.Address
		voteValue   *big.Int
		lossType    uint8
		value       *big.Int
	}

	pnsSuicideChange struct {
		account *common.Address
		suicide bool
		pnsType byte
		owner   common.Address
		data    []byte
	}

	assetSuicideChange struct {
		account     *common.Address
		suicide     bool
		assetType   byte
		value       *big.Int
		voteAccount common.Address
		voteValue   *big.Int
	}

	authorizeSuicideChange struct {
		account     *common.Address
		suicide     bool
		owner       common.Address
		pledgeValue *big.Int
		voteValue   *big.Int
		weight      *big.Int
		info        []byte
		validPeriod *big.Int
	}

	lossSuicideChange struct {
		account     *common.Address
		suicide     bool
		state       byte
		lossAccount common.Address
		newAccount  common.Address
		height      *big.Int
		infoDigest  []byte
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

	voteForRegularChange struct {
		account     *common.Address
		voteAccount common.Address
		voteValue   *big.Int
	}
	lossTypeForRegularChange struct {
		account *common.Address
		prev    uint8
	}

	voteValueForAuthorizeChange struct {
		account *common.Address
		prev    *big.Int
	}

	sendLossReportChange struct {
		account    *common.Address
		state      byte
		height     *big.Int
		infoDigest []byte
	}

	revealLossReportChange struct {
		account     *common.Address
		lossAccount common.Address
		newAccount  common.Address
		height      *big.Int
		state       byte
	}

	transferLostAccountChange struct {
		account *common.Address
		state   byte
	}

	redemptionForRegularChange struct {
		account     *common.Address
		voteAccount common.Address
		voteValue   *big.Int
		value       *big.Int
	}

	modifyPnsOwnerChange struct {
		account *common.Address
		owner   common.Address
	}

	modifyPnsContentChange struct {
		account *common.Address
		pnsType byte
		data    []byte
	}
	redemptionForAuthorizeChange struct {
		account     *common.Address
		pledgeValue *big.Int
		voteValue   *big.Int
	}

	dPosCandidateForAuthorizeChange struct {
		account   *common.Address
		info      []byte
		voteValue big.Int
	}
)

func (i sendLossReportChange) revert(db *StateDB) {
	var lossAccount = db.getStateObject(*i.account).lossAccount
	lossAccount.InfoDigest = i.infoDigest
	lossAccount.State = i.state
	lossAccount.Height = i.height
}

func (i sendLossReportChange) dirtied() *common.Address {
	return i.account
}

func (d voteValueForAuthorizeChange) revert(db *StateDB) {
	db.getStateObject(*d.account).authorizeAccount.VoteValue = d.prev
}

func (d voteValueForAuthorizeChange) dirtied() *common.Address {
	return d.account
}

func (l lossTypeForRegularChange) revert(db *StateDB) {
	db.getStateObject(*l.account).regularAccount.LossType = l.prev
}

func (l lossTypeForRegularChange) dirtied() *common.Address {
	return l.account
}

func (v voteForRegularChange) revert(db *StateDB) {
	regularAccount := db.getStateObject(*v.account).regularAccount
	regularAccount.VoteAccount = v.voteAccount
	regularAccount.VoteValue = v.voteValue
}

func (v voteForRegularChange) dirtied() *common.Address {
	return v.account
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

func (ch regularSuicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.suicide
		obj.regularAccount.VoteAccount = ch.voteAccount
		obj.regularAccount.VoteValue = ch.voteValue
		obj.regularAccount.LossType = ch.lossType
		obj.regularAccount.Value = ch.value
	}
}

func (ch regularSuicideChange) dirtied() *common.Address {
	return ch.account
}

func (ch pnsSuicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.suicide
		obj.pnsAccount.Type = ch.pnsType
		obj.pnsAccount.Owner = ch.owner
		obj.pnsAccount.Data = ch.data
	}
}

func (ch pnsSuicideChange) dirtied() *common.Address {
	return ch.account
}

func (ch assetSuicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.suicide
		obj.assetAccount.VoteAccount = ch.voteAccount
		obj.assetAccount.VoteValue = ch.voteValue
		obj.assetAccount.Value = ch.value
	}
}
func (ch assetSuicideChange) dirtied() *common.Address {
	return ch.account
}

func (ch authorizeSuicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.suicide
		obj.authorizeAccount.Owner = ch.owner
		obj.authorizeAccount.PledgeValue = ch.pledgeValue
		obj.authorizeAccount.VoteValue = ch.voteValue
		obj.authorizeAccount.Info = ch.info
		obj.authorizeAccount.ValidPeriod = ch.validPeriod
	}
}

func (ch authorizeSuicideChange) dirtied() *common.Address {
	return ch.account
}

func (ch lossSuicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.suicide
		obj.lossAccount.State = ch.state
		obj.lossAccount.LossAccount = ch.lossAccount
		obj.lossAccount.NewAccount = ch.newAccount
		obj.lossAccount.Height = ch.height
		obj.lossAccount.InfoDigest = ch.infoDigest
	}
}

func (ch lossSuicideChange) dirtied() *common.Address {
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

func (ch dPosCandidateForAuthorizeChange) revert(s *StateDB) {
	authorizeAccount := s.getStateObject(*ch.account).authorizeAccount

	dPosCandidateAccount := common.DPoSCandidateAccount{}
	dPosCandidateAccount.Owner = authorizeAccount.Owner
	dPosCandidateAccount.Vote = *ch.account
	dPosCandidateAccount.VoteValue = &ch.voteValue
	if len(ch.info) == 0 {
		dPosCandidateAccount.Enode = common.BytesToDposEnode(authorizeAccount.Info)
		GetDPosCandidates().DeleteDPosCandidate(dPosCandidateAccount)
	} else {
		dPosCandidateAccount.Enode = common.BytesToDposEnode(ch.info)
		GetDPosCandidates().UpdateDPosCandidate(dPosCandidateAccount)
	}
	authorizeAccount.VoteValue = &ch.voteValue
	authorizeAccount.Info = ch.info

}

func (ch dPosCandidateForAuthorizeChange) dirtied() *common.Address {
	return ch.account
}

func (ch revealLossReportChange) revert(s *StateDB) {
	lossAccount := s.getStateObject(*ch.account).lossAccount
	lossAccount.LossAccount = ch.lossAccount
	lossAccount.NewAccount = ch.newAccount
	lossAccount.State = ch.state
}
func (ch revealLossReportChange) dirtied() *common.Address {
	return ch.account
}

func (ch transferLostAccountChange) revert(s *StateDB) {
	lossAccount := s.getStateObject(*ch.account).lossAccount
	lossAccount.State = ch.state
}
func (ch transferLostAccountChange) dirtied() *common.Address {
	return ch.account
}

func (ch modifyPnsOwnerChange) revert(s *StateDB) {
	pnsAccount := s.getStateObject(*ch.account).pnsAccount
	pnsAccount.Owner = ch.owner
}
func (ch modifyPnsOwnerChange) dirtied() *common.Address {
	return ch.account
}

func (ch modifyPnsContentChange) revert(s *StateDB) {
	pnsAccount := s.getStateObject(*ch.account).pnsAccount
	pnsAccount.Type = ch.pnsType
	pnsAccount.Data = ch.data
}
func (ch modifyPnsContentChange) dirtied() *common.Address {
	return ch.account
}
