// Copyright 2014 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/common/hexutil"
	"github.com/probechain/go-probe/log"
	"github.com/probechain/go-probe/rlp"
	"github.com/probechain/go-probe/trie"
)

// DumpConfig is a set of options to control what portions of the statewill be
// iterated and collected.
type DumpConfig struct {
	SkipCode          bool
	SkipStorage       bool
	OnlyWithAddresses bool
	Start             []byte
	Max               uint64
}

// DumpCollector interface which the state trie calls during iteration
type DumpCollector interface {
	// OnRoot is called with the state root
	OnRoot(common.Hash)
	// OnAccount is called once for each account in the trie
	OnAccount(common.Address, DumpAccount)
}

// DumpAccount represents an account in the state.
type DumpAccount struct {
	Balance string `json:"balance"`
	//Nonce     uint64                 `json:"nonce"`
	Root hexutil.Bytes `json:"root"`
	//CodeHash  hexutil.Bytes          `json:"codeHash"`
	Code      hexutil.Bytes          `json:"code,omitempty"`
	Storage   map[common.Hash]string `json:"storage,omitempty"`
	Address   *common.Address        `json:"address,omitempty"` // Address only present in iterative (line-by-line) mode
	SecureKey hexutil.Bytes          `json:"key,omitempty"`     // If we don't have address, we can output the key

	VoteAccount *common.Address `json:"voteAccount"`
	LossAccount *common.Address `json:"lossAccount"`
	NewAccount  *common.Address `json:"newAccount"`
	Owner       *common.Address `json:"owner"`
	VoteValue   hexutil.Big     `json:"voteValue"`
	PledgeValue hexutil.Big     `json:"pledgeValue"`
	ValidPeriod hexutil.Big     `json:"validPeriod"`
	Height      hexutil.Big     `json:"height"`
	Weight      hexutil.Big     `json:"weight"`
	Value       hexutil.Big     `json:"value"`
	Nonce       uint64          `json:"nonce"`
	Data        hexutil.Bytes   `json:"data"`
	CodeHash    hexutil.Bytes   `json:"codeHash"`
	StorageRoot common.Hash     `json:"storageRoot"`
	Info        hexutil.Bytes   `json:"info"`
	InfoDigest  common.Hash     `json:"infoDigest"`
	State       hexutil.Uint8   `json:"lossState"`
	Type        hexutil.Uint8   `json:"type"`
	Ip          hexutil.Bytes   `json:"ip"`
	Port        hexutil.Uint8   `json:"port"`
	LossType    hexutil.Uint8   `json:"lossType"`
	PnsType     hexutil.Uint8   `json:"pnsType"`
	AccountType hexutil.Uint8   `json:"accountType"`
}

// Dump represents the full dump in a collected format, as one large map.
type Dump struct {
	Root     string                         `json:"root"`
	Accounts map[common.Address]DumpAccount `json:"accounts"`
}

// OnRoot implements DumpCollector interface
func (d *Dump) OnRoot(root common.Hash) {
	d.Root = fmt.Sprintf("%x", root)
}

// OnAccount implements DumpCollector interface
func (d *Dump) OnAccount(addr common.Address, account DumpAccount) {
	d.Accounts[addr] = account
}

// IteratorDump is an implementation for iterating over data.
type IteratorDump struct {
	Root     string                         `json:"root"`
	Accounts map[common.Address]DumpAccount `json:"accounts"`
	Next     []byte                         `json:"next,omitempty"` // nil if no more accounts
}

// OnRoot implements DumpCollector interface
func (d *IteratorDump) OnRoot(root common.Hash) {
	d.Root = fmt.Sprintf("%x", root)
}

// OnAccount implements DumpCollector interface
func (d *IteratorDump) OnAccount(addr common.Address, account DumpAccount) {
	d.Accounts[addr] = account
}

// iterativeDump is a DumpCollector-implementation which dumps output line-by-line iteratively.
type iterativeDump struct {
	*json.Encoder
}

// OnAccount implements DumpCollector interface
func (d iterativeDump) OnAccount(addr common.Address, account DumpAccount) {
	dumpAccount := &DumpAccount{
		/*		Balance:   account.Balance,
				Nonce:     account.Nonce,
				Root:      account.Root,
				CodeHash:  account.CodeHash,
				Code:      account.Code,
				Storage:   account.Storage,
				SecureKey: account.SecureKey,
				Address:   nil,*/

		Root:        account.Root,
		Code:        account.Code,
		Storage:     account.Storage,
		SecureKey:   account.SecureKey,
		VoteAccount: account.VoteAccount,
		LossAccount: account.LossAccount,
		NewAccount:  account.NewAccount,
		Owner:       account.Owner,
		VoteValue:   account.VoteValue,
		PledgeValue: account.PledgeValue,
		ValidPeriod: account.ValidPeriod,
		Height:      account.Height,
		Value:       account.Value,
		Nonce:       account.Nonce,
		Data:        account.Data,
		CodeHash:    account.CodeHash,
		StorageRoot: account.StorageRoot,
		Info:        account.Info,
		InfoDigest:  account.InfoDigest,
		State:       account.State,
		Type:        account.Type,
		Ip:          account.Ip,
		Port:        account.Port,
		LossType:    account.LossType,
		PnsType:     account.PnsType,
		AccountType: account.AccountType,
	}
	if addr != (common.Address{}) {
		dumpAccount.Address = &addr
	}
	d.Encode(dumpAccount)
}

