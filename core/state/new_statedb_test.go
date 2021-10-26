package state

import (
	"fmt"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/core/rawdb"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/rlp"
	"math/big"
	"testing"
)

type stateNewTest struct {
	db    probedb.Database
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
	//s.state.SetValueForRegular(address, big.NewInt(20))
	obj1 := s.state.GetRegular(address)
	//regular := s.state.GetValueForRegular(address)
	fmt.Printf(" after GetRegular：%v \n", obj1)
	//fmt.Printf(" after GetValueForRegular：%v \n", regular)
}

func TestSetData(t *testing.T) {
	s := newStateNewTest()
	address := common.BytesToAddress([]byte{0x01})
	//s.state.SetVoteValueForRegular(address, big.NewInt(10))
	fmt.Printf(" after SetValueForRegular：%v \n", s.state.GetRegular(address))
}

func TestTrieAndRlp(t *testing.T) {
	s := newStateNewTest()
	//address := common.BytesToAddress([]byte{0x01})
	address := common.HexToAddress("0x0085c9ef121fbdcb1bf8d0a7c606c363c0b3f172068cc3507b")
	//obj1 := s.state.GetOrNewStateObject(address)
	//fmt.Printf(" before GetOrNewStateObject：%v \n", obj1.regularAccount)
	////obj1.setValueForRegular(big.NewInt(20))
	//obj1.regularAccount.Value = big.NewInt(20)
	//s.state.SetValueForRegular(address, big.NewInt(210))
	//root := s.state.IntermediateRoot(false)
	//fmt.Printf("trie.TryGe root：%v \n", root)
	var data2 *RegularAccount
	// write some of them to the trie
	//s.state.updateStateObject(obj1)
	//s.state.updateStateObject(obj2)
	s.state.Commit(false)
	//s.state.Database().TrieDB().Commit(root, true, nil)
	result, _ := s.state.trie.TryGet(address.Bytes())
	data2 = new(RegularAccount)
	if err := rlp.DecodeBytes(result, data2); err != nil {
	}
	fmt.Printf("trie.TryGe：%v \n", data2)

	regular := s.state.GetRegular(address)
	fmt.Printf("GetRegular：%v \n", regular)
	//addr, b := s.state.DecodeDataByAddr(common.HexToAddress("0x00C350afF0fDdbED23B29Cf559acb5aA68C2C4F0247f5D58Bc"), nil)
	//fmt.Printf("addr：%v \n; b：%v \n", addr, b)
}

func TestDeleteData(t *testing.T) {
	s := newStateNewTest()
	//address := common.BytesToAddress([]byte{0x01})
	address := common.BytesToAddress(common.Hex2Bytes("0x006F0452548E1607836D06C7B2Be28576a076698bF59e47760"))
	obj1 := s.state.GetOrNewStateObject(address)
	obj1.setValueForRegular(big.NewInt(20))
	fmt.Printf(" before DeleteStateObjectByAddr：%v \n", s.state.GetRegular(address))
	s.state.updateStateObject(obj1)
	//s.state.DeleteStateObjectByAddr(address)
	fmt.Printf(" after DeleteStateObjectByAddr：%v \n", s.state.GetRegular(address))
}

func TestArray(t *testing.T) {
	a := []int{0, 1, 2, 3, 4}
	//删除第i个元素
	i := 3
	a = append(a[:i], a[i+1:]...)
	a = append(a[:i], a[i+1:]...)
	fmt.Printf(" after 数组a：%v \n", a)
}
func TestRlp(t *testing.T) {
	s := newStateNewTest()
	//address := common.BytesToAddress([]byte{0x01})
	address := common.HexToAddress("0x0085c9ef121fbdcb1bf8d0a7c606c363c0b3f172068cc3507b")
	address1 := common.Hash{1}
	result, _ := s.state.trie.TryGet(address.Bytes())

	arrdata, err := rlp.EncodeToBytes([]common.Hash{common.Hash{}, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot})
	s.db.Put(address1.Bytes(), arrdata)
	if err != nil {
		log.Crit("Failed to EncodeToBytes", "err", err, result)
	}
	fmt.Printf("arrdata：%v \n", arrdata)
	var intarray []common.Hash
	//hash := rawdb.ReadRootHash(db.TrieDB().DiskDB(), root)
	data, _ := s.db.Get(address1.Bytes())
	rlp.DecodeBytes(data, &intarray)
	fmt.Printf("trie.TryGe：%v \n", intarray)
}

func TestRlp1(t *testing.T) {
	s := newStateNewTest()
	//address := common.BytesToAddress([]byte{0x01})
	address := common.HexToAddress("0x0085c9ef121fbdcb1bf8d0a7c606c363c0b3f172068cc3507b")
	address1 := common.Hash{1}
	result, _ := s.state.trie.TryGet(address.Bytes())
	fmt.Printf("result：%v \n", result)
	//arrdata, _ := rlp.EncodeToBytes([]common.Hash{common.Hash{}, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot})
	arrdata := []common.Hash{common.Hash{}, emptyRoot, emptyRoot, emptyRoot, emptyRoot, emptyRoot}
	fmt.Printf("arrdata：%v \n", arrdata)
	fmt.Printf("arrdata：%v \n", address1)
	//var b []byte
	//for _, d := range hash {
	//	b = append(b, d.Bytes()...)
	//}
	//rawdb.WriteRootHash(db, root, b)

	//rootHash := rawdb.ReadRootHash(db, root)
	//s.db.Put(address1.Bytes(), arrdata)
	//if err != nil {
	//	log.Crit("Failed to EncodeToBytes", "err", err,result)
	//}
	//fmt.Printf("arrdata：%v \n", arrdata)
	//var intarray []common.Hash
	////hash := rawdb.ReadRootHash(db.TrieDB().DiskDB(), root)
	//data, _ := s.db.Get(address1.Bytes())
	//rlp.DecodeBytes(data, &intarray)
	//fmt.Printf("trie.TryGe：%v \n", intarray)
}
