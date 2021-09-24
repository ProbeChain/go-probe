package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to          *common.Address
	from        common.Address
	owner       *common.Address
	beneficiary *common.Address
	loss        *common.Address
	asset       *common.Address
	old         *common.Address
	new         *common.Address
	initiator   *common.Address
	receiver    *common.Address

	bizType    uint8
	nonce      uint64
	amount     *big.Int
	amount2    *big.Int
	height     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	gasFeeCap  *big.Int
	gasTipCap  *big.Int
	data       []byte
	mark       []byte
	infoDigest []byte
	accessList AccessList
	checkNonce bool
	accType    *hexutil.Uint8
	lossType   *hexutil.Uint8
	pnsType    *hexutil.Uint8
}

func (m Message) From() common.Address   { return m.from }
func (m Message) To() *common.Address    { return m.to }
func (m Message) GasPrice() *big.Int     { return m.gasPrice }
func (m Message) GasFeeCap() *big.Int    { return m.gasFeeCap }
func (m Message) GasTipCap() *big.Int    { return m.gasTipCap }
func (m Message) Value() *big.Int        { return m.amount }
func (m Message) Gas() uint64            { return m.gasLimit }
func (m Message) Nonce() uint64          { return m.nonce }
func (m Message) Data() []byte           { return m.data }
func (m Message) AccessList() AccessList { return m.accessList }
func (m Message) CheckNonce() bool       { return m.checkNonce }
func (m Message) BizType() uint8         { return m.bizType }

func (m Message) Owner() *common.Address       { return m.owner }
func (m Message) Beneficiary() *common.Address { return m.beneficiary }
func (m Message) Loss() *common.Address        { return m.loss }
func (m Message) Asset() *common.Address       { return m.asset }
func (m Message) Old() *common.Address         { return m.old }
func (m Message) New() *common.Address         { return m.new }
func (m Message) Initiator() *common.Address   { return m.initiator }
func (m Message) Receiver() *common.Address    { return m.receiver }
func (m Message) Value2() *big.Int             { return m.amount2 }
func (m Message) Height() *big.Int             { return m.height }
func (m Message) Mark() []byte                 { return m.mark }
func (m Message) InfoDigest() []byte           { return m.infoDigest }
func (m Message) AccType() *hexutil.Uint8      { return m.accType }
func (m Message) LossType() *hexutil.Uint8     { return m.lossType }
func (m Message) PnsType() *hexutil.Uint8      { return m.pnsType }