// OnRoot implements DumpCollector interface
func (d iterativeDump) OnRoot(root common.Hash) {
	d.Encode(struct {
		Root common.Hash `json:"root"`
	}{root})
}

// DumpToCollector iterates the state according to the given options and inserts
// the items into a collector for aggregation or serialization.
/*func (s *StateDB) DumpToCollector(c DumpCollector, conf *DumpConfig) (nextKey []byte) {
	// Sanitize the input to allow nil configs
	if conf == nil {
		conf = new(DumpConfig)
	}
	var (
		missingPreimages int
		accounts         uint64
		start            = time.Now()
		logged           = time.Now()
	)
	log.Info("Trie dumping started", "root", s.trie.Hash())
	c.OnRoot(s.trie.Hash())

	it := trie.NewIterator(s.trie.NodeIterator(conf.Start))
	for it.Next() {
		var data RegularAccount
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}
		account := DumpAccount{
			Balance:   data.Balance.String(),
			Nonce:     data.Nonce,
			Root:      data.Root[:],
			CodeHash:  data.CodeHash,
			SecureKey: it.Key,
		}
		addrBytes := s.trie.GetKey(it.Key)
		if addrBytes == nil {
			// Preimage missing
			missingPreimages++
			if conf.OnlyWithAddresses {
				continue
			}
			account.SecureKey = it.Key
		}
		addr := common.BytesToAddress(addrBytes)
		obj := newRegularAccount(s, addr, data)
		if !conf.SkipCode {
			account.Code = obj.Code(s.db)
		}
		if !conf.SkipStorage {
			account.Storage = make(map[common.Hash]string)
			storageIt := trie.NewIterator(obj.getTrie(s.db).NodeIterator(nil))
			for storageIt.Next() {
				_, content, _, err := rlp.Split(storageIt.Value)
				if err != nil {
					log.Error("Failed to decode the value returned by iterator", "error", err)
					continue
				}
				account.Storage[common.BytesToHash(s.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(content)
			}
		}
		c.OnAccount(addr, account)
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Trie dumping in progress", "at", it.Key, "accounts", accounts,
				"elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
		if conf.Max > 0 && accounts >= conf.Max {
			if it.Next() {
				nextKey = it.Key
			}
			break
		}
	}
	if missingPreimages > 0 {
		log.Warn("Dump incomplete due to missing preimages", "missing", missingPreimages)
	}
	log.Info("Trie dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))

	return nextKey
}*/

