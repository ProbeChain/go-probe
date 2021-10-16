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

// Package types contains data types related to Probeum consensus.
package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/crypto/probecrypto"
	"github.com/probeum/go-probeum/crypto/secp256k1"
	"io"
	"math/big"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/rlp"
)

var (
	EmptyRootHash           = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyUncleHash          = rlpHash([]*Header(nil))
	EmptyPowAnswerUncleHash = rlpHash([]*PowAnswer(nil))
	EmptyDposAckHash        = rlpHash([]*DposAck(nil))
)

type DposAckType uint8

const (
	AckTypeAgree  DposAckType = 0
	AckTypeOppose DposAckType = 1
	AckTypeAll    DposAckType = 255
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

//send from pow miner
type PowAnswer struct {
	Number    *big.Int       `json:"number"           gencodec:"required"`
	MixDigest common.Hash    `json:"mixHash"          gencodec:"required"`
	Nonce     BlockNonce     `json:"nonce"            gencodec:"required"`
	Miner     common.Address `json:"miner"            gencodec:"required"`
}

// Id returns the pow answer unique id
func (powAnswer *PowAnswer) Id() common.Hash {
	// We assume that the miners will only give one answer in a given block number
	id := append(append(powAnswer.Miner.Bytes(), powAnswer.Number.Bytes()...), []byte{0, 0, 0, 0}...)
	return common.BytesToHash(id)
}

//send from dpos witness node
type DposAck struct {
	EpochPosition uint8       `json:"epochPosition"   gencodec:"required"`
	Number        *big.Int    `json:"number"          gencodec:"required"`
	BlockHash     common.Hash `json:"blockHash"       gencodec:"required"`
	AckType       DposAckType `json:"ackType"         gencodec:"required"`
	WitnessSig    []byte      `json:"witnessSig"      gencodec:"required"`
}

// Id returns the pow answer unique id
func (dposAck *DposAck) Id() common.Hash {
	// We assume that the miners will only give one answer in a given block number
	return common.BytesToHash(dposAck.WitnessSig)
}

type DposAckCount struct {
	BlockNumber *big.Int `json:"blockNumber"        gencodec:"required"`
	AckCount    uint     `json:"ackCount"           gencodec:"required"`
}

// Hash returns the dpos ack Keccak256
func (dposAck *DposAck) Hash() []byte {
	b := new(bytes.Buffer)
	enc := []interface{}{
		dposAck.EpochPosition,
		dposAck.Number,
		dposAck.BlockHash,
		dposAck.AckType,
	}
	if err := rlp.Encode(b, enc); err != nil {
		panic("can't encode: " + err.Error())
	}

	return crypto.Keccak256(b.Bytes())
}

// RecoverOwner returns the dpos ack pubkey
func (dposAck *DposAck) RecoverOwner() (common.Address, error) {
	pubkey, err := secp256k1.RecoverPubkey(dposAck.Hash(), dposAck.WitnessSig)
	if err == nil {
		publicKey, err := probecrypto.UnmarshalPubkey(pubkey)
		if err == nil {
			return probecrypto.PubkeyToAddress(*publicKey), nil
		}
	}
	return common.Address{}, err
}

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

// Header represents a block header in the Probeum blockchain.
type Header struct {
	DposSigAddr common.Address `json:"dposMiner"        gencodec:"required"`
	DposSig     []byte         `json:"dposSig"          gencodec:"required"`
	//BlockHash        common.Hash     `json:"blockHash"        gencodec:"required"`
	DposAckCountList []*DposAckCount `json:"dposAckCountList" gencodec:"required"`
	DposAcksHash     common.Hash     `json:"dposAcksHash"     gencodec:"required"`
	PowAnswers       []*PowAnswer    `json:"powAnswers"       gencodec:"required"`
	ParentHash       common.Hash     `json:"parentHash"       gencodec:"required"`
	UncleHash        common.Hash     `json:"sha3Uncles"       gencodec:"required"`
	Coinbase         common.Address  `json:"miner"            gencodec:"required"`
	Root             common.Hash     `json:"stateRoot"        gencodec:"required"`
	TxHash           common.Hash     `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash      common.Hash     `json:"receiptsRoot"     gencodec:"required"`
	Bloom            Bloom           `json:"logsBloom"        gencodec:"required"`
	Difficulty       *big.Int        `json:"difficulty"       gencodec:"required"`
	Number           *big.Int        `json:"number"           gencodec:"required"`
	GasLimit         uint64          `json:"gasLimit"         gencodec:"required"`
	GasUsed          uint64          `json:"gasUsed"          gencodec:"required"`
	Time             uint64          `json:"timestamp"        gencodec:"required"`
	Extra            []byte          `json:"extraData"        gencodec:"required"`
	MixDigest        common.Hash     `json:"mixHash"`
	Nonce            BlockNonce      `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`
}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty *hexutil.Big
	Number     *hexutil.Big
	GasLimit   hexutil.Uint64
	GasUsed    hexutil.Uint64
	Time       hexutil.Uint64
	Extra      hexutil.Bytes
	BaseFee    *hexutil.Big
	Hash       common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

var headerSize = common.StorageSize(reflect.TypeOf(Header{}).Size())

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	return headerSize + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen())/8)
}

// SanityCheck checks a few basic things -- these checks are way beyond what
// any 'sane' production values should hold, and can mainly be used to prevent
// that the unbounded fields are stuffed with junk data to add processing
// overhead
func (h *Header) SanityCheck() error {
	if h.Number != nil && !h.Number.IsUint64() {
		return fmt.Errorf("too large block number: bitlen %d", h.Number.BitLen())
	}
	if h.Difficulty != nil {
		if diffLen := h.Difficulty.BitLen(); diffLen > 80 {
			return fmt.Errorf("too large block difficulty: bitlen %d", diffLen)
		}
	}
	if eLen := len(h.Extra); eLen > 100*1024 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	return nil
}

// EmptyBody returns true if there is no additional 'body' to complete the header
// that is: no transactions and no uncles.
func (h *Header) EmptyBody() bool {
	return h.TxHash == EmptyRootHash && h.UncleHash == EmptyUncleHash
}

// EmptyReceipts returns true if there are no receipts for this header/block.
func (h *Header) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyRootHash
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) togprobeer.
type Body struct {
	Transactions []*Transaction
	//todo:remove
	Uncles          []*Header
	PowAnswerUncles []*PowAnswer
	DposAcks        []*DposAck
}

// Block represents an entire block in the Probeum blockchain.
type Block struct {
	header          *Header
	uncles          []*Header
	transactions    Transactions
	powAnswerUncles []*PowAnswer
	dposAcks        []*DposAck

	// caches
	hash atomic.Value
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	td *big.Int

	// These fields are used by package probe to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// "external" block encoding. used for probe protocol, etc.
type extblock struct {
	Header          *Header
	Txs             []*Transaction
	Uncles          []*Header
	PowAnswerUncles []*PowAnswer
	DposAcks        []*DposAck
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header), td: new(big.Int)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcUncleHash(uncles)
		b.uncles = make([]*Header, len(uncles))
		for i := range uncles {
			b.uncles[i] = CopyHeader(uncles[i])
		}
	}

	return b
}

func DposNewBlock(header *Header, txs []*Transaction, powAnswerUncles []*PowAnswer, dposAcks []*DposAck, receipts []*Receipt, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header), td: new(big.Int)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(powAnswerUncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcPowAnswerUncleHash(powAnswerUncles)
		b.powAnswerUncles = make([]*PowAnswer, len(powAnswerUncles))
		copy(b.powAnswerUncles, powAnswerUncles)
	}

	if len(dposAcks) == 0 {
		b.header.DposAcksHash = EmptyDposAckHash
	} else {
		b.header.DposAcksHash = CalcDposAckHash(dposAcks)
		b.dposAcks = make([]*DposAck, len(dposAcks))
		copy(b.dposAcks, dposAcks)
	}

	//TODO：remove
	//b.uncles = nil;

	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
}

// DecodeRLP decodes the Probeum
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions, b.powAnswerUncles, b.dposAcks = eb.Header, eb.Uncles, eb.Txs, eb.PowAnswerUncles, eb.DposAcks
	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

// EncodeRLP serializes b into the Probeum RLP block format.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header:          b.header,
		Txs:             b.transactions,
		Uncles:          b.uncles,
		PowAnswerUncles: b.powAnswerUncles,
		DposAcks:        b.dposAcks,
	})
}

// TODO: copies

func (b *Block) Uncles() []*Header             { return b.uncles }
func (b *Block) Transactions() Transactions    { return b.transactions }
func (b *Block) PowAnswerUncles() []*PowAnswer { return b.powAnswerUncles }
func (b *Block) PowAnswers() []*PowAnswer      { return b.header.PowAnswers }
func (b *Block) DposAcks() []*DposAck          { return b.dposAcks }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64     { return b.header.GasLimit }
func (b *Block) GasUsed() uint64      { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() uint64         { return b.header.Time }

func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) SetDposSig(dposSig []byte) bool {
	b.header.DposSig = append(b.header.DposSig, dposSig...)
	return true
}

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) Header() *Header { return CopyHeader(b.header) }

// Body returns the non-header content of the block.
func (b *Block) Body() *Body { return &Body{b.transactions, b.uncles, b.powAnswerUncles, b.dposAcks} }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previsouly cached value.
func (b *Block) Size() common.StorageSize {
	if size := b.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// SanityCheck can be used to prevent that unbounded fields are
// stuffed with junk data to add processing overhead
func (b *Block) SanityCheck() error {
	return b.header.SanityCheck()
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) common.Hash {
	if len(uncles) == 0 {
		return EmptyUncleHash
	}
	return rlpHash(uncles)
}

func CalcPowAnswerUncleHash(powAnswerUncles []*PowAnswer) common.Hash {
	if len(powAnswerUncles) == 0 {
		return EmptyPowAnswerUncleHash
	}
	return rlpHash(powAnswerUncles)
}

func CalcDposAckHash(dposAcks []*DposAck) common.Hash {
	if len(dposAcks) == 0 {
		return EmptyDposAckHash
	}
	return rlpHash(dposAcks)
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		uncles:       b.uncles,
	}
}

func (b *Block) DposWithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		uncles:       b.uncles,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(transactions []*Transaction, uncles []*Header) *Block {
	block := &Block{
		header:       CopyHeader(b.header),
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*Header, len(uncles)),
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = CopyHeader(uncles[i])
	}
	return block
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBodyGreatri(transactions []*Transaction, uncles []*Header, powAnswerUncles []*PowAnswer, dposAcks []*DposAck) *Block {
	block := &Block{
		header:          CopyHeader(b.header),
		transactions:    make([]*Transaction, len(transactions)),
		uncles:          make([]*Header, len(uncles)),
		powAnswerUncles: make([]*PowAnswer, len(powAnswerUncles)),
		dposAcks:        make([]*DposAck, len(dposAcks)),
	}
	copy(block.transactions, transactions)
	copy(block.powAnswerUncles, powAnswerUncles)
	copy(block.dposAcks, dposAcks)
	for i := range uncles {
		block.uncles[i] = CopyHeader(uncles[i])
	}
	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type Blocks []*Block
