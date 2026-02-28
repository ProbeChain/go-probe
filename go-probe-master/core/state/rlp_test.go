package state

import (
	"fmt"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/rlp"
	"math/big"
	"os"
	"testing"
)

func TestRlpDataStruct(t *testing.T) {
	encodeRegular(1)

	encodeInterface(1)
}

type RegularAccountTest struct {
	Type        byte
	VoteAccount common.Address
	VoteValue   *big.Int
	LossType    uint8
	Value       *big.Int
	Nonce       uint64
}

func encodeRegular(rType byte) {
	// define data struct
	data := RegularAccountTest{
		Type:        rType,
		VoteAccount: common.Address{2},
		VoteValue:   big.NewInt(3),
		LossType:    uint8(4),
		Nonce:       uint64(5),
		Value:       big.NewInt(6),
	}

	// encode data
	b, err := rlp.EncodeToBytes(data)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)
	}
	head, err := rlp.ParseTypeByHead(b)
	if err != nil {
		return
	}
	end, err := rlp.ParseTypeByEnd(b)
	if err != nil {
		return
	}
	fmt.Println("encodeRegular RLP编码输出：\n", common.Bytes2Hex(b))
	fmt.Println("encodeRegular RLP编码输出 ParseTypeByHead 值：\n", head)
	fmt.Println("encodeRegular RLP编码输出 ParseTypeByEnd 值：\n", end)
}

func encodeInterface(pType byte) {
	items := []interface{}{
		pType,
		common.Address{2},
		big.NewInt(3),
		uint8(4),
		big.NewInt(6),
		uint64(5),
	}

	b, err := rlp.EncodeToBytes(items)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	head, err := rlp.ParseTypeByHead(b)
	if err != nil {
		return
	}
	end, err := rlp.ParseTypeByEnd(b)
	if err != nil {
		return
	}
	fmt.Println("encodeInterface RLP编码输出：\n", common.Bytes2Hex(b))
	fmt.Println("encodeInterface RLP编码输出 ParseTypeByHead 值：\n", head)
	fmt.Println("encodeInterface RLP编码输出 ParseTypeByEnd 值：\n", end)

	for i, v := range items {
		b, err := rlp.EncodeToBytes(v)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("items[%d]=RLP(%v)=%s\n", i, v, common.Bytes2Hex(b))
	}
}

func encodePns(pType byte) {
	// define data struct
	data := PnsAccount{
		Type:  pType,
		Owner: common.Address{1},
		Data:  []byte{1},
	}

	// encode data
	b, err := rlp.EncodeToBytes(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	head, err := rlp.ParseTypeByHead(b)
	if err != nil {
		return
	}
	fmt.Println("encodePns RLP编码输出：\n", common.Bytes2Hex(b))
	fmt.Println("encodePns RLP编码输出 pType 值：\n", head)
}
