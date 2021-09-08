package state

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"testing"
)

type stateNewTest struct {
	db    ethdb.Database
	state *StateDB
}

func newStateNewTest() *stateNewTest {
	db := rawdb.NewMemoryDatabase()
	sdb, _ := New(common.Hash{}, NewDatabase(db), nil)
	return &stateNewTest{db: db, state: sdb}
}

func TestGetData(t *testing.T) {
	s := newStateNewTest()
	address := common.BytesToAddress([]byte{0x01})
	s.state.SetValueForRegular(address, big.NewInt(20))
	obj1 := s.state.GetRegular(address)
	regular := s.state.GetValueForRegular(address)
	fmt.Printf(" after GetRegular：%v \n", obj1)
	fmt.Printf(" after GetValueForRegular：%v \n", regular)
}

func TestSetData(t *testing.T) {
	s := newStateNewTest()
	address := common.BytesToAddress([]byte{0x01})
	s.state.SetVoteValueForRegular(address, big.NewInt(10))
	fmt.Printf(" after SetValueForRegular：%v \n", s.state.GetRegular(address))
}

func TestTrieAndRlp(t *testing.T) {
	s := newStateNewTest()
	address := common.BytesToAddress([]byte{0x01})
	obj1 := s.state.GetOrNewStateObject(address)
	fmt.Printf(" before GetOrNewStateObject：%v \n", obj1.regularAccount)
	//obj1.setValueForRegular(big.NewInt(20))
	obj1.regularAccount.Value = big.NewInt(20)

	var data2 *RegularAccount
	// write some of them to the trie
	//s.state.updateStateObject(obj1)
	//s.state.updateStateObject(obj2)
	s.state.Commit(true)
	result, _ := s.state.trie.TryGet(obj1.address.Bytes())
	data2 = new(RegularAccount)
	if err := rlp.DecodeBytes(result, data2); err != nil {
	}
	fmt.Printf("trie.TryGe：%v \n", data2)

	regular := s.state.GetRegular(address)
	fmt.Printf("GetRegular：%v \n", regular)
}

func TestDeleteData(t *testing.T) {
	s := newStateNewTest()
	address := common.BytesToAddress([]byte{0x01})
	obj1 := s.state.GetOrNewStateObject(address)
	obj1.setValueForRegular(big.NewInt(20))
	fmt.Printf(" before DeleteStateObjectByAddr：%v \n", s.state.GetRegular(address))
	s.state.updateStateObject(obj1)
	s.state.DeleteStateObjectByAddr(address)
	fmt.Printf(" after DeleteStateObjectByAddr：%v \n", s.state.GetRegular(address))
}