func (s *StateDB) DumpToCollector(c DumpCollector, conf *DumpConfig) (nextKey []byte) {
	// Sanitize the input to allow nil configs
	if conf == nil {
		conf = new(DumpConfig)
	}
	var (
		missingPreimages int
		accountCounter   uint64
		start            = time.Now()
		logged           = time.Now()
		account          DumpAccount
		wrapper          *Wrapper
		err              error
	)
	trieArr := [1]Trie{s.trie}
	for i, t := range trieArr {
		log.Info("Trie dumping started", "root", t.Hash())
		c.OnRoot(t.Hash())
		it := trie.NewIterator(t.NodeIterator(conf.Start))
		for it.Next() {
			if i == 0 {
				wrapper, err = DecodeRLP(it.Value, common.ACC_TYPE_OF_REGULAR)
				if err != nil {
					continue
				}
				account = DumpAccount{
					VoteAccount: &wrapper.regularAccount.VoteAccount,
					VoteValue:   hexutil.Big(*wrapper.regularAccount.VoteValue),
					LossType:    hexutil.Uint8(wrapper.regularAccount.LossType),
					Nonce:       wrapper.regularAccount.Nonce,
					Value:       hexutil.Big(*wrapper.regularAccount.Value),
					SecureKey:   it.Key,

					//Balance:   wrapper.regularAccount.Balance.String(),
					//Nonce:     wrapper.regularAccount.Nonce,
					//Root:      wrapper.regularAccount.Root[:],
					//CodeHash:  wrapper.regularAccount.CodeHash,
					//SecureKey: it.Key,
				}
			}
			if i == 1 {
				wrapper, err = DecodeRLP(it.Value, common.ACC_TYPE_OF_PNS)
				if err != nil {
					continue
				}
				account = DumpAccount{
					Owner: &wrapper.pnsAccount.Owner,
					Type:  hexutil.Uint8(wrapper.pnsAccount.Type),
					Data:  wrapper.pnsAccount.Data,
					//Nonce:			wrapper.pnsAccount.Nonce,
				}
			}
			if i == 2 || i == 3 {
				wrapper, err = DecodeRLP(it.Value, common.ACC_TYPE_OF_CONTRACT)
				if err != nil {
					continue
				}
				account = DumpAccount{
					CodeHash:    wrapper.assetAccount.CodeHash,
					StorageRoot: wrapper.assetAccount.StorageRoot,
					Value:       hexutil.Big(*wrapper.assetAccount.Value),
					VoteAccount: &wrapper.assetAccount.VoteAccount,
					VoteValue:   hexutil.Big(*wrapper.assetAccount.VoteValue),
					Nonce:       wrapper.assetAccount.Nonce,
				}
			}
			if i == 4 {
				wrapper, err = DecodeRLP(it.Value, common.ACC_TYPE_OF_AUTHORIZE)
				if err != nil {
					continue
				}
				account = DumpAccount{
					Owner:       &wrapper.authorizeAccount.Owner,
					PledgeValue: hexutil.Big(*wrapper.authorizeAccount.PledgeValue),
					VoteValue:   hexutil.Big(*wrapper.authorizeAccount.VoteValue),
					Info:        wrapper.authorizeAccount.Info,
					ValidPeriod: hexutil.Big(*wrapper.authorizeAccount.ValidPeriod),
					//Nonce:			wrapper.authorizeAccount.Nonce,
				}
			}
			if i == 5 {
				wrapper, err = DecodeRLP(it.Value, common.ACC_TYPE_OF_LOSS)
				if err != nil {
					continue
				}
				account = DumpAccount{
					State:       hexutil.Uint8(wrapper.lossAccount.State),
					LossAccount: &wrapper.lossAccount.LostAccount,
					NewAccount:  &wrapper.lossAccount.NewAccount,
					Height:      hexutil.Big(*wrapper.lossAccount.Height),
					InfoDigest:  wrapper.lossAccount.InfoDigest,
					//Nonce:			wrapper.lossAccount.Nonce,
				}
			}
			addrBytes := t.GetKey(it.Key)
			if addrBytes == nil {
				// Preimage missing
				missingPreimages++
				if conf.OnlyWithAddresses {
					continue
				}
				account.SecureKey = it.Key
			}
			addr := common.BytesToAddress(addrBytes)
			obj := newObjectByWrapper(s, addr, wrapper)
			if !conf.SkipCode {
				account.Code = obj.Code(s.db)
			}
			if !conf.SkipStorage {
				account.Storage = make(map[common.Hash]string)
				storageIt := trie.NewIterator(obj.getTrie(s.db).NodeIterator(nil))
				for storageIt.Next() {
					_, content, _, err := rlp.Split(storageIt.Value)
					if err != nil {
						log.Error("Failed to decode the value returned by iterator", "error", err)
						continue
					}
					account.Storage[common.BytesToHash(t.GetKey(storageIt.Key))] = common.Bytes2Hex(content)
				}
			}
			c.OnAccount(addr, account)
			accountCounter++
			if time.Since(logged) > 8*time.Second {
				log.Info("Trie dumping in progress", "at", it.Key, "accounts", accountCounter,
					"elapsed", common.PrettyDuration(time.Since(start)))
				logged = time.Now()
			}
			if conf.Max > 0 && accountCounter >= conf.Max {
				if it.Next() {
					nextKey = it.Key
				}
				break
			}
		}
	}

	if missingPreimages > 0 {
		log.Warn("Dump incomplete due to missing preimages", "missing", missingPreimages)
	}
	log.Info("Trie dumping complete", "accounts", accountCounter,
		"elapsed", common.PrettyDuration(time.Since(start)))

	return nextKey
}

// RawDump returns the entire state an a single large object
func (s *StateDB) RawDump(opts *DumpConfig) Dump {
	dump := &Dump{
		Accounts: make(map[common.Address]DumpAccount),
	}
	s.DumpToCollector(dump, opts)
	return *dump
}

// Dump returns a JSON string representing the entire state as a single json-object
func (s *StateDB) Dump(opts *DumpConfig) []byte {
	dump := s.RawDump(opts)
	json, err := json.MarshalIndent(dump, "", "    ")
	if err != nil {
		fmt.Println("Dump err", err)
	}
	return json
}

// IterativeDump dumps out accounts as json-objects, delimited by linebreaks on stdout
func (s *StateDB) IterativeDump(opts *DumpConfig, output *json.Encoder) {
	s.DumpToCollector(iterativeDump{output}, opts)
}

// IteratorDump dumps out a batch of accounts starts with the given start key
func (s *StateDB) IteratorDump(opts *DumpConfig) IteratorDump {
	iterator := &IteratorDump{
		Accounts: make(map[common.Address]DumpAccount),
	}
	iterator.Next = s.DumpToCollector(iterator, opts)
	return *iterator
}
